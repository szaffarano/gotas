package repo

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var defaultConfig = map[string]string{
	"Confirmation": "true",
	"Log":          filepath.Join(os.TempDir(), "taskd.log"),
	"PidFile":      filepath.Join(os.TempDir(), "taskd.pid"),
	"QueueSize":    "10",
	"RequestLimit": "1048576",
	"Root":         "dataDir",
	"Trust":        "strict",
	"Verbose":      "true",
}

func TestNewRepository(t *testing.T) {
	t.Run("new repository works with empty directory", func(t *testing.T) {
		baseDir := tempDir(t)
		defer os.RemoveAll(baseDir)

		repo, err := NewRepository(baseDir, defaultConfig)

		assert.Nil(t, err)
		assert.NotNil(t, repo)
	})

	t.Run("new repository fails with non empty directory", func(t *testing.T) {
		baseDir := tempDir(t)
		defer os.RemoveAll(baseDir)

		repo, err := NewRepository(baseDir, defaultConfig)

		assert.Nil(t, err)
		assert.NotNil(t, repo)

		_, err = NewRepository(baseDir, defaultConfig)
		assert.NotNil(t, err)
	})

	t.Run("new repository fails with non existent repository", func(t *testing.T) {
		_, err := NewRepository("fake dir", defaultConfig)
		assert.NotNil(t, err)
	})

	t.Run("new repository fails when invalid datadir", func(t *testing.T) {
		baseDir := tempDir(t)
		defer os.RemoveAll(baseDir)

		filePath := filepath.Join(baseDir, "file")
		file, err := os.Create(filePath)
		assert.Nil(t, err)
		defer file.Close()

		_, err = NewRepository(filePath, defaultConfig)
		assert.NotNil(t, err)
	})
}

func TestOpenRepository(t *testing.T) {
	t.Run("open repository works with existing repository", func(t *testing.T) {
		repo, err := OpenRepository(filepath.Join("testdata", "repo_one"))

		assert.Nil(t, err)
		assert.Equal(t, 2, len(repo.orgs))
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

func TestDelOrganization(t *testing.T) {
	tempRepo := tempDir(t)
	repoOne := filepath.Join("testdata", "repo_one")
	defer os.RemoveAll(tempRepo)

	copy(t, repoOne, tempRepo)

	repo, err := OpenRepository(tempRepo)
	assert.Nil(t, err)

	t.Run("removes existing organization", func(t *testing.T) {
		before := len(repo.orgs)
		err := repo.DelOrg("Public")
		assert.Nil(t, err)

		assert.Equal(t, before-1, len(repo.orgs))
	})

	t.Run("removes organization fails if does not exists", func(t *testing.T) {
		err := repo.DelOrg("Public")
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

func TestDelUser(t *testing.T) {
	tempRepo := tempDir(t)
	repoOne := filepath.Join("testdata", "repo_one")
	defer os.RemoveAll(tempRepo)

	copy(t, repoOne, tempRepo)

	repo, err := OpenRepository(tempRepo)
	assert.Nil(t, err)

	t.Run("del existing user", func(t *testing.T) {
		err := repo.DelUser("Public", "53938cd8-b72e-4c2a-9fb5-3cd183cf1fa7")

		assert.NoError(t, err)
	})

	t.Run("del non org user fails", func(t *testing.T) {
		err := repo.DelUser("invalid", "53938cd8-b72e-4c2a-9fb5-3cd183cf1fa7")
		assert.Error(t, err)
	})

	t.Run("del non existent user fails", func(t *testing.T) {
		err := repo.DelUser("Public", "invalid")
		assert.Error(t, err)
	})

}

func tempDir(t *testing.T) string {
	t.Helper()

	dir, err := ioutil.TempDir(os.TempDir(), "gotas")

	assert.Nil(t, err)

	return dir
}

func copy(t *testing.T, source, destination string) {
	var err error = filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		var relPath string = strings.Replace(path, source, "", 1)
		if relPath == "" {
			return nil
		}
		if info.IsDir() {
			return os.Mkdir(filepath.Join(destination, relPath), 0755)
		}
		data, er := ioutil.ReadFile(filepath.Join(source, relPath))
		if er != nil {
			return er
		}
		return ioutil.WriteFile(filepath.Join(destination, relPath), data, 0664)
	})
	if err != nil {
		assert.FailNow(t, err.Error())
	}
}
