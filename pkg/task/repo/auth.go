package repo

import (
	"github.com/szaffarano/gotas/pkg/config"
	"github.com/szaffarano/gotas/pkg/task/task"
)

// Authenticator exposes the logic needed to deal with security functionality
type Authenticator interface {
	Authenticate(org, user, key string) (task.User, error)
}

// DefaultAuthenticator is the default Authenticator implementation on top of a
// simple fylesystem structure
type DefaultAuthenticator struct {
	repo *Repository
}

// AuthenticationError represents any authentication-related error.  It
// contains a code meant to be used as a response code.
type AuthenticationError struct {
	Code string
	Msg  string
}

// Error makes AuthenticationError an error.
func (e AuthenticationError) Error() string {
	return e.Msg
}

// NewDefaultAuthenticator creates a new Arthenticator
func NewDefaultAuthenticator(cfg config.Config) (*DefaultAuthenticator, error) {
	repo, err := OpenRepository(cfg.Get(task.Root))
	if err != nil {
		return nil, err
	}
	return &DefaultAuthenticator{repo}, nil
}

// Authenticate verifies that the given organiozation-user-key is valid.
func (a *DefaultAuthenticator) Authenticate(orgName, userName, key string) (task.User, error) {
	org, err := a.repo.GetOrg(orgName)
	if err != nil {
		return task.User{}, AuthenticationError{"400", "Invalid org"}
	}

	for _, u := range org.Users {
		if u.Key == key && u.Name == userName {
			return u, nil
		}
	}

	return task.User{}, AuthenticationError{"401", "Invalid username or key"}
}
