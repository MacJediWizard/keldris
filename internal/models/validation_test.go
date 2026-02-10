package models

import (
	"testing"

	"github.com/google/uuid"
)

func TestRepository_IsValidType(t *testing.T) {
	tests := []struct {
		name     string
		repoType RepositoryType
		valid    bool
	}{
		{"local", RepositoryTypeLocal, true},
		{"s3", RepositoryTypeS3, true},
		{"b2", RepositoryTypeB2, true},
		{"sftp", RepositoryTypeSFTP, true},
		{"rest", RepositoryTypeRest, true},
		{"dropbox", RepositoryTypeDropbox, true},
		{"invalid type", RepositoryType("azure"), false},
		{"empty type", RepositoryType(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &Repository{Type: tt.repoType}
			if got := repo.IsValidType(); got != tt.valid {
				t.Errorf("IsValidType() = %v, want %v", got, tt.valid)
			}
		})
	}
}

func TestValidRepositoryTypes(t *testing.T) {
	types := ValidRepositoryTypes()
	if len(types) != 6 {
		t.Errorf("expected 6 valid types, got %d", len(types))
	}

	expected := map[RepositoryType]bool{
		RepositoryTypeLocal:   true,
		RepositoryTypeS3:     true,
		RepositoryTypeB2:     true,
		RepositoryTypeSFTP:   true,
		RepositoryTypeRest:   true,
		RepositoryTypeDropbox: true,
	}
	for _, rt := range types {
		if !expected[rt] {
			t.Errorf("unexpected type: %s", rt)
		}
	}
}

func TestBackupScriptType_IsValid(t *testing.T) {
	tests := []struct {
		name       string
		scriptType BackupScriptType
		valid      bool
	}{
		{"pre_backup", BackupScriptTypePreBackup, true},
		{"post_success", BackupScriptTypePostSuccess, true},
		{"post_failure", BackupScriptTypePostFailure, true},
		{"post_always", BackupScriptTypePostAlways, true},
		{"invalid", BackupScriptType("during_backup"), false},
		{"empty", BackupScriptType(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.scriptType.IsValid(); got != tt.valid {
				t.Errorf("IsValid() = %v, want %v", got, tt.valid)
			}
		})
	}
}

func TestValidBackupScriptTypes(t *testing.T) {
	types := ValidBackupScriptTypes()
	if len(types) != 4 {
		t.Errorf("expected 4 valid types, got %d", len(types))
	}
}

func TestIsValidOrgRole(t *testing.T) {
	tests := []struct {
		name  string
		role  string
		valid bool
	}{
		{"owner", "owner", true},
		{"admin", "admin", true},
		{"member", "member", true},
		{"readonly", "readonly", true},
		{"invalid", "superadmin", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidOrgRole(tt.role); got != tt.valid {
				t.Errorf("IsValidOrgRole(%q) = %v, want %v", tt.role, got, tt.valid)
			}
		})
	}
}

func TestValidOrgRoles(t *testing.T) {
	roles := ValidOrgRoles()
	if len(roles) != 4 {
		t.Errorf("expected 4 valid roles, got %d", len(roles))
	}
}

func TestOrgMembership_RoleChecks(t *testing.T) {
	tests := []struct {
		name     string
		role     OrgRole
		isOwner  bool
		isAdmin  bool
		canWrite bool
	}{
		{"owner", OrgRoleOwner, true, true, true},
		{"admin", OrgRoleAdmin, false, true, true},
		{"member", OrgRoleMember, false, false, true},
		{"readonly", OrgRoleReadonly, false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewOrgMembership(uuid.New(), uuid.New(), tt.role)

			if got := m.IsOwner(); got != tt.isOwner {
				t.Errorf("IsOwner() = %v, want %v", got, tt.isOwner)
			}
			if got := m.IsAdmin(); got != tt.isAdmin {
				t.Errorf("IsAdmin() = %v, want %v", got, tt.isAdmin)
			}
			if got := m.CanWrite(); got != tt.canWrite {
				t.Errorf("CanWrite() = %v, want %v", got, tt.canWrite)
			}
		})
	}
}

func TestUser_IsAdmin(t *testing.T) {
	tests := []struct {
		name    string
		role    UserRole
		isAdmin bool
	}{
		{"admin", UserRoleAdmin, true},
		{"user", UserRoleUser, false},
		{"viewer", UserRoleViewer, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := NewUser(uuid.New(), "sub-123", "test@example.com", "Test", tt.role)
			if got := user.IsAdmin(); got != tt.isAdmin {
				t.Errorf("IsAdmin() = %v, want %v", got, tt.isAdmin)
			}
		})
	}
}

func TestBackupScript_TypeChecks(t *testing.T) {
	tests := []struct {
		name             string
		scriptType       BackupScriptType
		isPreBackup      bool
		isPostScript     bool
		shouldRunSuccess bool
		shouldRunFailure bool
	}{
		{"pre_backup", BackupScriptTypePreBackup, true, false, false, false},
		{"post_success", BackupScriptTypePostSuccess, false, true, true, false},
		{"post_failure", BackupScriptTypePostFailure, false, true, false, true},
		{"post_always", BackupScriptTypePostAlways, false, true, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			script := NewBackupScript(uuid.New(), tt.scriptType, "echo test")

			if got := script.IsPreBackup(); got != tt.isPreBackup {
				t.Errorf("IsPreBackup() = %v, want %v", got, tt.isPreBackup)
			}
			if got := script.IsPostScript(); got != tt.isPostScript {
				t.Errorf("IsPostScript() = %v, want %v", got, tt.isPostScript)
			}
			if got := script.ShouldRunOnSuccess(); got != tt.shouldRunSuccess {
				t.Errorf("ShouldRunOnSuccess() = %v, want %v", got, tt.shouldRunSuccess)
			}
			if got := script.ShouldRunOnFailure(); got != tt.shouldRunFailure {
				t.Errorf("ShouldRunOnFailure() = %v, want %v", got, tt.shouldRunFailure)
			}
		})
	}
}

func TestScheduleRepository_IsPrimary(t *testing.T) {
	tests := []struct {
		name     string
		priority int
		primary  bool
	}{
		{"primary", 0, true},
		{"secondary", 1, false},
		{"tertiary", 2, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sr := NewScheduleRepository(uuid.New(), uuid.New(), tt.priority)
			if got := sr.IsPrimary(); got != tt.primary {
				t.Errorf("IsPrimary() = %v, want %v", got, tt.primary)
			}
		})
	}
}

func TestRepositoryKey_HasEscrow(t *testing.T) {
	tests := []struct {
		name          string
		escrowEnabled bool
		escrowKey     []byte
		hasEscrow     bool
	}{
		{"enabled with key", true, []byte("encrypted-key"), true},
		{"enabled without key", true, nil, false},
		{"enabled with empty key", true, []byte{}, false},
		{"disabled with key", false, []byte("encrypted-key"), false},
		{"disabled without key", false, nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rk := NewRepositoryKey(uuid.New(), []byte("key"), tt.escrowEnabled, tt.escrowKey)
			if got := rk.HasEscrow(); got != tt.hasEscrow {
				t.Errorf("HasEscrow() = %v, want %v", got, tt.hasEscrow)
			}
		})
	}
}
