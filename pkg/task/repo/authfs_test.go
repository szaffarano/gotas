package repo

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/szaffarano/gotas/pkg/task/auth"
)

func TestAuthenticate(t *testing.T) {
	a := validAuthenticator(t)
	cases := []struct {
		org     string
		name    string
		key     string
		success bool
	}{
		{"Public", "noeh", "53938cd8-b72e-4c2a-9fb5-3cd183cf1fa7", true},
		{"Public", "john", "53938cd8-b72e-4c2a-9fb5-3cd183cf1fa7", false},
		{"non-existent", "noeh", "53938cd8-b72e-4c2a-9fb5-3cd183cf1fa7", false},
		{"Public", "non-existent", "53938cd8-b72e-4c2a-9fb5-3cd183cf1fa7", false},
		{"Public", "noeh", "invalid key", false},
	}

	for _, c := range cases {
		u, err := a.Authenticate(c.org, c.name, c.key)
		if c.success {
			assert.Nil(t, err)
			assert.Equal(t, u.Name, "noeh")
		} else {
			assert.NotNil(t, err)
			authErr, ok := err.(auth.AuthenticationError)
			assert.True(t, ok)
			assert.NotEmpty(t, authErr.Msg)
			assert.NotEmpty(t, authErr.Error())
		}
	}
}

func validAuthenticator(t *testing.T) *DefaultAuthenticator {
	t.Helper()

	configFilePath := filepath.Join("testdata", "repo_one")
	auth, err := NewDefaultAuthenticator(configFilePath)
	if err != nil {
		assert.FailNow(t, err.Error())
	}

	return auth
}
