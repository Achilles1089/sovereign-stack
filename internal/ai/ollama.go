package ai

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Client manages communication with the native llama-server (OpenAI-compatible API)
type Client struct {
	Host        string
	ModelsDir   string // Directory containing GGUF model files
	ServerBin   string // Path to llama-server binary
	HTTPClient  *http.Client
	activeModel string
	mu          sync.Mutex
}

// NewClient creates a llama-server API client
func NewClient(host string) *Client {
	modelsDir := os.Getenv("SOVEREIGN_MODELS_DIR")
	if modelsDir == "" {
		modelsDir = "/data/data/com.termux/files/home/models"
	}
	serverBin := os.Getenv("SOVEREIGN_LLAMA_BIN")
	if serverBin == "" {
		serverBin = "/data/data/com.termux/files/home/llama.cpp/bld/bin/llama-server"
	}

	return &Client{
		Host:      host,
		ModelsDir: modelsDir,
		ServerBin: serverBin,
		HTTPClient: &http.Client{
			Timeout: 0, // No timeout for streaming
		},
	}
}

// Model represents an installed GGUF model
type Model struct {
	Name       string    `json:"name"`
	Size       int64     `json:"size"`
	ModifiedAt time.Time `json:"modified_at"`
	Digest     string    `json:"digest"`
	Active     bool      `json:"active"`
	Filename   string    `json:"filename"`
}

// ChatMessage represents a chat message
type ChatMessage struct {
	Role    string `json:"role"` // "system", "user", "assistant"
	Content string `json:"content"`
}

// IsRunning checks if llama-server is reachable
func (c *Client) IsRunning() bool {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(c.baseURL() + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

// ActiveModel returns the currently loaded model name
func (c *Client) ActiveModel() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.activeModel
}

// ListModels returns all installed GGUF models by scanning the models directory
func (c *Client) ListModels() ([]Model, error) {
	entries, err := os.ReadDir(c.ModelsDir)
	if err != nil {
		// If models dir doesn't exist, return empty list
		if os.IsNotExist(err) {
			return []Model{}, nil
		}
		return nil, fmt.Errorf("cannot read models directory %s: %w", c.ModelsDir, err)
	}

	var models []Model
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".gguf") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".gguf")
		models = append(models, Model{
			Name:       name,
			Filename:   entry.Name(),
			Size:       info.Size(),
			ModifiedAt: info.ModTime(),
			Digest:     fmt.Sprintf("gguf-%s", name),
			Active:     entry.Name() == c.ActiveModel(),
		})
	}

	return models, nil
}

// PullModel downloads a GGUF model from a URL with progress streaming
func (c *Client) PullModel(name string, onProgress func(status string, completed, total int64)) error {
	// Look up the model in the catalog to get its URL
	entry := GetModelByName(name)
	if entry == nil {
		return fmt.Errorf("model %q not found in catalog", name)
	}

	if entry.URL == "" {
		return fmt.Errorf("model %q has no download URL", name)
	}

	destPath := filepath.Join(c.ModelsDir, entry.Filename)

	// Create models directory if it doesn't exist
	if err := os.MkdirAll(c.ModelsDir, 0755); err != nil {
		return fmt.Errorf("failed to create models directory: %w", err)
	}

	if onProgress != nil {
		onProgress("starting download", 0, 0)
	}

	// Start HTTP download
	resp, err := http.Get(entry.URL)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	total := resp.ContentLength

	out, err := os.Create(destPath + ".part")
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	var completed int64
	buf := make([]byte, 256*1024) // 256KB chunks
	lastReport := time.Now()

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			_, writeErr := out.Write(buf[:n])
			if writeErr != nil {
				out.Close()
				os.Remove(destPath + ".part")
				return fmt.Errorf("write error: %w", writeErr)
			}
			completed += int64(n)

			// Report progress every 500ms
			if onProgress != nil && time.Since(lastReport) > 500*time.Millisecond {
				onProgress("downloading", completed, total)
				lastReport = time.Now()
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			out.Close()
			os.Remove(destPath + ".part")
			return fmt.Errorf("download error: %w", readErr)
		}
	}

	out.Close()

	// Rename .part to final filename
	if err := os.Rename(destPath+".part", destPath); err != nil {
		return fmt.Errorf("failed to finalize download: %w", err)
	}

	if onProgress != nil {
		onProgress("success", total, total)
	}

	return nil
}

