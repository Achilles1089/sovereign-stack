package server

import (
	"encoding/json"
	"fmt"
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
	host := cfg.AI.OllamaHost
	if host == "" {
		host = "localhost:11434"
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
	mux.HandleFunc("/api/ai/models", s.handleAIModels)
	mux.HandleFunc("/api/ai/chat", s.handleAIChat)
	mux.HandleFunc("/api/ai/status", s.handleAIStatus)

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

	fmt.Printf("  🌐 Dashboard: http://%s\n", s.addr)
	fmt.Printf("  📡 API:       http://%s/api/\n", s.addr)
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

func (s *Server) handleAIModels(w http.ResponseWriter, r *http.Request) {
	models, err := s.client.ListModels()
	if err != nil {
		writeJSON(w, map[string]interface{}{"models": []interface{}{}, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"models": models})
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
		"mode":        s.cfg.AI.OllamaMode,
		"model":       s.cfg.AI.DefaultModel,
		"gpu_tier":    tierNames[tier],
		"recommended": hardware.RecommendedModel(&s.cfg.Hardware),
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

	// Stream the response
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Transfer-Encoding", "chunked")
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
