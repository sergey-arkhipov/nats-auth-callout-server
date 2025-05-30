package usersdebug

import (
	"os"
	"reflect"
	"sergey-arkhipov/nats-auth-callout-server/auth-server/auth"
	"testing"

	"github.com/nats-io/jwt/v2"
)

// TestNew tests the New function for creating a Repository from users.yaml
func TestNew(t *testing.T) {
	// Helper function to create a temporary users.yaml file in the current directory
	createTempUsersYAML := func(t *testing.T, content string) func() {
		t.Helper()
		// Ensure the file is named "users.yaml" in the current directory
		filePath := "users.yaml"
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write users.yaml: %v", err)
		}
		// Return a cleanup function
		return func() {
			if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
				t.Errorf("Failed to clean up users.yaml: %v", err)
			}
		}
	}

	// Test cases
	tests := []struct {
		name        string
		yamlContent string
		wantErr     bool
		validate    func(t *testing.T, repo *Repository)
	}{
		{
			name: "Valid YAML file",
			yamlContent: `
sys:
  Pass: sys
  Account: SYS
alice:
  Pass: alice
  Account: DEVELOPMENT
  Permissions:
    pub:
      allow:
        - $JS.API.STREAM.LIST
    sub:
      allow:
        - _INBOX.>
        - TEST.test
`,
			wantErr: false,
			validate: func(t *testing.T, repo *Repository) {
				if len(repo.users) != 2 {
					t.Errorf("Expected 2 users, got %d", len(repo.users))
				}
				if user, exists := repo.users["sys"]; !exists || user.Pass != "sys" || user.Account != "SYS" {
					t.Errorf("Expected user 'sys' with Pass=sys, Account=SYS, got %+v, exists=%v", user, exists)
				}
				if user, exists := repo.users["alice"]; !exists || user.Pass != "alice" || user.Account != "DEVELOPMENT" {
					t.Errorf("Expected user 'alice' with Pass=alice, Account=DEVELOPMENT, got %+v, exists=%v", user, exists)
				}
				if user, exists := repo.users["alice"]; exists && len(user.Permissions.Pub.Allow) != 1 {
					t.Errorf("Expected alice to have 1 Pub permission, got %v", user.Permissions.Pub.Allow)
				}
				if user, exists := repo.users["alice"]; exists && len(user.Permissions.Sub.Allow) != 2 {
					t.Errorf("Expected alice to have 2 Sub permissions, got %v", user.Permissions.Sub.Allow)
				}
			},
		},
		{
			name:        "Non-existent YAML file",
			yamlContent: "", // No file created
			wantErr:     true,
		},
		{
			name:        "Invalid YAML file",
			yamlContent: `invalid yaml: : :`,
			wantErr:     true,
		},
		{
			name:        "Empty YAML file",
			yamlContent: `{}`,
			wantErr:     false,
			validate: func(t *testing.T, repo *Repository) {
				if len(repo.users) != 0 {
					t.Errorf("Expected 0 users, got %d", len(repo.users))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create users.yaml if yamlContent is provided
			var cleanup func()
			if tt.yamlContent != "" {
				cleanup = createTempUsersYAML(t, tt.yamlContent)
				defer cleanup()
			}

			// Run the New function
			repo, err := New()
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, repo)
			}
		})
	}
}

// TestGet tests the Get function for retrieving users from the Repository
func TestGet(t *testing.T) {
	// Create a test repository
	repo := &Repository{
		users: map[string]*auth.User{
			"sys": {
				Pass:    "sys",
				Account: "SYS",
			},
			"alice": {
				Pass:    "alice",
				Account: "DEVELOPMENT",
				Permissions: jwt.Permissions{
					Pub: jwt.Permission{Allow: []string{"$JS.API.STREAM.LIST"}},
					Sub: jwt.Permission{Allow: []string{"_INBOX.>", "TEST.test"}},
				},
			},
		},
	}

	tests := []struct {
		name      string
		username  string
		wantExist bool
		wantUser  *auth.User
	}{
		{
			name:      "Existing user sys",
			username:  "sys",
			wantExist: true,
			wantUser: &auth.User{
				Pass:    "sys",
				Account: "SYS",
			},
		},
		{
			name:      "Existing user alice with permissions",
			username:  "alice",
			wantExist: true,
			wantUser: &auth.User{
				Pass:    "alice",
				Account: "DEVELOPMENT",
				Permissions: jwt.Permissions{
					Pub: jwt.Permission{Allow: []string{"$JS.API.STREAM.LIST"}},
					Sub: jwt.Permission{Allow: []string{"_INBOX.>", "TEST.test"}},
				},
			},
		},
		{
			name:      "Non-existent user",
			username:  "unknown",
			wantExist: false,
			wantUser:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotUser, gotExist := repo.Get(tt.username)
			if gotExist != tt.wantExist {
				t.Errorf("Get(%q) exists = %v, want %v", tt.username, gotExist, tt.wantExist)
			}
			if !reflect.DeepEqual(gotUser, tt.wantUser) {
				t.Errorf("Get(%q) user = %+v, want %+v", tt.username, gotUser, tt.wantUser)
			}
		})
	}
}
