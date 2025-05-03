// Package usersdebug return users for test purpose
package usersdebug

import (
	"sergey-arkhipov/nats-auth-callout-server/auth-server/auth"

	"github.com/nats-io/jwt/v2"
)

// Repository allow call test user
type Repository struct {
	users map[string]*auth.User
}

// New return struct with users for test
func New() *Repository {
	return &Repository{
		users: map[string]*auth.User{
			"sys": {
				Pass:    "sys",
				Account: "SYS",
			},
			"alice": {
				Pass:    "alice",
				Account: "DEVELOPMENT",
				Permissions: jwt.Permissions{
					Pub:  jwt.Permission{Allow: []string{"$JS.API.STREAM.LIST"}},
					Sub:  jwt.Permission{Allow: []string{"_INBOX.>", "TEST.test"}},
					Resp: &jwt.ResponsePermission{MaxMsgs: 1},
				},
			},
			"test": {
				Pass:    "test",
				Account: "TEST",
			},
			"dev": {
				Pass:    "dev",
				Account: "DEVELOPMENT",
			},
		},
	}
}

// Get return User from test file
func (r *Repository) Get(username string) (*auth.User, bool) {
	user, exists := r.users[username]
	return user, exists
}
