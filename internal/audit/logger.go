package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Achilles1089/sovereign-stack/internal/config"
)

// Event represents a single audit log entry
type Event struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Action    string    `json:"action"`   // e.g., "app.install", "backup.create", "config.change"
	Actor     string    `json:"actor"`    // "admin", "system", "cron", or username
	Target    string    `json:"target"`   // e.g., "nextcloud", "backup/snapshot-123"
	Details   string    `json:"details"`  // human-readable description
	Severity  string    `json:"severity"` // "info", "warning", "critical"
	Success   bool      `json:"success"`
}

// Logger writes audit events to a JSONL log file
type Logger struct {
	mu      sync.Mutex
	logDir  string
	maxSize int64 // max log file size before rotation (bytes)
}

// NewLogger creates an audit logger
func NewLogger() *Logger {
	return &Logger{
		logDir:  filepath.Join(config.ConfigDir(), "audit"),
		maxSize: 10 * 1024 * 1024, // 10MB
	}
}

// Log records an audit event
func (l *Logger) Log(event Event) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	if event.ID == "" {
		event.ID = fmt.Sprintf("%d", event.Timestamp.UnixNano())
	}

	// Ensure audit directory exists
	if err := os.MkdirAll(l.logDir, 0700); err != nil {
		return err
	}

	logPath := l.currentLogPath()

	// Rotate if needed
	if info, err := os.Stat(logPath); err == nil && info.Size() > l.maxSize {
		l.rotate(logPath)
	}

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(f, "%s\n", data)
	return err
}

// Query returns recent audit events matching the filter
func (l *Logger) Query(action string, limit int) ([]Event, error) {
	logPath := l.currentLogPath()
	data, err := os.ReadFile(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var events []Event
	for _, line := range splitLines(string(data)) {
		if line == "" {
			continue
		}
		var ev Event
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			continue
		}
		if action == "" || ev.Action == action {
			events = append(events, ev)
		}
	}

	// Return last N events (most recent)
	if limit > 0 && len(events) > limit {
		events = events[len(events)-limit:]
	}

	return events, nil
}

// LogAppInstall records an app installation
func (l *Logger) LogAppInstall(appName string, success bool) {
	l.Log(Event{
		Action:   "app.install",
		Actor:    "admin",
		Target:   appName,
		Details:  fmt.Sprintf("Installed app: %s", appName),
		Severity: "info",
		Success:  success,
	})
}

// LogAppRemove records an app removal
func (l *Logger) LogAppRemove(appName string) {
	l.Log(Event{
		Action:   "app.remove",
		Actor:    "admin",
		Target:   appName,
		Details:  fmt.Sprintf("Removed app: %s", appName),
		Severity: "info",
		Success:  true,
	})
}

// LogBackup records a backup event
func (l *Logger) LogBackup(tag string, success bool) {
	l.Log(Event{
		Action:   "backup.create",
		Actor:    "admin",
		Target:   "backup/" + tag,
		Details:  fmt.Sprintf("Created backup snapshot (tag: %s)", tag),
		Severity: "info",
		Success:  success,
	})
}

// LogConfigChange records a config modification
func (l *Logger) LogConfigChange(field string, oldVal, newVal string) {
	l.Log(Event{
		Action:   "config.change",
		Actor:    "admin",
		Target:   "config/" + field,
		Details:  fmt.Sprintf("Changed %s: %s → %s", field, oldVal, newVal),
		Severity: "warning",
		Success:  true,
	})
}

// LogMeshEvent records a mesh networking event
func (l *Logger) LogMeshEvent(action string, peerName string) {
	l.Log(Event{
		Action:   "mesh." + action,
		Actor:    "admin",
		Target:   "mesh/" + peerName,
		Details:  fmt.Sprintf("Mesh %s: %s", action, peerName),
		Severity: "info",
		Success:  true,
	})
}

// LogAuthEvent records an authentication event
func (l *Logger) LogAuthEvent(username string, success bool) {
	sev := "info"
	if !success {
		sev = "warning"
	}
	l.Log(Event{
		Action:   "auth.login",
		Actor:    username,
		Target:   "dashboard",
		Details:  fmt.Sprintf("Login attempt by %s", username),
		Severity: sev,
		Success:  success,
	})
}

func (l *Logger) currentLogPath() string {
	return filepath.Join(l.logDir, "audit.jsonl")
}

func (l *Logger) rotate(path string) {
	rotated := fmt.Sprintf("%s.%s", path, time.Now().Format("20060102-150405"))
	os.Rename(path, rotated)
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
