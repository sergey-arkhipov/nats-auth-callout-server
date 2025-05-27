// Package usersdebug provides users for test purposes by loading from a YAML file
package usersdebug

import (
	"os"
	"sergey-arkhipov/nats-auth-callout-server/auth-server/auth"

	"github.com/nats-io/jwt/v2"
	"gopkg.in/yaml.v3"
)

// Repository allows calling test users
type Repository struct {
	users map[string]*auth.User
}

// New returns a Repository struct with users loaded from users.yaml
func New() (*Repository, error) {
	// Read the YAML file
	data, err := os.ReadFile("users.yaml")
	if err != nil {
		return nil, err
	}

	// Define a struct to match the YAML structure
	type yamlUser struct {
		Pass        string           `yaml:"Pass"`
		Account     string           `yaml:"Account"`
		Permissions *jwt.Permissions `yaml:"Permissions,omitempty"`
	}

	// Unmarshal YAML into a map
	var yamlUsers map[string]yamlUser
	if err := yaml.Unmarshal(data, &yamlUsers); err != nil {
		return nil, err
	}

	// Convert yamlUser to auth.User
	users := make(map[string]*auth.User)
	for username, yu := range yamlUsers {
		user := &auth.User{
			Pass:    yu.Pass,
			Account: yu.Account,
		}
		if yu.Permissions != nil {
			user.Permissions = *yu.Permissions
		}
		users[username] = user
	}

	return &Repository{
		users: users,
	}, nil
}

// Get returns a User from the repository
func (r *Repository) Get(username string) (*auth.User, bool) {
	user, exists := r.users[username]
	return user, exists
}
