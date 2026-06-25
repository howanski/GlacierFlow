package main

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

// ---- File paths ----

const (
	configTargetPath = "shared/inference_config.yml"
	lockFilePath     = "shared/INFERENCE_LOCK"
	modelFilePath    = "shared/inference_preset_name.txt"
	modelsDir        = "shared/inference_presets/local"
	nameTargetPath   = "shared/inference_preset_name_target.txt"
	statusFilePath   = "shared/inference_status.txt"
)

// ---- Helpers ----

func isLocked() bool {
	_, err := os.Stat(lockFilePath)
	return err == nil
}

func setLockFile() error {
	_, err := os.Stat(lockFilePath)
	if err == nil {
		return nil // already exists
	}
	return os.WriteFile(lockFilePath, nil, 0o644)
}

func removeLockFile() error {
	return os.Remove(lockFilePath)
}

func readLineFile(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func getStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":        readLineFile(statusFilePath),
		"locked":        isLocked(),
		"current_model": readLineFile(modelFilePath),
	})
}

func setLock(c *gin.Context) {
	lockPath := c.FullPath()
	var err error
	if lockPath == "/set-lock/on" {
		err = setLockFile()
	} else {
		err = removeLockFile()
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("lock error: %v", err)})
		return
	}
	c.JSON(http.StatusOK, gin.H{"locked": isLocked()})
}

func getModels(c *gin.Context) {
	entries, err := os.ReadDir(modelsDir)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"models": []gin.H{}})
		return
	}

	modelList := make([]gin.H, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yml") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".yml")
		source := filepath.Join(modelsDir, e.Name())
		content, _ := os.ReadFile(source)
		modelList = append(modelList, gin.H{"name": name, "content": string(content)})
	}

	c.JSON(http.StatusOK, gin.H{"models": modelList})
}

var validModelName = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func loadModel(c *gin.Context) {
	var body struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if !validModelName.MatchString(body.Name) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid model name"})
		return
	}

	source := filepath.Join(modelsDir, body.Name+".yml")
	data, err := os.ReadFile(source)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "model not found: " + body.Name})
		return
	}

	if err := os.WriteFile(configTargetPath, data, 0o644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("write config: %v", err)})
		return
	}

	if err := os.WriteFile(nameTargetPath, []byte(body.Name), 0o644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("write name: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"loaded": body.Name})
}

// ---- Main ----

func main() {
	r := gin.Default()

	r.LoadHTMLGlob("front/html/*")
	r.Static("/css", "front/css")
	r.Static("/js", "front/js")
	r.Static("/favicon", "front/files/favicon")

	r.GET("/", func(c *gin.Context) {
		port := os.Getenv("GF_PI_DEV_TTYD_PORT")
		if port == "" {
			port = "7681"
		}
		host, _, err := net.SplitHostPort(c.Request.Host)
		if err != nil {
			host = c.Request.Host
		}
		devHttpdUrl := fmt.Sprintf("http://%s:%s/", host, port)
		c.HTML(
			http.StatusOK,
			"index.html",
			gin.H{
				"title":       "GlacierFlow",
				"devHttpdUrl": devHttpdUrl,
			},
		)
	})

	r.GET("/tools/stats-viewer", func(c *gin.Context) {
		c.HTML(
			http.StatusOK,
			"stats_view.html",
			nil,
		)
	})

	r.GET("/status", getStatus)
	r.POST("/set-lock/on", setLock)
	r.POST("/set-lock/off", setLock)
	r.GET("/models", getModels)
	r.POST("/load-model", loadModel)

	r.Run()
}
