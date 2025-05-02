// pkg/usersdebug/users.go
package usersdebug

import (
	"sergey-arkhipov/nats-auth-callout-server/auth-server/auth"

	"github.com/nats-io/jwt/v2"
)

type Repository struct {
	users map[string]*auth.User
}

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

func (r *Repository) Get(username string) (*auth.User, bool) {
	user, exists := r.users[username]
	return user, exists
}
