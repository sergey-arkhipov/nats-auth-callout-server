// pkg/users_debug/users.go

// Package usersdebug for debug user permisiions
package usersdebug

import (
	"encoding/json"

	"github.com/nats-io/jwt/v2"
)

// User store info about user
type User struct {
	Permissions jwt.Permissions `json:"permissions"`
	Pass        string          `json:"pass"`
	Account     string          `json:"account"`
}

// embeddedUsers contains the debug user data
var embeddedUsers = []byte(`{
  "sys": {
    "pass": "sys",
    "account": "SYS"
  },
  "alice": {
    "pass": "alice",
    "account": "DEVELOPMENT",
    "permissions": {
      "pub": {
        "allow": ["$JS.API.STREAM.LIST"]
      },
      "sub": {
        "allow": ["_INBOX.>", "TEST.test"]
      },
      "resp": {
        "max": 1
      }
    }
  },
  "test": {
    "pass": "test",
    "account": "TEST"
  },
  "dev": {
    "pass": "dev",
    "account": "DEVELOPMENT"
  }
}`)

var users map[string]*User

func init() {
	if err := json.Unmarshal(embeddedUsers, &users); err != nil {
		panic("failed to parse embedded users data: " + err.Error())
	}
}

// Get retrieves a user by username
func Get(username string) (*User, bool) {
	user, exists := users[username]
	return user, exists
}
