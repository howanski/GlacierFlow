package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	modeLLM = "llm"
	modeSD  = "sd"

	statusFile       = "/glacierflow-shared/inference_status.txt"
	presetNameFile   = "/glacierflow-shared/inference_preset_name.txt"
	presetTargetFile = "/glacierflow-shared/inference_preset_name_target.txt"
	inferenceConfig  = "/glacierflow-shared/inference_config.yml"
	presetsLocalDir  = "/glacierflow-shared/inference_presets/local/"
)

var (
	targetModelLlm = os.Getenv("GF_DEFAULT_MODEL_LLM")
	targetModelSd  = os.Getenv("GF_DEFAULT_MODEL_SD")
)

// fifoQueue serializes HTTP requests to a backend, processing them strictly
// in the order they arrive. A single worker goroutine consumes an ordered
// channel, so arrival order is preserved. Requests whose client disconnected
// while waiting in the queue are skipped without occupying the backend.
type fifoQueue struct {
	jobs    chan fifoJob
	backend *url.URL
	client  *http.Client
	mode    string // current operating mode (modeLLM or modeSD)
}

type fifoJob struct {
	ctx  context.Context
	r    *http.Request
	w    http.ResponseWriter
	done chan struct{}
	mode string
}

func newFifoQueue(backendURL string) *fifoQueue {
	u, err := url.Parse(backendURL)
	if err != nil {
		log.Fatalf("invalid backend URL %q: %v", backendURL, err)
	}
	q := &fifoQueue{
		jobs:    make(chan fifoJob),
		backend: u,
		client: &http.Client{
			Timeout: 0, // no timeout – streaming can last a long time
		},
		mode: modeLLM, // backend is assumed to start in LLM mode
	}
	go q.worker()
	return q
}

// worker processes queued requests one at a time, in arrival order.
func (q *fifoQueue) worker() {
	for job := range q.jobs {
		if job.ctx.Err() != nil {
			log.Printf("skipping queued %s %s: client disconnected while waiting", job.r.Method, job.r.URL.Path)
		} else {
			q.maybeSwitchMode(job.mode)
			q.dispatch(job.r, job.w)
		}
		close(job.done)
	}
}

// enqueue schedules a request for FIFO processing and blocks until it has
// been processed (or skipped because the client went away). A client that
// disconnects while waiting no longer holds its slot: the request is
// dropped as soon as it reaches the head of the queue.
func (q *fifoQueue) enqueue(ctx context.Context, mode string, r *http.Request, w http.ResponseWriter) {
	if ctx.Err() != nil {
		return
	}
	job := fifoJob{ctx: ctx, r: r, w: w, done: make(chan struct{}), mode: mode}
	q.jobs <- job
	<-job.done
}

// maybeSwitchMode switches the queue's operating mode to reqMode when it
// differs from the current mode and auto mode switching is enabled. It runs
// inside the worker, right before the request is proxied to the backend.
func (q *fifoQueue) maybeSwitchMode(reqMode string) {
	if q.mode == reqMode {
		return
	}
	if os.Getenv("GF_PROXY_AUTO_MODE") != "1" {
		return
	}
	log.Printf("mode switch requested: %s -> %s", q.mode, reqMode)
	q.switchMode(q.mode, reqMode)
}

// switchMode performs the physical mode transition from `from` to `to` by
// coordinating with the backend via the shared status/preset files.
func (q *fifoQueue) switchMode(from, to string) {
	// 1. Wait until the backend is idle (RUNNING).
	waitForStatus("RUNNING")

	// 2. Remember the model currently loaded so we can restore it later.
	current := readPresetName()
	if from == modeLLM {
		targetModelLlm = current
	} else {
		targetModelSd = current
	}

	// 3. Determine the model to load for the target mode.
	var target string
	if to == modeSD {
		target = targetModelSd
	} else {
		target = targetModelLlm
	}
	if target == "" {
		log.Printf("target model for %s mode is empty, skipping switch", to)
		return
	}

	// 4. Ask the backend to load the target model and wait for it to be ready.
	writeFile(presetTargetFile, target)
	presetSrc := presetsLocalDir + target + ".yml"
	if err := copyFile(presetSrc, inferenceConfig); err != nil {
		log.Printf("copy preset %s -> %s error: %v, skipping switch", presetSrc, inferenceConfig, err)
		return
	}
	writeFile(statusFile, "AUTO_SWITCHING")
	waitForStatus("RUNNING")

	q.mode = to
	log.Printf("mode switched to %s (model: %s)", to, target)
}

// waitForStatus polls the shared status file until its body equals want.
// There is no time cap.
func waitForStatus(want string) {
	for {
		b, err := os.ReadFile(statusFile)
		if err == nil && strings.TrimSpace(string(b)) == want {
			return
		}
		time.Sleep(300 * time.Millisecond)
	}
}

// readPresetName returns the current model name from the shared preset file.
func readPresetName() string {
	b, err := os.ReadFile(presetNameFile)
	if err != nil {
		log.Printf("read preset name error: %v", err)
		return ""
	}
	return strings.TrimSpace(string(b))
}

// writeFile writes content to path, logging (not failing) on error.
func writeFile(path, content string) {
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		log.Printf("write %s error: %v", path, err)
	}
}

// copyFile copies the contents of src to dst.
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

