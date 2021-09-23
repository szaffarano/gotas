// +build linux

package task

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/tj/assert"
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
	t.Run("open repository fails because is not implemented", func(t *testing.T) {
		_, err := OpenRepository("")
		assert.NotNil(t, err)
	})

	t.Run("open repository works with existing repository", func(t *testing.T) {
	})

	t.Run("open repository fails with non existent data directory", func(t *testing.T) {
	})

	t.Run("open repository fails with invalid data directory", func(t *testing.T) {
	})

	t.Run("open repository fails with invalid config file", func(t *testing.T) {
	})
}

func tempDir(t *testing.T) string {
	dir, err := ioutil.TempDir(os.TempDir(), "gotas")

	assert.Nil(t, err)

	return dir
}
