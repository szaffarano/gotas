// go:build !windows
package repo

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewRepositoryOnlyLinux(t *testing.T) {
	t.Run("new repository fails when invalid permission for read dir", func(t *testing.T) {
		baseDir := tempDir(t)
		defer os.RemoveAll(baseDir)

		err := os.Chmod(baseDir, 0400)
		assert.Nil(t, err)

		_, err = NewRepository(baseDir, defaultConfig)

		assert.NotNil(t, err)
	})

	t.Run("new repository fails when invalid invalid permission", func(t *testing.T) {
		baseDir := tempDir(t)
		defer os.RemoveAll(baseDir)

		err := os.Chmod(baseDir, 0000)
		assert.Nil(t, err)

		_, err = NewRepository(baseDir, defaultConfig)

		assert.NotNil(t, err)
	})
}
