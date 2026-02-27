package backup

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// Snapshot represents a Restic backup snapshot
type Snapshot struct {
	ID       string    `json:"short_id"`
	Time     time.Time `json:"time"`
	Hostname string    `json:"hostname"`
	Tags     []string  `json:"tags"`
	Paths    []string  `json:"paths"`
}

// Manager handles Restic backup operations
type Manager struct {
	RepoPath  string
	Password  string
	DataDir   string
	ConfigDir string
}

// NewManager creates a new backup manager
func NewManager(configDir string) *Manager {
	return &Manager{
		RepoPath:  filepath.Join(configDir, "backups"),
		DataDir:   filepath.Join(configDir, "data"),
		ConfigDir: configDir,
	}
}

// SetPassword sets the repository encryption password
func (m *Manager) SetPassword(password string) {
	m.Password = password
}

// IsResticInstalled checks if restic is available
func IsResticInstalled() bool {
	_, err := exec.LookPath("restic")
	return err == nil
}

// InitRepo initializes a new Restic repository
func (m *Manager) InitRepo() error {
	if err := os.MkdirAll(m.RepoPath, 0700); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Check if already initialized
	cmd := m.resticCmd("cat", "config")
	if err := cmd.Run(); err == nil {
		return nil // Already initialized
	}

	cmd = m.resticCmd("init")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Backup creates a new backup snapshot
func (m *Manager) Backup(tags ...string) error {
	if err := os.MkdirAll(m.DataDir, 0755); err != nil {
		return fmt.Errorf("data directory not found: %w", err)
	}

	args := []string{"backup", m.DataDir, m.ConfigDir + "/config.yaml", m.ConfigDir + "/docker-compose.yml"}
	for _, tag := range tags {
		args = append(args, "--tag", tag)
	}

	// Exclude the backup repo itself
	args = append(args, "--exclude", m.RepoPath)
	args = append(args, "--exclude", "*.log")

	cmd := m.resticCmd(args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// ListSnapshots returns all backup snapshots
func (m *Manager) ListSnapshots() ([]Snapshot, error) {
	cmd := m.resticCmd("snapshots", "--json")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list snapshots: %w", err)
	}

	var snapshots []Snapshot
	if err := json.Unmarshal(out, &snapshots); err != nil {
		return nil, fmt.Errorf("failed to parse snapshots: %w", err)
	}
	return snapshots, nil
}

// Restore restores a specific snapshot
func (m *Manager) Restore(snapshotID string, target string) error {
	if target == "" {
		target = m.ConfigDir
	}

	cmd := m.resticCmd("restore", snapshotID, "--target", target)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Prune removes old snapshots based on retention policy
func (m *Manager) Prune(keepLast int, keepDaily int, keepWeekly int) error {
	args := []string{
		"forget", "--prune",
		"--keep-last", fmt.Sprintf("%d", keepLast),
		"--keep-daily", fmt.Sprintf("%d", keepDaily),
		"--keep-weekly", fmt.Sprintf("%d", keepWeekly),
	}

	cmd := m.resticCmd(args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Stats returns repository statistics
func (m *Manager) Stats() (string, error) {
	cmd := m.resticCmd("stats")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (m *Manager) resticCmd(args ...string) *exec.Cmd {
	cmd := exec.Command("restic", args...)
	cmd.Env = append(os.Environ(),
		"RESTIC_REPOSITORY="+m.RepoPath,
		"RESTIC_PASSWORD="+m.password(),
	)
	return cmd
}

func (m *Manager) password() string {
	if m.Password != "" {
		return m.Password
	}
	// Default password (user should change this)
	return "sovereign-default-key"
}
