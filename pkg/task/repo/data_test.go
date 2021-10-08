package repo

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/szaffarano/gotas/pkg/config"
)

func TestGetData(t *testing.T) {
	auth := validAuthenticator(t)
	ra := validReadAppender(t)

	user, err := auth.Authenticate("Public", "noeh", "53938cd8-b72e-4c2a-9fb5-3cd183cf1fa7")
	assert.Nil(t, err)

	data, err := ra.Read(user)
	assert.Nil(t, err)
	assert.NotNil(t, data)

	user.Key = "invalid"
	data, err = ra.Read(user)
	assert.Nil(t, data)
	assert.NotNil(t, err)
}

func TestAppendData(t *testing.T) {
	auth := validAuthenticator(t)
	ra := validReadAppender(t)

	defer func() {
		tx := filepath.Join("testdata", "repo_one", orgsFolder, "Public", usersFolder, "f793325d-c0d4-4f11-91d3-1388a02e727c", txFile)
		assert.NoError(t, os.Remove(tx))
	}()

	user, err := auth.Authenticate("Public", "john", "f793325d-c0d4-4f11-91d3-1388a02e727c")
	assert.Nil(t, err)

	data := []string{
		"hello",
		"world",
	}
	assert.NoError(t, ra.Append(user, data))
	assert.NoError(t, ra.Append(user, data))
}

func TestCopy(t *testing.T) {
	dir := tempDir(t)
	src := tempFile(t)
	defer func() {
		src.Close()
		os.Remove(src.Name())
		os.RemoveAll(dir)
	}()

	t.Run("invalid source", func(t *testing.T) {
		assert.Error(t, (source("invalid")).copy(filepath.Join(dir, "bla")))
	})

	t.Run("invalid target", func(t *testing.T) {
		assert.Error(t, (source(src.Name())).copy(filepath.Join(dir, "bla", "ble")))
	})

	t.Run("target is dir", func(t *testing.T) {
		assert.Error(t, (source(dir)).copy(filepath.Join(dir, "bla", "ble")))
	})

	t.Run("fail if source is not writable", func(t *testing.T) {
		defer assert.NoError(t, os.Chmod(src.Name(), 06400))

		assert.NoError(t, os.Chmod(src.Name(), 0000))
		assert.Error(t, (source(src.Name())).copy(filepath.Join(dir, "bla")))
	})
}

func validReadAppender(t *testing.T) ReadAppender {
	t.Helper()

	configFilePath := filepath.Join("testdata", "repo_one", "config")
	cfg, err := config.Load(configFilePath)
	if err != nil {
		assert.FailNow(t, err.Error())
	}
	return NewDefaultReadAppender(cfg)
}

func tempFile(t *testing.T) *os.File {
	t.Helper()

	file, err := ioutil.TempFile(os.TempDir(), "gotas")

	assert.Nil(t, err)

	return file
}
