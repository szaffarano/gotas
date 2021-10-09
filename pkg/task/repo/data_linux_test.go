// go:build !windows
package repo

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCopyOnlyLinux(t *testing.T) {
	dir := tempDir(t)
	src := tempFile(t)
	defer func() {
		src.Close()
		os.Remove(src.Name())
		os.RemoveAll(dir)
	}()

	t.Run("fail if source is not writable", func(t *testing.T) {
		defer assert.NoError(t, os.Chmod(src.Name(), 06400))

		assert.NoError(t, os.Chmod(src.Name(), 0000))
		assert.Error(t, (source(src.Name())).copy(filepath.Join(dir, "bla")))
	})
}
