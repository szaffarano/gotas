// +build linux

package repo

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
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
		assert.Equal(t, "testdata/repo_one", repo.Config.Get(Root))
		assert.True(t, repo.Config.GetBool(Confirmation))
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

func tempDir(t *testing.T) string {
	dir, err := ioutil.TempDir(os.TempDir(), "gotas")

	assert.Nil(t, err)

	return dir
}