// DeleteModel removes a GGUF model file from disk
func (c *Client) DeleteModel(name string) error {
	// Find the model file
	entry := GetModelByName(name)
	var filename string
	if entry != nil {
		filename = entry.Filename
	} else {
		// Try direct filename
		filename = name + ".gguf"
	}

	path := filepath.Join(c.ModelsDir, filename)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("model %q not found at %s", name, path)
	}

	return os.Remove(path)
}

// formatRWKVPrompt converts chat messages into RWKV's native User:/Assistant: format
func formatRWKVPrompt(messages []ChatMessage) string {
	var sb strings.Builder
	for _, msg := range messages {
		switch msg.Role {
		case "system":
			sb.WriteString(msg.Content)
			sb.WriteString("\n\n")
		case "user":
			sb.WriteString("User: ")
			sb.WriteString(msg.Content)
			sb.WriteString("\n\n")
		case "assistant":
			sb.WriteString("Assistant: ")
			sb.WriteString(msg.Content)
			sb.WriteString("\n\n")
		}
	}
	sb.WriteString("Assistant:")
	return sb.String()
}

// Chat sends a chat message using the /completion endpoint with RWKV prompt formatting
func (c *Client) Chat(model string, messages []ChatMessage, onChunk func(content string, done bool)) error {
	// Format messages into RWKV's native prompt format
	prompt := formatRWKVPrompt(messages)

	reqBody := struct {
		Prompt      string   `json:"prompt"`
		NPredict    int      `json:"n_predict"`
		Stream      bool     `json:"stream"`
		Stop        []string `json:"stop"`
		Temperature float64  `json:"temperature"`
	}{
		Prompt:      prompt,
		NPredict:    1024,
		Stream:      true,
		Stop:        []string{"User:", "User :", "\nUser"},
		Temperature: 0.7,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL()+"/completion", strings.NewReader(string(data)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Encoding", "identity")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to llama-server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("llama-server returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse streaming JSON lines from /completion endpoint
	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("error reading stream: %w", err)
		}

		lineStr := strings.TrimSpace(string(line))

		// Handle SSE "data: " prefix if present
		lineStr = strings.TrimPrefix(lineStr, "data: ")

		if lineStr == "" || lineStr == "[DONE]" {
			continue
		}

		var chunk struct {
			Content string `json:"content"`
			Stop    bool   `json:"stop"`
		}

		if err := json.Unmarshal([]byte(lineStr), &chunk); err != nil {
			continue
		}

		if onChunk != nil {
			onChunk(chunk.Content, chunk.Stop)
		}

		if chunk.Stop {
			break
		}
	}

	return nil
}

// Generate sends a single prompt and streams the response
func (c *Client) Generate(model string, prompt string, onChunk func(response string, done bool)) error {
	// Convert to chat format
	messages := []ChatMessage{
		{Role: "user", Content: prompt},
	}
	return c.Chat(model, messages, onChunk)
}

// SwitchModel restarts llama-server with a different model
// This is a no-op if running on brainnet (the phone manages its own server)
func (c *Client) SwitchModel(modelName string) error {
	entry := GetModelByName(modelName)
	var filename string
	if entry != nil {
		filename = entry.Filename
	} else {
		filename = modelName + ".gguf"
	}

	modelPath := filepath.Join(c.ModelsDir, filename)
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		return fmt.Errorf("model file not found: %s", modelPath)
	}

	// Kill existing llama-server
	exec.Command("pkill", "-f", "llama-server").Run()
	time.Sleep(1 * time.Second)

	// Start new llama-server with the model
	cmd := exec.Command(c.ServerBin,
		"-m", modelPath,
		"--host", "0.0.0.0",
		"--port", extractPort(c.Host),
		"-t", "8",
		"-c", "2048",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start llama-server: %w", err)
	}

	c.mu.Lock()
	c.activeModel = filename
	c.mu.Unlock()

	// Wait for server to be ready
	for i := 0; i < 30; i++ {
		time.Sleep(1 * time.Second)
		if c.IsRunning() {
			return nil
		}
	}

	return fmt.Errorf("llama-server did not start within 30 seconds")
}

func (c *Client) baseURL() string {
	host := c.Host
	if !strings.HasPrefix(host, "http") {
		host = "http://" + host
	}
	return host
}

func extractPort(host string) string {
	parts := strings.Split(host, ":")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return "8085"
}

func escapeJSONString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	return s
}