// dispatch forwards the request to the backend, copies the response back,
// and handles streaming (SSE) correctly.
func (q *fifoQueue) dispatch(r *http.Request, w http.ResponseWriter) {
	backendReq, requestBody, err := buildBackendRequest(q.backend, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp, err := q.client.Do(backendReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy all headers from the backend response.
	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header()[k] = append(w.Header()[k], v)
		}
	}
	w.WriteHeader(resp.StatusCode)

	// Flush so the client gets the status/headers immediately.
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	// Set up response body capture if DUMP_REQUESTS is enabled.
	var responseCapture *bytes.Buffer
	var bodyReader io.Reader
	if os.Getenv("DUMP_REQUESTS") == "1" {
		responseCapture = &bytes.Buffer{}
		bodyReader = io.TeeReader(resp.Body, responseCapture)
	} else {
		bodyReader = resp.Body
	}

	// Copy body – flush after each chunk so SSE streams are snappy.
	buf := make([]byte, 1024)
	f, flushOK := w.(http.Flusher)
	ctx := r.Context()
	for {
		// Stop reading from the backend as soon as the client disconnects.
		select {
		case <-ctx.Done():
			log.Printf("client disconnected: %v", ctx.Err())
			if responseCapture != nil {
				dumpRequestResponse(r.Method, r.URL.Path, requestBody, responseCapture.Bytes())
			}
			return // resp.Body.Close() fires via defer
		default:
		}
		n, readErr := bodyReader.Read(buf)
		if n > 0 {
			if _, writeErr := w.Write(buf[:n]); writeErr != nil {
				log.Printf("write error: %v", writeErr)
				break
			}
			if flushOK {
				f.Flush()
			}
		}
		if readErr != nil {
			if readErr != io.EOF {
				log.Printf("copy error: %v", readErr)
			}
			break
		}
	}

	// Dump request/response pair if DUMP_REQUESTS is enabled.
	if responseCapture != nil {
		dumpRequestResponse(r.Method, r.URL.Path, requestBody, responseCapture.Bytes())
	}
}

func buildBackendRequest(backend *url.URL, r *http.Request) (*http.Request, []byte, error) {
	// Clone the request so we can change its target.
	backendReq := r.Clone(r.Context())
	backendReq.RequestURI = "" // must be empty for client requests
	backendReq.URL = &url.URL{
		Scheme:   backend.Scheme,
		Host:     backend.Host,
		Path:     r.URL.Path,
		RawQuery: r.URL.RawQuery,
	}

	var requestBody []byte
	// If the original request carried a body, re-create it for the backend call.
	if r.Body != nil {
		buf, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, nil, fmt.Errorf("read request body: %w", err)
		}
		requestBody = buf
		r.Body = io.NopCloser(bytes.NewReader(buf)) // restore for caller
		backendReq.Body = io.NopCloser(bytes.NewReader(buf))
	}

	// Strip hop-by-hop headers that shouldn't be forwarded.
	backendReq.Header.Del("Connection")
	backendReq.Header.Del("Proxy-Connection")
	backendReq.Header.Del("Keep-Alive")
	backendReq.Header.Del("TE")
	backendReq.Header.Del("Trailer")
	backendReq.Header.Del("Transfer-Encoding")
	backendReq.Header.Del("Upgrade")

	return backendReq, requestBody, nil
}

func dumpRequestResponse(requestMethod, requestPath string, requestBody, responseBody []byte) {
	timestamp := time.Now().Format("2006_01_02_15_04_05")
	prefix := fmt.Sprintf("dumps/%s", timestamp)

	if err := os.MkdirAll("dumps", 0755); err != nil {
		log.Printf("dump mkdir error: %v", err)
		return
	}

	// Request file contains the exact request body, unmodified.
	if err := os.WriteFile(prefix+"_request.json", requestBody, 0644); err != nil {
		log.Printf("dump write error: %v", err)
		return
	}

	// Response file contains the full response body: pretty-printed when it
	// is valid JSON, raw bytes otherwise (e.g. SSE event streams).
	if err := os.WriteFile(prefix+"_response.json", formatBody(responseBody), 0644); err != nil {
		log.Printf("dump write error: %v", err)
		return
	}

	details, err := json.MarshalIndent(map[string]string{
		"requestMethod": requestMethod,
		"requestPath":   requestPath,
	}, "", "  ")
	if err != nil {
		log.Printf("dump marshal error: %v", err)
		return
	}

	if err := os.WriteFile(prefix+"_details.json", details, 0644); err != nil {
		log.Printf("dump write error: %v", err)
		return
	}

	log.Printf("dumped request/response to %s_request.json, %s_response.json, %s_details.json", prefix, prefix, prefix)
}

// formatBody returns the body pretty-printed when it is valid JSON,
// otherwise the original bytes unchanged.
func formatBody(body []byte) []byte {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 || !json.Valid(trimmed) {
		return body
	}
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, trimmed, "", "  "); err != nil {
		return body
	}
	return pretty.Bytes()
}

func main() {
	backendAddr := os.Getenv("BACKEND_URL")
	if backendAddr == "" {
		backendAddr = "http://glacierflow-llamacpp:8080"
		log.Printf("BACKEND_URL not set, defaulting to %s", backendAddr)
	}

	fifo := newFifoQueue(backendAddr)

	// Reverse-proxy for non-/v1 routes (parallel).
	backendURL, _ := url.Parse(backendAddr)
	parallelProxy := httputil.NewSingleHostReverseProxy(backendURL)

	listenAddr := ":8080"
	mux := http.NewServeMux()

	// docker health check endpoint
	mux.HandleFunc("/proxy-health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/v1/chat/") {
			log.Printf("[F.I.F.O. LLM] %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
			fifo.enqueue(r.Context(), modeLLM, r, w)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/v1/images/") {
			log.Printf("[F.I.F.O. SD] %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
			fifo.enqueue(r.Context(), modeSD, r, w)
			return
		}
		log.Printf("[PARALLEL] %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
		parallelProxy.ServeHTTP(w, r)
	})

	log.Printf("Proxy listening on %s  ->  backend: %s", listenAddr, backendAddr)
	if err := http.ListenAndServe(listenAddr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
