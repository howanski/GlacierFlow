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
	"sync"
	"time"
)

// fifoQueue serializes HTTP requests to a backend, processing them strictly
// in the order they arrive.
type fifoQueue struct {
	mu      sync.Mutex
	cond    *sync.Cond
	running bool
	backend *url.URL
	client  *http.Client
}

func newFifoQueue(backendURL string) *fifoQueue {
	u, err := url.Parse(backendURL)
	if err != nil {
		log.Fatalf("invalid backend URL %q: %v", backendURL, err)
	}
	q := &fifoQueue{
		backend: u,
		client: &http.Client{
			Timeout: 0, // no timeout – streaming can last a long time
		},
	}
	q.cond = sync.NewCond(&q.mu)
	return q
}

// enqueue schedules a request for FIFO processing. It blocks until it is
// this request's turn, then dispatches it and signals the next waiter.
func (q *fifoQueue) enqueue(ctx context.Context, r *http.Request, w http.ResponseWriter) {
	q.mu.Lock()
	for q.running {
		q.cond.Wait()
	}
	q.running = true
	q.mu.Unlock()

	q.dispatch(r, w)

	q.mu.Lock()
	q.running = false
	q.cond.Broadcast()
	q.mu.Unlock()
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
	filename := fmt.Sprintf("dumps/%s.json", timestamp)

	data := map[string]interface{}{
		"requestMethod":  requestMethod,
		"requestPath":    requestPath,
		"request":        string(requestBody),
		"response":       string(responseBody),
	}

	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Printf("dump marshal error: %v", err)
		return
	}

	if err := os.MkdirAll("dumps", 0755); err != nil {
		log.Printf("dump mkdir error: %v", err)
		return
	}

	if err := os.WriteFile(filename, jsonBytes, 0644); err != nil {
		log.Printf("dump write error: %v", err)
		return
	}

	log.Printf("dumped request/response to %s", filename)
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
			log.Printf("[F.I.F.O.] %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
			fifo.enqueue(r.Context(), r, w)
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
