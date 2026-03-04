package server

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

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
	mux.HandleFunc("/api/ai/phone-models", s.handlePhoneModels)
	mux.HandleFunc("/api/ai/phone-switch", s.handlePhoneSwitch)
	mux.HandleFunc("/api/ai/phone-start", s.handlePhoneStart)
	mux.HandleFunc("/api/ai/image-generate", s.handleImageGenerate)
	mux.HandleFunc("/api/ai/image-status", s.handleImageStatus)
	mux.HandleFunc("/api/ai/transcribe", s.handleTranscribe)
	mux.HandleFunc("/api/ai/speak", s.handleSpeak)
	mux.HandleFunc("/api/ai/voice-chat", s.handleVoiceChat)
	mux.HandleFunc("/api/ai/voice-status", s.handleVoiceStatus)
	mux.HandleFunc("/api/ai/music-generate", s.handleMusicGenerate)
	mux.HandleFunc("/api/ai/music-status", s.handleMusicStatus)
	mux.HandleFunc("/api/ai/rag-upload", s.handleRAGProxy)
	mux.HandleFunc("/api/ai/rag-search", s.handleRAGProxy)
	mux.HandleFunc("/api/ai/rag-documents", s.handleRAGProxy)
	mux.HandleFunc("/api/ai/rag-delete", s.handleRAGProxy)
	mux.HandleFunc("/api/ai/rag-status", s.handleRAGProxy)
	mux.HandleFunc("/api/gallery", s.handleGalleryList)
	mux.HandleFunc("/api/gallery/image/", s.handleGalleryImage)
	mux.HandleFunc("/api/gallery/delete/", s.handleGalleryDelete)
	mux.HandleFunc("/api/resources/live", s.handleResourcesLive)
	mux.HandleFunc("/api/envy/sysinfo", s.handleEnvySysinfo)

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

	fmt.Printf("  [WEB] Dashboard: http://%s\n", s.addr)
	fmt.Printf("  [API] API:       http://%s/api/\n", s.addr)

	// Warm models in background — primes LLM, image gen pipeline, and TTS
	go s.warmModels()

	return http.ListenAndServe(s.addr, corsMiddleware(mux))
}

// warmModels sends tiny requests to each inference service to prime their pipelines.
// Called as a goroutine at startup — does not block server readiness.
func (s *Server) warmModels() {
	time.Sleep(5 * time.Second) // give services time to initialize
	client := &http.Client{Timeout: 120 * time.Second}

	// 1. Warm LLM on phone (prime model in memory)
	fmt.Println("[warm] Warming LLM on phone...")
	llmBody := `{"model":"default","messages":[{"role":"user","content":"hi"}],"max_tokens":1,"stream":false}`
	if resp, err := client.Post("http://127.0.0.1:8085/v1/chat/completions",
		"application/json", strings.NewReader(llmBody)); err == nil {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		fmt.Println("[warm] LLM warmed ✓")
	} else {
		fmt.Printf("[warm] LLM not reachable (will warm on first use): %s\n", err.Error())
	}

	// 2. Warm image gen on Envy (compile OpenVINO pipeline)
	imageHost := s.cfg.AI.ImageGenHost
	if imageHost != "" {
		fmt.Printf("[warm] Warming image gen at %s...\n", imageHost)
		imgBody := `{"prompt":"warmup","width":256,"height":256,"steps":1}`
		if resp, err := client.Post(fmt.Sprintf("http://%s/generate", imageHost),
			"application/json", strings.NewReader(imgBody)); err == nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			fmt.Println("[warm] Image gen warmed ✓")
		} else {
			fmt.Printf("[warm] Image gen not reachable: %s\n", err.Error())
		}
	}

	// 3. Warm TTS (load piper ONNX model)
	voiceHost := s.cfg.AI.VoiceHost
	if voiceHost != "" {
		fmt.Printf("[warm] Warming TTS at %s...\n", voiceHost)
		ttsBody := `{"text":"ready"}`
		if resp, err := client.Post(fmt.Sprintf("http://%s/speak", voiceHost),
			"application/json", strings.NewReader(ttsBody)); err == nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			fmt.Println("[warm] TTS warmed ✓")
		} else {
			fmt.Printf("[warm] TTS not reachable: %s\n", err.Error())
		}
	}

	fmt.Println("[warm] Model warming complete")
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

