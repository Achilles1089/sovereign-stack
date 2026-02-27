package ai

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client manages communication with the Ollama API
type Client struct {
	Host       string
	HTTPClient *http.Client
}

// NewClient creates an Ollama API client
func NewClient(host string) *Client {
	return &Client{
		Host: host,
		HTTPClient: &http.Client{
			Timeout: 0, // No timeout for streaming
		},
	}
}

// Model represents an installed Ollama model
type Model struct {
	Name       string    `json:"name"`
	Size       int64     `json:"size"`
	ModifiedAt time.Time `json:"modified_at"`
	Digest     string    `json:"digest"`
}

// ChatMessage represents a chat message
type ChatMessage struct {
	Role    string `json:"role"` // "system", "user", "assistant"
	Content string `json:"content"`
}

// IsRunning checks if Ollama is reachable
func (c *Client) IsRunning() bool {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(c.baseURL() + "/api/tags")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

// ListModels returns all installed models
func (c *Client) ListModels() ([]Model, error) {
	resp, err := c.HTTPClient.Get(c.baseURL() + "/api/tags")
	if err != nil {
		return nil, fmt.Errorf("cannot reach Ollama at %s: %w", c.Host, err)
	}
	defer resp.Body.Close()

	var result struct {
		Models []Model `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse models: %w", err)
	}

	return result.Models, nil
}

// PullModel downloads a model with progress streaming
func (c *Client) PullModel(name string, onProgress func(status string, completed, total int64)) error {
	payload := fmt.Sprintf(`{"name":"%s","stream":true}`, name)

	resp, err := c.HTTPClient.Post(c.baseURL()+"/api/pull", "application/json", strings.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to start pull: %w", err)
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	// Increase buffer for large responses
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		var progress struct {
			Status    string `json:"status"`
			Total     int64  `json:"total"`
			Completed int64  `json:"completed"`
			Error     string `json:"error"`
		}

		if err := json.Unmarshal(scanner.Bytes(), &progress); err != nil {
			continue
		}

		if progress.Error != "" {
			return fmt.Errorf("pull failed: %s", progress.Error)
		}

		if onProgress != nil {
			onProgress(progress.Status, progress.Completed, progress.Total)
		}
	}

	return scanner.Err()
}

// Chat sends a chat message and streams the response
func (c *Client) Chat(model string, messages []ChatMessage, onChunk func(content string, done bool)) error {
	// Build request
	reqBody := struct {
		Model    string        `json:"model"`
		Messages []ChatMessage `json:"messages"`
		Stream   bool          `json:"stream"`
	}{
		Model:    model,
		Messages: messages,
		Stream:   true,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.HTTPClient.Post(c.baseURL()+"/api/chat", "application/json", strings.NewReader(string(data)))
	if err != nil {
		return fmt.Errorf("failed to connect to Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Ollama returned status %d: %s", resp.StatusCode, string(body))
	}

	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("error reading stream: %w", err)
		}

		var chunk struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			Done  bool   `json:"done"`
			Error string `json:"error"`
		}

		if err := json.Unmarshal(line, &chunk); err != nil {
			continue
		}

		if chunk.Error != "" {
			return fmt.Errorf("chat error: %s", chunk.Error)
		}

		if onChunk != nil {
			onChunk(chunk.Message.Content, chunk.Done)
		}

		if chunk.Done {
			break
		}
	}

	return nil
}

// Generate sends a single prompt (non-chat) and streams the response
func (c *Client) Generate(model string, prompt string, onChunk func(response string, done bool)) error {
	payload := fmt.Sprintf(`{"model":"%s","prompt":"%s","stream":true}`,
		model, escapeJSONString(prompt))

	resp, err := c.HTTPClient.Post(c.baseURL()+"/api/generate", "application/json", strings.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to connect to Ollama: %w", err)
	}
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		var chunk struct {
			Response string `json:"response"`
			Done     bool   `json:"done"`
		}

		if err := json.Unmarshal(line, &chunk); err != nil {
			continue
		}

		if onChunk != nil {
			onChunk(chunk.Response, chunk.Done)
		}

		if chunk.Done {
			break
		}
	}

	return nil
}

func (c *Client) baseURL() string {
	host := c.Host
	if !strings.HasPrefix(host, "http") {
		host = "http://" + host
	}
	return host
}

func escapeJSONString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	return s
}
