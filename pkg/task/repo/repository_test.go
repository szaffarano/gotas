//go:build linux

package repo

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/szaffarano/gotas/pkg/task/task"
)

func TestNewRepository(t *testing.T) {
	t.Run("new repository works with empty directory", func(t *testing.T) {
		baseDir := tempDir(t)
		defer os.RemoveAll(baseDir)

		repo, err := NewRepository(baseDir)

		assert.Nil(t, err)
		assert.NotNil(t, repo)
	})

	t.Run("new repository fails with non empty directory", func(t *testing.T) {
		baseDir := tempDir(t)
		defer os.RemoveAll(baseDir)

		repo, err := NewRepository(baseDir)

		assert.Nil(t, err)
		assert.NotNil(t, repo)

		_, err = NewRepository(baseDir)
		assert.NotNil(t, err)
	})

	t.Run("new repository fails with non existent repository", func(t *testing.T) {
		_, err := NewRepository("fake dir")
		assert.NotNil(t, err)
	})

	t.Run("new repository fails when invalid datadir", func(t *testing.T) {
		baseDir := tempDir(t)
		defer os.RemoveAll(baseDir)

		filePath := filepath.Join(baseDir, "file")
		file, err := os.Create(filePath)
		assert.Nil(t, err)
		defer file.Close()

		_, err = NewRepository(filePath)
		assert.NotNil(t, err)
	})

	t.Run("new repository fails when invalid permission for read dir", func(t *testing.T) {
		baseDir := tempDir(t)
		defer os.RemoveAll(baseDir)

		err := os.Chmod(baseDir, 0400)
		assert.Nil(t, err)

		_, err = NewRepository(baseDir)

		assert.NotNil(t, err)
	})

	t.Run("new repository fails when invalid invalid permission", func(t *testing.T) {
		baseDir := tempDir(t)
		defer os.RemoveAll(baseDir)

		err := os.Chmod(baseDir, 0000)
		assert.Nil(t, err)

		_, err = NewRepository(baseDir)

		assert.NotNil(t, err)
	})

}

func TestOpenRepository(t *testing.T) {
	t.Run("open repository works with existing repository", func(t *testing.T) {
		repo, err := OpenRepository(filepath.Join("testdata", "repo_one"))

		assert.Nil(t, err)
		assert.Equal(t, 2, len(repo.orgs))
		assert.Equal(t, "testdata/repo_one", repo.cfg.Get(task.Root))
		assert.True(t, repo.cfg.GetBool(task.Confirmation))
	})

	t.Run("open repository fails with non existent data directory", func(t *testing.T) {
		_, err := OpenRepository(filepath.Join("testdata", "repo_two"))
		assert.NotNil(t, err)
	})

	t.Run("open repository fails with invalid data directory", func(t *testing.T) {
		_, err := OpenRepository(filepath.Join("testdata", "invalid_repo"))
		assert.NotNil(t, err)
	})

	t.Run("open repository fails with invalid config file", func(t *testing.T) {
		_, err := OpenRepository(filepath.Join("testdata", "repo_invalid_config"))
		assert.NotNil(t, err)
	})
}

func TestGetOrganization(t *testing.T) {
	repo, err := OpenRepository(filepath.Join("testdata", "repo_one"))
	assert.Nil(t, err)

	t.Run("get valid organization should work", func(t *testing.T) {
		org, err := repo.GetOrg("Public")
		assert.Nil(t, err)

		a := assert.New(t)
		a.Equal("Public", org.Name)
		a.Equal(3, len(org.Users))
	})

	t.Run("get valid organization with invalid user should work", func(t *testing.T) {
		org, err := repo.GetOrg("Private")
		assert.Nil(t, err)

		a := assert.New(t)
		a.Equal("Private", org.Name)
		a.Equal(1, len(org.Users))
		a.Equal("peter", org.Users[0].Name)
		a.Equal("4e489103-04a9-4f7f-b676-ce8b45c6b634", org.Users[0].Key)
		a.NotNil(org.Users[0].Org)
	})

	t.Run("get invalid organization should fail", func(t *testing.T) {
		_, err := repo.GetOrg("PublicBAD")
		assert.NotNil(t, err)
	})
}

func TestNewOrganization(t *testing.T) {
	repo, err := OpenRepository(filepath.Join("testdata", "repo_one"))
	assert.Nil(t, err)

	t.Run("new organization works with valid data dir", func(t *testing.T) {
		before := len(repo.orgs)
		org, err := repo.NewOrg("delete-me")
		assert.Nil(t, err)
		defer func() {
			if err := os.RemoveAll(filepath.Join("testdata", "repo_one", "orgs", "delete-me")); err != nil {
				t.Fatal(err)
			}
		}()

		assert.Equal(t, "delete-me", org.Name)
		assert.Equal(t, before+1, len(repo.orgs))
	})

	t.Run("new organization fails if already exists", func(t *testing.T) {
		_, err := repo.NewOrg("Public")
		assert.NotNil(t, err)
	})

	t.Run("new organization fails if invalid name", func(t *testing.T) {
		_, err := repo.NewOrg("Pu/blic")
		assert.NotNil(t, err)
	})

}

