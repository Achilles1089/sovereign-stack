package rbac

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Achilles1089/sovereign-stack/internal/config"
)

// Role defines a set of permissions
type Role string

const (
	RoleAdmin    Role = "admin"    // Full access to everything
	RoleOperator Role = "operator" // Can manage services, apps, backups — cannot change RBAC
	RoleViewer   Role = "viewer"   // Read-only access to dashboard and status
	RoleBackup   Role = "backup"   // Can only manage backups
)

// Permission represents a specific action
type Permission string

const (
	PermAppInstall    Permission = "app.install"
	PermAppRemove     Permission = "app.remove"
	PermAppList       Permission = "app.list"
	PermServiceStart  Permission = "service.start"
	PermServiceStop   Permission = "service.stop"
	PermServiceLogs   Permission = "service.logs"
	PermBackupCreate  Permission = "backup.create"
	PermBackupRestore Permission = "backup.restore"
	PermBackupList    Permission = "backup.list"
	PermConfigRead    Permission = "config.read"
	PermConfigWrite   Permission = "config.write"
	PermMeshManage    Permission = "mesh.manage"
	PermAIChat        Permission = "ai.chat"
	PermAIManage      Permission = "ai.manage"
	PermRBACManage    Permission = "rbac.manage"
	PermAuditRead     Permission = "audit.read"
	PermDashboard     Permission = "dashboard.view"
)

// User represents an RBAC user
type User struct {
	Username string `json:"username"`
	Role     Role   `json:"role"`
	Email    string `json:"email,omitempty"`
	Active   bool   `json:"active"`
}

// RBACConfig holds all RBAC configuration
type RBACConfig struct {
	Enabled bool   `json:"enabled"`
	Users   []User `json:"users"`
}

// rolePermissions maps roles to their permissions
var rolePermissions = map[Role][]Permission{
	RoleAdmin: {
		PermAppInstall, PermAppRemove, PermAppList,
		PermServiceStart, PermServiceStop, PermServiceLogs,
		PermBackupCreate, PermBackupRestore, PermBackupList,
		PermConfigRead, PermConfigWrite,
		PermMeshManage, PermAIChat, PermAIManage,
		PermRBACManage, PermAuditRead, PermDashboard,
	},
	RoleOperator: {
		PermAppInstall, PermAppRemove, PermAppList,
		PermServiceStart, PermServiceStop, PermServiceLogs,
		PermBackupCreate, PermBackupRestore, PermBackupList,
		PermConfigRead,
		PermMeshManage, PermAIChat,
		PermAuditRead, PermDashboard,
	},
	RoleViewer: {
		PermAppList,
		PermServiceLogs,
		PermBackupList,
		PermConfigRead,
		PermAuditRead, PermDashboard,
	},
	RoleBackup: {
		PermBackupCreate, PermBackupRestore, PermBackupList,
		PermDashboard,
	},
}

// HasPermission checks if a role has a specific permission
func HasPermission(role Role, perm Permission) bool {
	perms, ok := rolePermissions[role]
	if !ok {
		return false
	}
	for _, p := range perms {
		if p == perm {
			return true
		}
	}
	return false
}

// GetPermissions returns all permissions for a role
func GetPermissions(role Role) []Permission {
	return rolePermissions[role]
}

// AvailableRoles returns all defined roles
func AvailableRoles() []Role {
	return []Role{RoleAdmin, RoleOperator, RoleViewer, RoleBackup}
}

// LoadConfig loads RBAC configuration
func LoadConfig() (*RBACConfig, error) {
	path := filepath.Join(config.ConfigDir(), "rbac.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Default: RBAC disabled, single admin user
			return &RBACConfig{
				Enabled: false,
				Users: []User{
					{Username: "admin", Role: RoleAdmin, Active: true},
				},
			}, nil
		}
		return nil, err
	}

	var cfg RBACConfig
	return &cfg, json.Unmarshal(data, &cfg)
}

// SaveConfig saves RBAC configuration
func SaveConfig(cfg *RBACConfig) error {
	path := filepath.Join(config.ConfigDir(), "rbac.json")
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// AddUser adds a user with a role
func AddUser(username string, role Role, email string) error {
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}

	// Check for duplicates
	for _, u := range cfg.Users {
		if u.Username == username {
			return fmt.Errorf("user %q already exists", username)
		}
	}

	cfg.Users = append(cfg.Users, User{
		Username: username,
		Role:     role,
		Email:    email,
		Active:   true,
	})

	return SaveConfig(cfg)
}

// RemoveUser removes a user
func RemoveUser(username string) error {
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}

	var updated []User
	found := false
	for _, u := range cfg.Users {
		if u.Username == username {
			found = true
			continue
		}
		updated = append(updated, u)
	}

	if !found {
		return fmt.Errorf("user %q not found", username)
	}

	cfg.Users = updated
	return SaveConfig(cfg)
}

// GetUser finds a user by username
func GetUser(username string) (*User, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	for _, u := range cfg.Users {
		if u.Username == username {
			return &u, nil
		}
	}

	return nil, fmt.Errorf("user %q not found", username)
}
