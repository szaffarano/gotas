package auth

// Authenticator exposes the logic needed to deal with security functionality
type Authenticator interface {
	Authenticate(org, user, key string) (User, error)
}

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
