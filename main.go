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

package main

import (
	"bytes"
	"encoding/json"
	"log"

	"github.com/szaffarano/gotas/cmd"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
	builtBy = "unknown"
)

func main() {
	cmd.Execute(getVersion())
}

func getVersion() string {
	version := struct {
		Version string `json:",omitempty"`
		Commit  string `json:",omitempty"`
		Date    string `json:",omitempty"`
		BuiltBy string `json:",omitempty"`
	}{version, commit, date, builtBy}

	var buffer bytes.Buffer
	if err := json.NewEncoder(&buffer).Encode(version); err != nil {
		log.Fatal("Error building version")
	}

	return buffer.String()
}