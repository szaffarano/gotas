package repo

// Organization represents an Organization grouping users.
type Organization struct {
	Name  string
	Users []User
}

// User is a system user, it belongs to one organization.
type User struct {
	Name string
	Key  string
	Org  *Organization
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
func NewDefaultAuthenticator(rootDir string) (*DefaultAuthenticator, error) {
	repo, err := OpenRepository(rootDir)
	if err != nil {
		return nil, err
	}
	return &DefaultAuthenticator{repo}, nil
}

// Authenticate verifies that the given organiozation-user-key is valid.
func (a *DefaultAuthenticator) Authenticate(orgName, userName, key string) (User, error) {
	org, err := a.repo.GetOrg(orgName)
	if err != nil {
		return User{}, AuthenticationError{"400", "Invalid org"}
	}

	for _, u := range org.Users {
		if u.Key == key && u.Name == userName {
			return u, nil
		}
	}

	return User{}, AuthenticationError{"401", "Invalid username or key"}
}
