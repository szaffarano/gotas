package config

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	validConfig = `
# comment
confirmation=1
ip.log=on

## more comments
log =    /tmp/taskd.log
pid.file=/tmp/taskd.pid
queue.size=10
request.limit=1048576
root=/path/to/taskddata
server=localhost:53589
trust=strict
verbose=1
server.key=/path/to/server.key.pem
server.cert=/path/to/server.cert.pem
ca.cert=/path/to/ca.cert.pem`
	invalidConfig = validConfig + "\n invalid format"
)

func TestConfig(t *testing.T) {
	validConfigPath, validConfigDir := mockConfig(t, validConfig)
	invalidConfigPath, invalidConfigDir := mockConfig(t, invalidConfig)
	_, emptyDataDir := mockConfig(t, "")

	defer os.RemoveAll(validConfigDir)
	defer os.RemoveAll(invalidConfigDir)

	t.Run("load valid config", func(t *testing.T) {
		cfg, err := Load(validConfigPath)

		assert.Nil(t, err)
		assertConfig(t, cfg)
	})

	t.Run("fail with invalid config", func(t *testing.T) {
		_, err := Load(invalidConfigPath)

		assert.NotNil(t, err)
	})

	t.Run("fail with non existent config", func(t *testing.T) {
		_, err := Load(filepath.Join("bad", invalidConfigPath))

		assert.NotNil(t, err)
	})

	t.Run("init new config", func(t *testing.T) {
		_, err := New(filepath.Join(emptyDataDir, "some-config"))
		assert.Nil(t, err)
	})

	t.Run("init new config fails with invalid dir", func(t *testing.T) {
		_, err := New(filepath.Join("some", emptyDataDir, "some-config"))
		assert.NotNil(t, err)
	})

	t.Run("save config", func(t *testing.T) {
		cfg, err := Load(validConfigPath)

		assert.Nil(t, err)

		err = Save(cfg)

		assert.Nil(t, err)
	})

	t.Run("save config fail if config was not initialized", func(t *testing.T) {
		cfg := Config{}
		err := Save(cfg)
		assert.NotNil(t, err)
	})

	t.Run("setters and getters", func(t *testing.T) {
		cfg, err := New(filepath.Join(emptyDataDir, "some-config"))

		assert.Nil(t, err)

		cfg.Set("str", "hello")
		cfg.SetInt("num", 1)
		cfg.SetBool("bool", true)

		assert.Equal(t, "hello", cfg.Get("str"))
		assert.Equal(t, 1, cfg.GetInt("num"))
		assert.Equal(t, true, cfg.GetBool("bool"))

	})

}

func assertConfig(t *testing.T, conf Config) {
	t.Helper()
	assert := assert.New(t)

	assert.Equal(true, conf.GetBool("confirmation"))
	assert.Equal("/tmp/taskd.log", conf.Get("log"))
	assert.Equal("/tmp/taskd.pid", conf.Get("pid.file"))
	assert.Equal(10, conf.GetInt("queue.size"))
	assert.Equal(1048576, conf.GetInt("request.limit"))
	assert.Equal("/path/to/taskddata", conf.Get("root"))
	assert.Equal("strict", conf.Get("trust"))
	assert.Equal(true, conf.GetBool("verbose"))
	assert.Equal("localhost:53589", conf.Get("server"))
	assert.Equal("/path/to/server.key.pem", conf.Get("server.key"))
	assert.Equal("/path/to/server.cert.pem", conf.Get("server.cert"))
	assert.Equal("/path/to/ca.cert.pem", conf.Get("ca.cert"))
}

func mockConfig(t *testing.T, content string) (string, string) {
	t.Helper()

	dir, err := ioutil.TempDir(os.TempDir(), "gotas")
	if err != nil {
		t.Error(err.Error())
	}
	configPath := filepath.Join(dir, "config")

	if content == "" {
		return "", dir
	}
	file, err := os.Create(configPath)
	if err != nil {
		t.Error(err.Error())
	}
	_, err = file.Write([]byte(content))
	if err != nil {
		t.Error(err.Error())
	}

	return configPath, dir
}
