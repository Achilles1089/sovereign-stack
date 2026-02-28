package rbac

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHasPermission(t *testing.T) {
	tests := []struct {
		role     Role
		perm     Permission
		expected bool
	}{
		// Admin has everything
		{RoleAdmin, PermAppInstall, true},
		{RoleAdmin, PermRBACManage, true},
		{RoleAdmin, PermConfigWrite, true},

		// Operator has most but not RBAC
		{RoleOperator, PermAppInstall, true},
		{RoleOperator, PermRBACManage, false},
		{RoleOperator, PermConfigWrite, false},

		// Viewer is read-only
		{RoleViewer, PermAppList, true},
		{RoleViewer, PermAppInstall, false},
		{RoleViewer, PermDashboard, true},

		// Backup role is restricted
		{RoleBackup, PermBackupCreate, true},
		{RoleBackup, PermBackupList, true},
		{RoleBackup, PermAppInstall, false},
		{RoleBackup, PermConfigRead, false},
	}

	for _, tt := range tests {
		name := string(tt.role) + "/" + string(tt.perm)
		t.Run(name, func(t *testing.T) {
			result := HasPermission(tt.role, tt.perm)
			if result != tt.expected {
				t.Errorf("HasPermission(%s, %s) = %v, want %v",
					tt.role, tt.perm, result, tt.expected)
			}
		})
	}
}

func TestAvailableRoles(t *testing.T) {
	roles := AvailableRoles()
	if len(roles) != 4 {
		t.Errorf("expected 4 roles, got %d", len(roles))
	}
}

func TestGetPermissions(t *testing.T) {
	adminPerms := GetPermissions(RoleAdmin)
	viewerPerms := GetPermissions(RoleViewer)

	if len(adminPerms) <= len(viewerPerms) {
		t.Error("admin should have more permissions than viewer")
	}
}

func TestAddRemoveUser(t *testing.T) {
	// Use temp dir for config
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	os.MkdirAll(filepath.Join(tmpDir, ".sovereign"), 0755)
	defer os.Unsetenv("HOME")

	err := AddUser("testuser", RoleOperator, "test@example.com")
	if err != nil {
		t.Fatalf("AddUser failed: %v", err)
	}

	user, err := GetUser("testuser")
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}
	if user.Role != RoleOperator {
		t.Errorf("expected operator role, got %s", user.Role)
	}

	// Duplicate should fail
	err = AddUser("testuser", RoleViewer, "")
	if err == nil {
		t.Error("expected error for duplicate user")
	}

	// Remove
	err = RemoveUser("testuser")
	if err != nil {
		t.Fatalf("RemoveUser failed: %v", err)
	}

	_, err = GetUser("testuser")
	if err == nil {
		t.Error("expected error after removal")
	}
}