// handleResourcesLive returns real-time system stats for Brain Net
func (s *Server) handleResourcesLive(w http.ResponseWriter, r *http.Request) {
	result := map[string]interface{}{}

	// RAM from /proc/meminfo
	if data, err := os.ReadFile("/proc/meminfo"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				switch parts[0] {
				case "MemTotal:":
					if v, e := strconv.Atoi(parts[1]); e == nil {
						result["ram_total_mb"] = v / 1024
					}
				case "MemAvailable:":
					if v, e := strconv.Atoi(parts[1]); e == nil {
						result["ram_available_mb"] = v / 1024
					}
				}
			}
		}
	}

	// Load average from /proc/loadavg
	if data, err := os.ReadFile("/proc/loadavg"); err == nil {
		parts := strings.Fields(string(data))
		if len(parts) >= 3 {
			if v, e := strconv.ParseFloat(parts[0], 64); e == nil {
				result["load_1m"] = v
			}
			if v, e := strconv.ParseFloat(parts[1], 64); e == nil {
				result["load_5m"] = v
			}
			if v, e := strconv.ParseFloat(parts[2], 64); e == nil {
				result["load_15m"] = v
			}
		}
	}

	// Uptime from /proc/uptime
	if data, err := os.ReadFile("/proc/uptime"); err == nil {
		parts := strings.Fields(string(data))
		if len(parts) >= 1 {
			if v, e := strconv.ParseFloat(parts[0], 64); e == nil {
				result["uptime_secs"] = int(v)
			}
		}
	}

	// CPU temp from /sys/class/thermal
	if data, err := os.ReadFile("/sys/class/thermal/thermal_zone0/temp"); err == nil {
		if v, e := strconv.Atoi(strings.TrimSpace(string(data))); e == nil {
			result["temp_c"] = v / 1000
		}
	}

	// Disk usage for /
	out, err := exec.Command("df", "-B1", "/").Output()
	if err == nil {
		lines := strings.Split(string(out), "\n")
		if len(lines) >= 2 {
			parts := strings.Fields(lines[1])
			if len(parts) >= 4 {
				if v, e := strconv.ParseInt(parts[1], 10, 64); e == nil {
					result["disk_total_gb"] = int(v / (1024 * 1024 * 1024))
				}
				if v, e := strconv.ParseInt(parts[3], 10, 64); e == nil {
					result["disk_free_gb"] = int(v / (1024 * 1024 * 1024))
				}
			}
		}
	}

	writeJSON(w, result)
}

// handleEnvySysinfo proxies the Envy's sysinfo server
func (s *Server) handleEnvySysinfo(w http.ResponseWriter, r *http.Request) {
	// Envy sysinfo server runs on port 8092 at the same host as image gen
	imageHost := s.cfg.AI.ImageGenHost
	if imageHost == "" {
		writeJSON(w, map[string]interface{}{"online": false})
		return
	}
	// Extract just the IP from imageHost (strip port)
	host := strings.Split(imageHost, ":")[0]
	sysinfoURL := fmt.Sprintf("http://%s:8092", host)

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(sysinfoURL)
	if err != nil {
		writeJSON(w, map[string]interface{}{"online": false})
		return
	}
	defer resp.Body.Close()

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		writeJSON(w, map[string]interface{}{"online": false})
		return
	}
	data["online"] = true
	writeJSON(w, data)
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(v)
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
	// Try to get hardware info from brain net (the actual sovereign stack host)
	// Uses the SSH tunnel on port 2222
	if res := s.detectRemoteHardware(); res != nil {
		writeJSON(w, res)
		return
	}
	// Fallback to local detection
	hw := &s.cfg.Hardware
	if hw.CPUCores == 0 || hw.RAMTotalMB == 0 {
		detected, err := hardware.Detect()
		if err == nil {
			hw = detected
			s.cfg.Hardware = *detected
		}
	}
	writeJSON(w, map[string]interface{}{
		"cpu_model":     hw.CPUModel,
		"cpu_cores":     hw.CPUCores,
		"ram_total_mb":  hw.RAMTotalMB,
		"disk_total_gb": hw.DiskTotalGB,
		"disk_free_gb":  hw.DiskFreeGB,
		"gpu_type":      hw.GPUType,
		"gpu_name":      hw.GPUName,
		"gpu_memory_mb": hw.GPUMemoryMB,
	})
}

