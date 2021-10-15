package repo

import "github.com/szaffarano/gotas/pkg/task/auth"

// DefaultAuthenticator is the default Authenticator implementation on top of a
// simple fylesystem structure
type DefaultAuthenticator struct {
	repo *Repository
}

// NewDefaultAuthenticator creates a new Arthenticator
func NewDefaultAuthenticator(rootDir string) (*DefaultAuthenticator, error) {
	repo, err := OpenRepository(rootDir)
	if err != nil {
		return nil, err
	}
	return &DefaultAuthenticator{repo}, nil
}

// Authenticate verifies that the given organiozation-user-key is valid.
func (a *DefaultAuthenticator) Authenticate(orgName, userName, key string) (auth.User, error) {
	org, err := a.repo.GetOrg(orgName)
	if err != nil {
		return auth.User{}, auth.AuthenticationError{Code: "400", Msg: "Invalid org"}
	}

	for _, u := range org.Users {
		if u.Key == key && u.Name == userName {
			return u, nil
		}
	}

	return auth.User{}, auth.AuthenticationError{Code: "401", Msg: "Invalid username or key"}
}
