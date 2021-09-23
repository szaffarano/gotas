package task

import (
	"io/ioutil"
	"os"
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

}

func TestOpenRepository(t *testing.T) {
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
