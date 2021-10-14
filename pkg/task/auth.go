package task

import "github.com/szaffarano/gotas/pkg/task/repo"

// Authenticator exposes the logic needed to deal with security functionality
type Authenticator interface {
	Authenticate(org, user, key string) (repo.User, error)
}