func (s *Server) detectRemoteHardware() map[string]interface{} {
	cmd := exec.Command("ssh", "-p", "2222", "-o", "ConnectTimeout=2", "-o", "StrictHostKeyChecking=no",
		"achilles1089@localhost",
		`cat /proc/cpuinfo | grep 'model name' | head -1 | cut -d: -f2 | xargs && nproc && free -m | grep Mem | awk '{print $2, $7}' && df -BG / | tail -1 | awk '{print $2, $4}' | tr -d G`)
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 4 {
		return nil
	}
	cpuModel := strings.TrimSpace(lines[0])
	cores, _ := strconv.Atoi(strings.TrimSpace(lines[1]))
	memParts := strings.Fields(lines[2])
	diskParts := strings.Fields(lines[3])
	ramTotal := 0
	if len(memParts) >= 1 {
		ramTotal, _ = strconv.Atoi(memParts[0])
	}
	diskTotal, diskFree := 0, 0
	if len(diskParts) >= 2 {
		diskTotal, _ = strconv.Atoi(diskParts[0])
		diskFree, _ = strconv.Atoi(diskParts[1])
	}
	return map[string]interface{}{
		"cpu_model":     cpuModel,
		"cpu_cores":     cores,
		"ram_total_mb":  ramTotal,
		"disk_total_gb": diskTotal,
		"disk_free_gb":  diskFree,
		"gpu_type":      "intel_uhd",
		"gpu_name":      "Intel UHD 600",
		"gpu_memory_mb": 0,
	}
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

// handleImageGenerate proxies image generation requests to the Envy's sd_server
func (s *Server) handleImageGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(200)
		return
	}
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	imageHost := s.cfg.AI.ImageGenHost
	if imageHost == "" {
		writeJSON(w, map[string]interface{}{"error": "image_gen_host not configured"})
		return
	}

	// Read and forward the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, map[string]interface{}{"error": "failed to read request"})
		return
	}

	// Proxy to sd_server
	client := &http.Client{Timeout: 180 * time.Second}
	proxyReq, err := http.NewRequest("POST", fmt.Sprintf("http://%s/generate", imageHost), bytes.NewReader(body))
	if err != nil {
		writeJSON(w, map[string]interface{}{"error": err.Error()})
		return
	}
	proxyReq.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(proxyReq)
	if err != nil {
		writeJSON(w, map[string]interface{}{"error": fmt.Sprintf("image gen node unreachable: %s", err.Error())})
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	// Auto-save to gallery in background
	go func() {
		var genResult struct {
			Image  string `json:"image"`
			Prompt string `json:"prompt"`
			Width  int    `json:"width"`
			Height int    `json:"height"`
		}
		if err := json.Unmarshal(respBody, &genResult); err == nil && genResult.Image != "" {
			s.saveToGallery(genResult.Image, genResult.Prompt, genResult.Width, genResult.Height)
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	w.Write(respBody)
}

// handleImageStatus checks if the Envy image gen node is online
func (s *Server) handleImageStatus(w http.ResponseWriter, r *http.Request) {
	imageHost := s.cfg.AI.ImageGenHost
	if imageHost == "" {
		writeJSON(w, map[string]interface{}{"online": false, "model": ""})
		return
	}

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://%s/status", imageHost))
	if err != nil {
		writeJSON(w, map[string]interface{}{"online": false, "model": ""})
		return
	}
	defer resp.Body.Close()

	var status struct {
		Model  string `json:"model"`
		Online bool   `json:"online"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		writeJSON(w, map[string]interface{}{"online": true, "model": "unknown"})
		return
	}
	writeJSON(w, map[string]interface{}{"online": true, "model": status.Model})
}

func (s *Server) handlePhoneStatus(w http.ResponseWriter, r *http.Request) {
	// Query llama-server via ADB port forward (127.0.0.1:8085 → phone:8085)
	resp, err := http.Get("http://127.0.0.1:8085/v1/models")
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
	displayName := deriveModelDisplayName(model.ID)

	// Build base response
	response := map[string]interface{}{
		"running":      true,
		"model":        model.ID,
		"display_name": displayName,
		"params":       model.Meta.NParams,
		"vocab":        model.Meta.NVocab,
		"context":      model.Meta.NCtxTrain,
		"size_bytes":   model.Meta.Size,
		"engine":       "llama-server",
	}

	// Try to get phone hardware from sysinfo companion via ADB forward (127.0.0.1:8086)
	phoneHW := fetchPhoneHardware("127.0.0.1:8085")
	if phoneHW != nil {
		response["phone_model"] = phoneHW.PhoneModel
		response["soc"] = phoneHW.SoC
		response["android_version"] = phoneHW.AndroidVersion
		response["phone_cpu_cores"] = phoneHW.CPUCores
		response["phone_ram_total_mb"] = phoneHW.RAMTotalMB
		response["phone_ram_available_mb"] = phoneHW.RAMAvailableMB
		response["phone_storage_free_gb"] = phoneHW.StorageFreeGB
		response["battery_pct"] = phoneHW.BatteryPct
	}

	writeJSON(w, response)
}

// phoneHardwareInfo holds hardware data from the sysinfo companion
type phoneHardwareInfo struct {
	PhoneModel     string `json:"phone_model"`
	SoC            string `json:"soc"`
	AndroidVersion string `json:"android_version"`
	CPUCores       int    `json:"cpu_cores"`
	CPUFreqMHz     int    `json:"cpu_freq_mhz"`
	RAMTotalMB     int    `json:"ram_total_mb"`
	RAMAvailableMB int    `json:"ram_available_mb"`
	StorageTotalGB int    `json:"storage_total_gb"`
	StorageFreeGB  int    `json:"storage_free_gb"`
	BatteryPct     int    `json:"battery_pct"`
}

// fetchPhoneHardware queries the sysinfo companion endpoint on port 8086
// The companion runs on the phone but is accessed via ADB port forwarding
// through localhost:8086
func fetchPhoneHardware(llamaHost string) *phoneHardwareInfo {
	// ADB forward maps localhost:8086 → phone:8086
	sysinfoURL := "http://127.0.0.1:8086"

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(sysinfoURL)
	if err != nil {
		return nil // sysinfo companion not running -- graceful fallback
	}
	defer resp.Body.Close()

	var hw phoneHardwareInfo
	if err := json.NewDecoder(resp.Body).Decode(&hw); err != nil {
		return nil
	}
	return &hw
}

// handlePhoneModels lists available GGUF models on the phone
func (s *Server) handlePhoneModels(w http.ResponseWriter, r *http.Request) {
	// ADB forward maps localhost:8086 → phone:8086
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://127.0.0.1:8086/models")
	if err != nil {
		writeJSON(w, map[string]interface{}{"error": "sysinfo companion not reachable"})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Write(body)
}

// handlePhoneSwitch switches the active model on the phone by restarting llama-server
func (s *Server) handlePhoneSwitch(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(200)
		return
	}
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// ADB forward maps localhost:8086 → phone:8086
	body, _ := io.ReadAll(r.Body)
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("POST", "http://127.0.0.1:8086/switch", strings.NewReader(string(body)))
	if err != nil {
		writeJSON(w, map[string]interface{}{"ok": false, "error": err.Error()})
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		writeJSON(w, map[string]interface{}{"ok": false, "error": "sysinfo companion not reachable"})
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Write(respBody)
}

// handlePhoneStart uses ADB to start llama-server and sysinfo on the USB-connected phone
func (s *Server) handlePhoneStart(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(200)
		return
	}
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse optional model name from request body
	var req struct {
		Model string `json:"model"`
	}
	if r.Body != nil {
		json.NewDecoder(r.Body).Decode(&req)
	}
	if req.Model == "" {
		req.Model = "rwkv7-2.9B-world-q4_k_m.gguf" // default
	}

	// Step 1: Check ADB device
	out, err := exec.Command("adb", "devices").CombinedOutput()
	if err != nil || !strings.Contains(string(out), "device") {
		writeJSON(w, map[string]interface{}{"ok": false, "error": "No phone connected via USB", "detail": string(out)})
		return
	}

	// Step 2: Set up ADB port forwarding
	exec.Command("adb", "forward", "tcp:8085", "tcp:8085").Run()
	exec.Command("adb", "forward", "tcp:8086", "tcp:8086").Run()

	// Step 3: Start sysinfo_server.py
	sysinfoCmd := `export PATH=/data/data/com.termux/files/usr/bin:$PATH; ` +
		`export LD_LIBRARY_PATH=/data/data/com.termux/files/usr/lib; ` +
		`export HOME=/data/data/com.termux/files/home; ` +
		`export TMPDIR=$HOME/tmp; mkdir -p $TMPDIR; ` +
		`pkill -f sysinfo_server 2>/dev/null; sleep 1; ` +
		`nohup python3 $HOME/sysinfo_server.py > /dev/null 2>&1 &`
	exec.Command("adb", "shell", "run-as", "com.termux", "sh", "-c", sysinfoCmd).Run()

	// Step 4: Start llama-server with optimized flags for T616 (2x A75 big + 6x A55 little)
	// Pinned to big cores 6,7 via taskset. KV cache quantized to q8_0 for 50% memory savings.
	llamaCmd := fmt.Sprintf(
		`export PATH=/data/data/com.termux/files/usr/bin:$PATH; `+
			`export LD_LIBRARY_PATH=/data/data/com.termux/files/usr/lib; `+
			`export HOME=/data/data/com.termux/files/home; `+
			`killall llama-server 2>/dev/null; sleep 2; `+
			`nohup taskset -c 6,7 $HOME/llama.cpp/bld/bin/llama-server `+
			`-m $HOME/models/%s `+
			`--host 0.0.0.0 --port 8085 `+
			`--threads 2 --parallel 1 --ctx-size 4096 `+
			`--batch-size 512 --ubatch-size 256 `+
			`--flash-attn on `+
			`> /dev/null 2>&1 &`,
		req.Model,
	)
	exec.Command("adb", "shell", "run-as", "com.termux", "sh", "-c", llamaCmd).Run()

	writeJSON(w, map[string]interface{}{"ok": true, "model": req.Model, "status": "starting"})
}

// deriveModelDisplayName converts a GGUF filename/model ID into a readable display name
// Examples: "rwkv7-0.4B-world-q8_0.gguf" → "RWKV-7 0.4B"
//
//	"qwen2.5-1.5b-instruct-q4_k_m.gguf" → "Qwen 2.5 1.5B"
//	"SmolLM2-360M-Instruct-f16.gguf" → "SmolLM2 360M"
func deriveModelDisplayName(modelID string) string {
	id := strings.TrimSuffix(modelID, ".gguf")

	// RWKV-7 pattern: rwkv7-{size}-world-{quant}
	rwkvRe := regexp.MustCompile(`(?i)rwkv7[- ](\d+\.?\d*[BbMm])`)
	if m := rwkvRe.FindStringSubmatch(id); len(m) > 1 {
		return "RWKV-7 " + strings.ToUpper(m[1])
	}

	// Qwen pattern: qwen2.5-{size}-instruct-{quant}
	qwenRe := regexp.MustCompile(`(?i)qwen(\d+\.?\d*)[- ](\d+\.?\d*[BbMm])`)
	if m := qwenRe.FindStringSubmatch(id); len(m) > 2 {
		return "Qwen " + m[1] + " " + strings.ToUpper(m[2])
	}

	// SmolLM pattern: SmolLM2-360M-...
	smolRe := regexp.MustCompile(`(?i)(smollm\d*)[- ](\d+[BbMm])`)
	if m := smolRe.FindStringSubmatch(id); len(m) > 2 {
		return m[1] + " " + strings.ToUpper(m[2])
	}

	// Phi pattern: Phi-3-mini-...
	phiRe := regexp.MustCompile(`(?i)(phi[- ]?\d+)[- ](mini|small|medium)`)
	if m := phiRe.FindStringSubmatch(id); len(m) > 2 {
		return m[1] + " " + strings.Title(m[2])
	}

	// Fallback: return raw ID cleaned up
	return id
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

// --- Voice Pipeline Handlers ---

// handleTranscribe proxies audio to the local voice_server for STT
func (s *Server) handleTranscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	voiceHost := s.cfg.AI.VoiceHost
	if voiceHost == "" {
		writeJSON(w, map[string]interface{}{"error": "voice_host not configured"})
		return
	}

	// Forward the raw audio body to voice_server /transcribe
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, map[string]interface{}{"error": "failed to read audio"})
		return
	}

	client := &http.Client{Timeout: 30 * time.Second}
	proxyReq, err := http.NewRequest("POST", fmt.Sprintf("http://%s/transcribe", voiceHost), bytes.NewReader(body))
	if err != nil {
		writeJSON(w, map[string]interface{}{"error": err.Error()})
		return
	}
	proxyReq.Header.Set("Content-Type", r.Header.Get("Content-Type"))

	resp, err := client.Do(proxyReq)
	if err != nil {
		writeJSON(w, map[string]interface{}{"error": fmt.Sprintf("voice server unreachable: %s", err.Error())})
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	w.Header().Set("Content-Type", "application/json")
	w.Write(respBody)
}

// handleSpeak proxies text to the local voice_server for TTS
func (s *Server) handleSpeak(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	voiceHost := s.cfg.AI.VoiceHost
	if voiceHost == "" {
		writeJSON(w, map[string]interface{}{"error": "voice_host not configured"})
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, map[string]interface{}{"error": "failed to read request"})
		return
	}

	client := &http.Client{Timeout: 30 * time.Second}
	proxyReq, err := http.NewRequest("POST", fmt.Sprintf("http://%s/speak", voiceHost), bytes.NewReader(body))
	if err != nil {
		writeJSON(w, map[string]interface{}{"error": err.Error()})
		return
	}
	proxyReq.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(proxyReq)
	if err != nil {
		writeJSON(w, map[string]interface{}{"error": fmt.Sprintf("voice server unreachable: %s", err.Error())})
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	w.Header().Set("Content-Type", "application/json")
	w.Write(respBody)
}

// handleVoiceChat chains STT → LLM → TTS in a single request.
// Accepts raw audio blob, returns {transcript, response, audio}.
func (s *Server) handleVoiceChat(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(200)
		return
	}
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	voiceHost := s.cfg.AI.VoiceHost
	if voiceHost == "" {
		writeJSON(w, map[string]interface{}{"error": "voice_host not configured"})
		return
	}

	// Step 1: Forward audio to Whisper STT
	audioBody, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, map[string]interface{}{"error": "failed to read audio"})
		return
	}

	httpClient := &http.Client{Timeout: 30 * time.Second}
	sttReq, _ := http.NewRequest("POST", fmt.Sprintf("http://%s/transcribe", voiceHost), bytes.NewReader(audioBody))
	sttReq.Header.Set("Content-Type", r.Header.Get("Content-Type"))

	sttResp, err := httpClient.Do(sttReq)
	if err != nil {
		writeJSON(w, map[string]interface{}{"error": fmt.Sprintf("STT failed: %s", err.Error())})
		return
	}
	defer sttResp.Body.Close()

	var sttResult struct {
		Text   string `json:"text"`
		Error  string `json:"error"`
		TimeMs int    `json:"time_ms"`
	}
	if err := json.NewDecoder(sttResp.Body).Decode(&sttResult); err != nil {
		writeJSON(w, map[string]interface{}{"error": "failed to parse STT response"})
		return
	}
	if sttResult.Error != "" {
		writeJSON(w, map[string]interface{}{"error": fmt.Sprintf("STT: %s", sttResult.Error)})
		return
	}

	transcript := strings.TrimSpace(sttResult.Text)
	if transcript == "" {
		writeJSON(w, map[string]interface{}{"error": "no speech detected"})
		return
	}

	// Step 2: Send transcript to Phone LLM (collect full response)
	messages := []ai.ChatMessage{
		{Role: "user", Content: transcript},
	}

	var llmResponse strings.Builder
	err = s.client.Chat("", messages, func(content string, done bool) {
		llmResponse.WriteString(content)
	})
	if err != nil {
		writeJSON(w, map[string]interface{}{
			"error":      fmt.Sprintf("LLM failed: %s", err.Error()),
			"transcript": transcript,
		})
		return
	}

	responseText := strings.TrimSpace(llmResponse.String())
	if responseText == "" {
		writeJSON(w, map[string]interface{}{
			"transcript": transcript,
			"response":   "",
			"error":      "LLM returned empty response",
		})
		return
	}

	// Step 3: Send LLM response to Piper TTS
	ttsBody, _ := json.Marshal(map[string]string{"text": responseText})
	ttsClient := &http.Client{Timeout: 30 * time.Second}
	ttsReq, _ := http.NewRequest("POST", fmt.Sprintf("http://%s/speak", voiceHost), bytes.NewReader(ttsBody))
	ttsReq.Header.Set("Content-Type", "application/json")

	ttsResp, err := ttsClient.Do(ttsReq)
	if err != nil {
		// Return text even if TTS fails
		writeJSON(w, map[string]interface{}{
			"transcript": transcript,
			"response":   responseText,
			"error":      fmt.Sprintf("TTS failed: %s", err.Error()),
		})
		return
	}
	defer ttsResp.Body.Close()

	var ttsResult struct {
		Audio  string `json:"audio"`
		Error  string `json:"error"`
		TimeMs int    `json:"time_ms"`
	}
	json.NewDecoder(ttsResp.Body).Decode(&ttsResult)

	writeJSON(w, map[string]interface{}{
		"transcript": transcript,
		"response":   responseText,
		"audio":      ttsResult.Audio,
		"stt_ms":     sttResult.TimeMs,
		"tts_ms":     ttsResult.TimeMs,
	})
}
func (s *Server) handleVoiceStatus(w http.ResponseWriter, r *http.Request) {
	voiceHost := s.cfg.AI.VoiceHost
	if voiceHost == "" {
		writeJSON(w, map[string]interface{}{"stt_online": false, "tts_online": false})
		return
	}

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://%s/status", voiceHost))
	if err != nil {
		writeJSON(w, map[string]interface{}{"stt_online": false, "tts_online": false})
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	w.Header().Set("Content-Type", "application/json")
	w.Write(respBody)
}

// --- Music Gen Handlers ---

// handleMusicGenerate proxies to the Envy music_server for spectrogram→audio
func (s *Server) handleMusicGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	musicHost := s.cfg.AI.MusicGenHost
	if musicHost == "" {
		writeJSON(w, map[string]interface{}{"error": "music_gen_host not configured"})
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, map[string]interface{}{"error": "failed to read request"})
		return
	}

	client := &http.Client{Timeout: 120 * time.Second}
	proxyReq, err := http.NewRequest("POST", fmt.Sprintf("http://%s/generate", musicHost), bytes.NewReader(body))
	if err != nil {
		writeJSON(w, map[string]interface{}{"error": err.Error()})
		return
	}
	proxyReq.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(proxyReq)
	if err != nil {
		writeJSON(w, map[string]interface{}{"error": fmt.Sprintf("music server unreachable: %s", err.Error())})
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	w.Header().Set("Content-Type", "application/json")
	w.Write(respBody)
}

// handleMusicStatus checks if the music gen server is online
func (s *Server) handleMusicStatus(w http.ResponseWriter, r *http.Request) {
	musicHost := s.cfg.AI.MusicGenHost
	if musicHost == "" {
		writeJSON(w, map[string]interface{}{"online": false})
		return
	}

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://%s/status", musicHost))
	if err != nil {
		writeJSON(w, map[string]interface{}{"online": false})
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	w.Header().Set("Content-Type", "application/json")
	w.Write(respBody)
}

// handleRAGProxy generically proxies /api/ai/rag-{action} to the local rag_server.
// Maps: rag-upload→/upload, rag-search→/search, rag-documents→/documents, rag-delete→/document, rag-status→/status
func (s *Server) handleRAGProxy(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(200)
		return
	}

	ragHost := s.cfg.AI.RAGHost
	if ragHost == "" {
		ragHost = "localhost:8093"
	}

	// Map route to rag_server path
	var targetPath string
	switch {
	case strings.HasSuffix(r.URL.Path, "/rag-upload"):
		targetPath = "/upload"
	case strings.HasSuffix(r.URL.Path, "/rag-search"):
		targetPath = "/search"
	case strings.HasSuffix(r.URL.Path, "/rag-documents"):
		targetPath = "/documents"
	case strings.HasSuffix(r.URL.Path, "/rag-delete"):
		targetPath = "/document"
	case strings.HasSuffix(r.URL.Path, "/rag-status"):
		targetPath = "/status"
	default:
		http.Error(w, "unknown RAG endpoint", 404)
		return
	}

	// Build target URL preserving query string
	targetURL := fmt.Sprintf("http://%s%s", ragHost, targetPath)
	if r.URL.RawQuery != "" {
		targetURL += "?" + r.URL.RawQuery
	}

	// Read and forward body
	body, _ := io.ReadAll(r.Body)
	client := &http.Client{Timeout: 120 * time.Second} // uploads + embeddings can be slow

	var method string
	if strings.HasSuffix(r.URL.Path, "/rag-delete") {
		method = "DELETE"
	} else {
		method = r.Method
	}

	proxyReq, err := http.NewRequest(method, targetURL, bytes.NewReader(body))
	if err != nil {
		writeJSON(w, map[string]interface{}{"error": err.Error()})
		return
	}
	proxyReq.Header.Set("Content-Type", r.Header.Get("Content-Type"))

	resp, err := client.Do(proxyReq)
	if err != nil {
		writeJSON(w, map[string]interface{}{"error": fmt.Sprintf("RAG server unreachable: %s", err.Error())})
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	w.Header().Set("Content-Type", "application/json")
	w.Write(respBody)
}

// ─── Gallery ────────────────────────────────────────────────────────────────

const galleryDir = "/home/achilles1089/gallery"

// saveToGallery saves a base64 PNG to the gallery directory with companion metadata.
func (s *Server) saveToGallery(imageB64, prompt string, width, height int) {
	os.MkdirAll(galleryDir, 0755)

	// Strip data URI prefix if present
	if idx := strings.Index(imageB64, ","); idx > 0 {
		imageB64 = imageB64[idx+1:]
	}

	imgData, err := base64.StdEncoding.DecodeString(imageB64)
	if err != nil {
		fmt.Printf("[gallery] decode error: %v\n", err)
		return
	}

	id := fmt.Sprintf("%d", time.Now().UnixMilli())
	pngPath := filepath.Join(galleryDir, id+".png")
	metaPath := filepath.Join(galleryDir, id+".json")

	if err := os.WriteFile(pngPath, imgData, 0644); err != nil {
		fmt.Printf("[gallery] save error: %v\n", err)
		return
	}

	meta := map[string]interface{}{
		"id":         id,
		"prompt":     prompt,
		"width":      width,
		"height":     height,
		"created_at": time.Now().Format(time.RFC3339),
		"size_bytes": len(imgData),
	}
	metaJSON, _ := json.Marshal(meta)
	os.WriteFile(metaPath, metaJSON, 0644)
	fmt.Printf("[gallery] saved: %s (%d bytes)\n", id, len(imgData))
}

// handleGalleryList returns metadata for all gallery images, newest first.
func (s *Server) handleGalleryList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	entries, err := os.ReadDir(galleryDir)
	if err != nil {
		writeJSON(w, map[string]interface{}{"images": []interface{}{}})
		return
	}

	type galleryItem struct {
		ID        string `json:"id"`
		Prompt    string `json:"prompt"`
		Width     int    `json:"width"`
		Height    int    `json:"height"`
		CreatedAt string `json:"created_at"`
		SizeBytes int    `json:"size_bytes"`
	}

	var items []galleryItem
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(galleryDir, e.Name()))
		if err != nil {
			continue
		}
		var item galleryItem
		if json.Unmarshal(data, &item) == nil {
			items = append(items, item)
		}
	}

	// Sort newest first
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt > items[j].CreatedAt
	})

	writeJSON(w, map[string]interface{}{"images": items})
}

// handleGalleryImage serves a gallery image file directly.
func (s *Server) handleGalleryImage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	id := strings.TrimPrefix(r.URL.Path, "/api/gallery/image/")
	if id == "" {
		http.Error(w, "id required", 400)
		return
	}

	pngPath := filepath.Join(galleryDir, id+".png")
	if _, err := os.Stat(pngPath); os.IsNotExist(err) {
		http.Error(w, "not found", 404)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	http.ServeFile(w, r, pngPath)
}

// handleGalleryDelete removes a gallery image and its metadata.
func (s *Server) handleGalleryDelete(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Methods", "DELETE, POST, OPTIONS")
		w.WriteHeader(200)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/gallery/delete/")
	if id == "" {
		writeJSON(w, map[string]interface{}{"error": "id required"})
		return
	}

	os.Remove(filepath.Join(galleryDir, id+".png"))
	os.Remove(filepath.Join(galleryDir, id+".json"))
	writeJSON(w, map[string]interface{}{"deleted": id})
}
