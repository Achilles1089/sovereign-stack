package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Achilles1089/sovereign-stack/internal/ai"
	"github.com/Achilles1089/sovereign-stack/internal/apps"
	"github.com/Achilles1089/sovereign-stack/internal/config"
	"github.com/Achilles1089/sovereign-stack/internal/docker"
	"github.com/Achilles1089/sovereign-stack/internal/hardware"
)

// Server is the Sovereign Stack API + dashboard server
type Server struct {
	cfg       *config.Config
	client    *ai.Client
	addr      string
	staticDir string
}

// New creates a new dashboard server
func New(cfg *config.Config, addr string) *Server {
	host := cfg.AI.Host
	if host == "" {
		host = "localhost:8085"
	}
	return &Server{
		cfg:    cfg,
		client: ai.NewClient(host),
		addr:   addr,
	}
}

// SetStaticDir sets the path to the built dashboard frontend
func (s *Server) SetStaticDir(dir string) {
	s.staticDir = dir
}

// Start begins serving
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/resources", s.handleResources)
	mux.HandleFunc("/api/apps", s.handleApps)
	mux.HandleFunc("/api/apps/install", s.handleAppInstall)
	mux.HandleFunc("/api/apps/remove", s.handleAppRemove)
	mux.HandleFunc("/api/ai/models", s.handleAIModels)
	mux.HandleFunc("/api/ai/catalog", s.handleAICatalog)
	mux.HandleFunc("/api/ai/chat", s.handleAIChat)
	mux.HandleFunc("/api/ai/server-chat", s.handleServerChat)
	mux.HandleFunc("/api/ai/status", s.handleAIStatus)
	mux.HandleFunc("/api/ai/pull", s.handleAIPull)
	mux.HandleFunc("/api/ai/delete", s.handleAIDelete)
	mux.HandleFunc("/api/ai/switch", s.handleAISwitch)
	mux.HandleFunc("/api/ai/phone-status", s.handlePhoneStatus)

	// Serve static dashboard files (SPA fallback)
	if s.staticDir != "" {
		mux.Handle("/", spaHandler(s.staticDir))
	} else {
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"name":    "Sovereign Stack API",
				"version": "0.1.0",
				"status":  "running",
			})
		})
	}

	fmt.Printf("  \ud83c\udf10 Dashboard: http://%s\n", s.addr)
	fmt.Printf("  \ud83d\udce1 API:       http://%s/api/\n", s.addr)
	return http.ListenAndServe(s.addr, corsMiddleware(mux))
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(204)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// --- Handlers ---

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	services, err := docker.CheckAllServices()
	if err != nil {
		writeJSON(w, map[string]interface{}{
			"services": []interface{}{},
			"error":    err.Error(),
		})
		return
	}
	writeJSON(w, map[string]interface{}{"services": services})
}

func (s *Server) handleResources(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]interface{}{
		"cpu_model":     s.cfg.Hardware.CPUModel,
		"cpu_cores":     s.cfg.Hardware.CPUCores,
		"ram_total_mb":  s.cfg.Hardware.RAMTotalMB,
		"disk_total_gb": s.cfg.Hardware.DiskTotalGB,
		"disk_free_gb":  s.cfg.Hardware.DiskFreeGB,
		"gpu_type":      s.cfg.Hardware.GPUType,
		"gpu_name":      s.cfg.Hardware.GPUName,
		"gpu_memory_mb": s.cfg.Hardware.GPUMemoryMB,
	})
}

func (s *Server) handleApps(w http.ResponseWriter, r *http.Request) {
	installed, _ := apps.InstalledApps()
	installedMap := make(map[string]bool)
	for _, name := range installed {
		installedMap[name] = true
	}

	type appResponse struct {
		Name        string `json:"name"`
		DisplayName string `json:"display_name"`
		Description string `json:"description"`
		Category    string `json:"category"`
		Version     string `json:"version"`
		Installed   bool   `json:"installed"`
	}

	var result []appResponse
	for _, app := range apps.BuiltinApps {
		result = append(result, appResponse{
			Name:        app.Name,
			DisplayName: app.DisplayName,
			Description: app.Description,
			Category:    app.Category,
			Version:     app.Version,
			Installed:   installedMap[app.Name],
		})
	}

	writeJSON(w, map[string]interface{}{"apps": result})
}

