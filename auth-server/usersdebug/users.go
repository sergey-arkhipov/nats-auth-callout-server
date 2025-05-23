// Package usersdebug provides users for test purposes by loading from a JSON file
package usersdebug

import (
	"encoding/json"
	"os"
	"sergey-arkhipov/nats-auth-callout-server/auth-server/auth"

	"github.com/nats-io/jwt/v2"
)

// Repository allows calling test users
type Repository struct {
	users map[string]*auth.User
}

// New returns a Repository struct with users loaded from users.json
func New(filePath string) (*Repository, error) {
	// Read the JSON file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Define a struct to match the JSON structure
	type jsonUser struct {
		Pass        string           `json:"Pass"`
		Account     string           `json:"Account"`
		Permissions *jwt.Permissions `json:"Permissions,omitempty"`
	}

	// Unmarshal JSON into a map
	var jsonUsers map[string]jsonUser
	if err := json.Unmarshal(data, &jsonUsers); err != nil {
		return nil, err
	}

	// Convert jsonUser to auth.User
	users := make(map[string]*auth.User)
	for username, ju := range jsonUsers {
		user := &auth.User{
			Pass:    ju.Pass,
			Account: ju.Account,
		}
		if ju.Permissions != nil {
			user.Permissions = *ju.Permissions
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
