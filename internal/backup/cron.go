package backup

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// SetupCron installs a cron job for automated backups
func SetupCron(schedule string, binaryPath string) error {
	if runtime.GOOS == "linux" {
		return setupLinuxCron(schedule, binaryPath)
	}
	if runtime.GOOS == "darwin" {
		return setupMacOSCron(schedule, binaryPath)
	}
	return fmt.Errorf("cron not supported on %s", runtime.GOOS)
}

// RemoveCron removes the sovereign backup cron job
func RemoveCron() error {
	// Get existing crontab
	out, err := exec.Command("crontab", "-l").Output()
	if err != nil {
		return nil // No crontab, nothing to remove
	}

	var newLines []string
	for _, line := range strings.Split(string(out), "\n") {
		if !strings.Contains(line, "sovereign backup") {
			newLines = append(newLines, line)
		}
	}

	return writeCrontab(strings.Join(newLines, "\n"))
}

// IsCronInstalled checks if the sovereign backup cron job exists
func IsCronInstalled() bool {
	out, err := exec.Command("crontab", "-l").Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "sovereign backup")
}

func setupLinuxCron(schedule string, binaryPath string) error {
	return installCronEntry(schedule, binaryPath)
}

func setupMacOSCron(schedule string, binaryPath string) error {
	return installCronEntry(schedule, binaryPath)
}

func installCronEntry(schedule string, binaryPath string) error {
	// Remove existing entries first
	RemoveCron()

	// Get existing crontab
	existing, _ := exec.Command("crontab", "-l").Output()

	entry := fmt.Sprintf("%s %s backup --tag auto 2>&1 | logger -t sovereign-backup\n",
		schedule, binaryPath)

	newCrontab := string(existing) + entry
	return writeCrontab(newCrontab)
}

func writeCrontab(content string) error {
	// Write to temp file and install
	tmpFile, err := os.CreateTemp("", "sovereign-cron-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		return err
	}
	tmpFile.Close()

	return exec.Command("crontab", tmpFile.Name()).Run()
}