func (s *Server) handleAppInstall(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"error": "invalid request"})
		return
	}
	app := apps.FindApp(req.Name)
	if app == nil {
		writeJSON(w, map[string]interface{}{"error": "app not found"})
		return
	}
	if err := apps.InstallApp(app); err != nil {
		writeJSON(w, map[string]interface{}{"error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"ok": true, "message": fmt.Sprintf("%s installed successfully", app.DisplayName)})
}

func (s *Server) handleAppRemove(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"error": "invalid request"})
		return
	}
	if err := apps.RemoveApp(req.Name); err != nil {
		writeJSON(w, map[string]interface{}{"error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"ok": true, "message": fmt.Sprintf("%s removed successfully", req.Name)})
}

func (s *Server) handleAIModels(w http.ResponseWriter, r *http.Request) {
	models, err := s.client.ListModels()
	if err != nil {
		writeJSON(w, map[string]interface{}{"models": []interface{}{}, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"models": models})
}

func (s *Server) handleAICatalog(w http.ResponseWriter, r *http.Request) {
	// Return all available models from the catalog (not just installed)
	installed, _ := s.client.ListModels()
	installedMap := make(map[string]bool)
	for _, m := range installed {
		installedMap[m.Name] = true
	}

	type catalogEntry struct {
		ai.ModelEntry
		Installed bool `json:"installed"`
	}

	var catalog []catalogEntry
	for _, m := range ai.ModelCatalog {
		catalog = append(catalog, catalogEntry{
			ModelEntry: m,
			Installed:  installedMap[m.Name],
		})
	}
	writeJSON(w, map[string]interface{}{"catalog": catalog})
}

func (s *Server) handleAIStatus(w http.ResponseWriter, r *http.Request) {
	tier := hardware.GetGPUTier(&s.cfg.Hardware)
	tierNames := map[hardware.GPUTier]string{
		hardware.GPUTierNone:  "cpu",
		hardware.GPUTierBasic: "basic",
		hardware.GPUTierMid:   "mid",
		hardware.GPUTierHigh:  "high",
		hardware.GPUTierUltra: "ultra",
		hardware.GPUTierApex:  "apex",
	}
	writeJSON(w, map[string]interface{}{
		"running":     s.client.IsRunning(),
		"host":        s.client.Host,
		"mode":        "native",
		"model":       s.client.ActiveModel(),
		"gpu_tier":    tierNames[tier],
		"recommended": hardware.RecommendedModel(&s.cfg.Hardware),
		"engine":      "llama-server",
		"models_dir":  s.client.ModelsDir,
	})
}

func (s *Server) handleAIChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Model    string           `json:"model"`
		Messages []ai.ChatMessage `json:"messages"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Model == "" {
		req.Model = s.cfg.AI.DefaultModel
	}

	// Stream the response — anti-buffering headers are critical for Caddy/proxy
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)

	err := s.client.Chat(req.Model, req.Messages, func(content string, done bool) {
		fmt.Fprint(w, content)
		if ok {
			flusher.Flush()
		}
	})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleServerChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Message string `json:"message"`
		Model   string `json:"model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	model := req.Model
	if model == "" {
		model = s.cfg.AI.DefaultModel
	}

	// Build live server context
	ctx := ai.BuildServerContext(s.cfg)
	systemPrompt := ai.SystemPrompt(ctx)

	// Construct messages with system prompt
	messages := []ai.ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: req.Message},
	}

	// Stream the response — anti-buffering headers are critical for Caddy/proxy
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)

	err := s.client.Chat(model, messages, func(content string, done bool) {
		fmt.Fprint(w, content)
		if ok {
			flusher.Flush()
		}
	})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleAIPull(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Model string `json:"model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Stream progress back
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Transfer-Encoding", "chunked")
	flusher, ok := w.(http.Flusher)

	err := s.client.PullModel(req.Model, func(status string, completed, total int64) {
		if total > 0 {
			pct := float64(completed) / float64(total) * 100
			fmt.Fprintf(w, "%s: %.0f%%\n", status, pct)
		} else {
			fmt.Fprintf(w, "%s\n", status)
		}
		if ok {
			flusher.Flush()
		}
	})

	if err != nil {
		fmt.Fprintf(w, "ERROR: %s\n", err.Error())
		return
	}
	fmt.Fprintf(w, "DONE\n")
}

func (s *Server) handleAIDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Model string `json:"model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"error": "invalid request"})
		return
	}

	err := s.client.DeleteModel(req.Model)
	if err != nil {
		writeJSON(w, map[string]interface{}{"error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"ok": true, "message": fmt.Sprintf("%s deleted", req.Model)})
}

func (s *Server) handleAISwitch(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Model string `json:"model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"error": "invalid request"})
		return
	}

	err := s.client.SwitchModel(req.Model)
	if err != nil {
		writeJSON(w, map[string]interface{}{"error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"ok": true, "model": req.Model, "message": fmt.Sprintf("Switched to %s", req.Model)})
}

func (s *Server) handlePhoneStatus(w http.ResponseWriter, r *http.Request) {
	// Query llama-server's /v1/models endpoint to get loaded model info
	resp, err := http.Get("http://" + s.client.Host + "/v1/models")
	if err != nil {
		writeJSON(w, map[string]interface{}{
			"running":      false,
			"model":        "",
			"display_name": "",
			"params":       0,
			"vocab":        0,
			"context":      0,
			"size_bytes":   0,
			"engine":       "llama-server",
		})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// Parse the llama-server response
	var result struct {
		Data []struct {
			ID   string `json:"id"`
			Meta struct {
				NParams   int `json:"n_params"`
				NVocab    int `json:"n_vocab"`
				NCtxTrain int `json:"n_ctx_train"`
				Size      int `json:"size"`
			} `json:"meta"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil || len(result.Data) == 0 {
		writeJSON(w, map[string]interface{}{
			"running":      true,
			"model":        "unknown",
			"display_name": "Unknown",
			"params":       0,
			"vocab":        0,
			"context":      0,
			"size_bytes":   0,
			"engine":       "llama-server",
		})
		return
	}

	model := result.Data[0]
	// Derive display name from model ID: "rwkv7-2.9B-world-q4_k_m.gguf" -> "RWKV-7 2.9B"
	displayName := model.ID
	if len(model.ID) > 5 && model.ID[:5] == "rwkv7" {
		displayName = "RWKV-7 2.9B"
	}

	writeJSON(w, map[string]interface{}{
		"running":      true,
		"model":        model.ID,
		"display_name": displayName,
		"params":       model.Meta.NParams,
		"vocab":        model.Meta.NVocab,
		"context":      model.Meta.NCtxTrain,
		"size_bytes":   model.Meta.Size,
		"engine":       "llama-server",
	})
}

// spaHandler serves static files with SPA fallback to index.html
func spaHandler(staticDir string) http.Handler {
	fs := http.FileServer(http.Dir(staticDir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(staticDir, r.URL.Path)
		// Check if the file exists
		if _, err := os.Stat(path); os.IsNotExist(err) {
			// SPA fallback — serve index.html for non-API, non-file routes
			if !strings.HasPrefix(r.URL.Path, "/api/") {
				http.ServeFile(w, r, filepath.Join(staticDir, "index.html"))
				return
			}
			http.NotFound(w, r)
			return
		}
		fs.ServeHTTP(w, r)
	})
}