func TestNewUser(t *testing.T) {
	repo, err := OpenRepository(filepath.Join("testdata", "repo_one"))
	assert.Nil(t, err)
	org, err := repo.NewOrg("delete-me")
	assert.Nil(t, err)
	defer func() {
		if err := os.RemoveAll(filepath.Join("testdata", "repo_one", "orgs", "delete-me")); err != nil {
			t.Fatal(err)
		}
	}()

	t.Run("add user works with valid organization", func(t *testing.T) {
		user, err := repo.AddUser("delete-me", "user_one")

		a := assert.New(t)
		a.Nil(err)
		a.NotNil(user)
		a.NotNil(user.Org)
		a.Equal("user_one", user.Name)
		a.Equal(org.Name, user.Org.Name)
		a.NotEmpty(user.Key)
	})

	t.Run("add user fails with infalid organization", func(t *testing.T) {
		_, err := repo.AddUser("invalid-org", "user_one")
		assert.NotNil(t, err)
	})

	t.Run("add user fails if user already exists", func(t *testing.T) {
		_, err := repo.AddUser("Public", "noeh")
		assert.NotNil(t, err)
	})
}

func TestAuthenticate(t *testing.T) {
	repo, err := OpenRepository(filepath.Join("testdata", "repo_one"))
	assert.Nil(t, err)

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
		u, err := repo.Authenticate(c.org, c.name, c.key)
		if c.success {
			assert.Nil(t, err)
			assert.Equal(t, u.Name, "noeh")
		} else {
			assert.NotNil(t, err)
			authErr, ok := err.(AuthenticationError)
			assert.True(t, ok)
			assert.NotEmpty(t, authErr.Msg)
			assert.NotEmpty(t, authErr.Error())
		}
	}
}

func TestGetData(t *testing.T) {
	repo, err := OpenRepository(filepath.Join("testdata", "repo_one"))
	assert.Nil(t, err)

	user, err := repo.Authenticate("Public", "noeh", "53938cd8-b72e-4c2a-9fb5-3cd183cf1fa7")
	assert.Nil(t, err)

	data, err := repo.Read(user)
	assert.Nil(t, err)
	assert.NotNil(t, data)

	user.Key = "invalid"
	data, err = repo.Read(user)
	assert.Nil(t, data)
	assert.NotNil(t, err)
}

func TestAppendData(t *testing.T) {
	defer func() {
		tx := filepath.Join("testdata", "repo_one", orgsFolder, "Public", usersFolder, "f793325d-c0d4-4f11-91d3-1388a02e727c", txFile)
		assert.NoError(t, os.Remove(tx))
	}()

	repo, err := OpenRepository(filepath.Join("testdata", "repo_one"))
	assert.Nil(t, err)

	user, err := repo.Authenticate("Public", "john", "f793325d-c0d4-4f11-91d3-1388a02e727c")
	assert.Nil(t, err)

	data := []string{
		"hello",
		"world",
	}
	assert.NoError(t, repo.Append(user, data))
	assert.NoError(t, repo.Append(user, data))
}

func TestCopy(t *testing.T) {
	dir := tempDir(t)
	source := tempFile(t)
	defer func() {
		source.Close()
		os.Remove(source.Name())
		os.RemoveAll(dir)
	}()

	t.Run("invalid source", func(t *testing.T) {
		assert.Error(t, copy("invalid", filepath.Join(dir, "bla")))
	})

	t.Run("invalid target", func(t *testing.T) {
		assert.Error(t, copy(source.Name(), filepath.Join(dir, "bla", "ble")))
	})

	t.Run("target is dir", func(t *testing.T) {
		assert.Error(t, copy(dir, filepath.Join(dir, "bla", "ble")))
	})

	t.Run("fail if source is not writable", func(t *testing.T) {
		defer assert.NoError(t, os.Chmod(source.Name(), 06400))

		assert.NoError(t, os.Chmod(source.Name(), 0000))
		assert.Error(t, copy(source.Name(), filepath.Join(dir, "bla")))
	})
}

func tempFile(t *testing.T) *os.File {
	t.Helper()

	file, err := ioutil.TempFile(os.TempDir(), "gotas")

	assert.Nil(t, err)

	return file
}

func tempDir(t *testing.T) string {
	t.Helper()

	dir, err := ioutil.TempDir(os.TempDir(), "gotas")

	assert.Nil(t, err)

	return dir
}
