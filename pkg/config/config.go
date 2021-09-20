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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

type Flags struct {
	ConfigFile string
	Quiet      bool
	Verbose    bool
	DataDir    string
}

type config struct {
	Flags
	Confirmation bool
	Log          string
	Pid          struct {
		File string
	}
	Queue struct {
		Size int
	}
	Request struct {
		Limit int
	}
	Root   string
	Trust  string
	Client struct {
		Cert string
		Key  string
	}
	Server struct {
		BindAddress string
		Key         string
		Cert        string
		Crl         string
	}
	Ca struct {
		Cert string
	}
}

var conf config

func InitConfig(flags Flags) error {
	// log configuration, @TODO be aware of `config.Log`
	log.SetHandler(cli.Default)
	if flags.Verbose {
		log.SetLevel(log.DebugLevel)
	} else if flags.Quiet {
		log.SetLevel(log.ErrorLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	// configuration file lookup:
	//   1. --config flag
	//   2. if --config is not defined:
	//   2.1 if --data is defined, $data/config
	//   2.2 if --data is not defined, $TASKDATA/config
	//   3. Otherwise fail
	if flags.ConfigFile == "" {
		if flags.DataDir == "" {
			value, ok := os.LookupEnv("TASKDDATA")
			if !ok {
				return fmt.Errorf("you have to define either $TASKDDATA variable or data flag")
			}
			flags.DataDir = value
		}
		flags.ConfigFile = filepath.Join(flags.DataDir, "config")
	}

	content, err := ioutil.ReadFile(flags.ConfigFile)
	if err != nil {
		return errors.Wrapf(err, "Error opening config configuration: %s", conf.ConfigFile)
	}
	err = yaml.Unmarshal(content, &conf)
	if err != nil {
		return errors.Wrapf(err, "Error reading config configuration: %s", conf.ConfigFile)
	}

	overrideFromEnvironment()

	conf.ConfigFile = flags.ConfigFile
	conf.Verbose = flags.Verbose
	conf.Quiet = flags.Quiet
	conf.DataDir = flags.DataDir

	log.Debugf("Config file initialized: %s", conf.ConfigFile)
	return nil
}

func Get() *config {
	return &conf
}

func clearConfig() {
	conf = config{}
}

func overrideFromEnvironment() {
	// @TODO read environment variables to override configurations
	// corresponds to `--NAME=VALUE   Temporary configuration override` taskd flags
}
