// Copyright © 2021 Sebastián Zaffarano <sebas@zaffarano.com.ar>.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/apex/log"
	"github.com/apex/log/handlers/memory"
	"github.com/stretchr/testify/assert"
)

var (
	validConfig = `
---
confirmation: true
ip:
  log: on
log: /tmp/taskd.log
pid:
  file: /tmp/taskd.pid
queue:
  size: 10
request:
  limit: 1048576
root: /tmp/dummy
trust: strict
verbose: false
client:
  cert: /path/to/cert
  key: /path/to/key
server:
  bindaddress: host:1234
  key: /path/to/key
  cert: /path/to/cert
  crl: /path/to/crl
ca:
  cert: /path/to/ca
  `
	invalidConfig = validConfig + "\n invalid format"
)

func TestConfig(t *testing.T) {
	validConfigPath, validConfigDir := mockConfig(t, validConfig)
	invalidConfigPath, invalidConfigDir := mockConfig(t, invalidConfig)
	_, nonExistentDataDir := mockConfig(t, "")

	defer os.RemoveAll(validConfigDir)

	t.Run("configure works with valid --config flag", func(t *testing.T) {
		defer clearConfig()
		if err := InitConfig(Flags{ConfigFile: validConfigPath}); err != nil {
			t.Errorf("Unexpected error: %w", err)
		}

		conf := Get()

		assertConfig(t, conf)
	})

	t.Run("configure set quiet log level", func(t *testing.T) {
		defer clearConfig()

		if err := InitConfig(Flags{ConfigFile: validConfigPath, Quiet: true}); err != nil {
			t.Errorf("Unexpected error: %w", err)
		}

		conf := Get()

		memoryHandler := memory.New()
		log.SetHandler(memoryHandler)

		before := len(memoryHandler.Entries)

		assert.True(t, conf.Quiet)

		log.Debug("log something")
		assert.Equal(t, before, len(memoryHandler.Entries))

		log.Error("log something")
		assert.Equal(t, before+1, len(memoryHandler.Entries))
	})

	t.Run("configure set debug log level", func(t *testing.T) {
		defer clearConfig()

		if err := InitConfig(Flags{ConfigFile: validConfigPath, Verbose: true}); err != nil {
			t.Errorf("Unexpected error: %w", err)
		}

		conf := Get()

		memoryHandler := memory.New()
		log.SetHandler(memoryHandler)

		before := len(memoryHandler.Entries)

		assert.True(t, conf.Verbose)

		log.Debug("log something")
		assert.Equal(t, before+1, len(memoryHandler.Entries))
	})

	t.Run("configure set debug log level even if quiet is set as well", func(t *testing.T) {
		defer clearConfig()

		if err := InitConfig(Flags{ConfigFile: validConfigPath, Verbose: true, Quiet: true}); err != nil {
			t.Errorf("Unexpected error: %w", err)
		}

		conf := Get()

		memoryHandler := memory.New()
		log.SetHandler(memoryHandler)

		before := len(memoryHandler.Entries)

		assert.True(t, conf.Verbose)

		log.Debug("log something")
		assert.Equal(t, before+1, len(memoryHandler.Entries))
	})

	t.Run("configure fails with invalid --config flag", func(t *testing.T) {
		defer clearConfig()
		if err := InitConfig(Flags{ConfigFile: invalidConfigPath}); err == nil {
			t.Error("Error expected")
		}
	})

	t.Run("configure fails with non-existent --data flag", func(t *testing.T) {
		defer clearConfig()
		if err := InitConfig(Flags{DataDir: nonExistentDataDir}); err == nil {
			t.Error("Error expected")
		}
	})

	t.Run("configure works with valid --data flag", func(t *testing.T) {
		defer clearConfig()
		if err := InitConfig(Flags{DataDir: validConfigDir}); err != nil {
			t.Errorf("Unexpected error: %w", err)
		}

		conf := Get()

		assertConfig(t, conf)
	})

	t.Run("configure fails with invalid --data flag", func(t *testing.T) {
		defer clearConfig()
		if err := InitConfig(Flags{DataDir: invalidConfigDir}); err == nil {
			t.Error("Error expected")
		}
	})

	t.Run("configure works with TASKDDATA environment var", func(t *testing.T) {
		defer clearConfig()
		if err := os.Setenv("TASKDDATA", validConfigDir); err != nil {
			t.Errorf("Error setting environment variable: %w", err)
		}
		defer func() {
			if err := os.Unsetenv("TASKDDATA"); err != nil {
				t.Errorf("Error unsetting environment variable: %w", err)
			}
		}()

		if err := InitConfig(Flags{}); err != nil {
			t.Errorf("Unexpected error: %w", err)
		}

		conf := Get()

		assertConfig(t, conf)
	})

	t.Run("configure fails with neither --config nor --data", func(t *testing.T) {
		defer clearConfig()
		if err := InitConfig(Flags{}); err == nil {
			t.Error("Error expected")
		}
	})

}

func assertConfig(t *testing.T, conf *config) {
	t.Helper()
	assert := assert.New(t)

	assert.Equal(true, conf.Confirmation)
	assert.Equal("/tmp/taskd.log", conf.Log)
	assert.Equal("/tmp/taskd.pid", conf.Pid.File)
	assert.Equal(10, conf.Queue.Size)
	assert.Equal(1048576, conf.Request.Limit)
	assert.Equal("/tmp/dummy", conf.Root)
	assert.Equal("strict", conf.Trust)
	assert.Equal(false, conf.Verbose)
	assert.Equal("/path/to/cert", conf.Client.Cert)
	assert.Equal("/path/to/key", conf.Client.Key)
	assert.Equal("host:1234", conf.Server.BindAddress)
	assert.Equal("/path/to/key", conf.Server.Key)
	assert.Equal("/path/to/key", conf.Server.Key)
	assert.Equal("/path/to/ca", conf.Ca.Cert)
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
