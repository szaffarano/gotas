package main

import (
	"bytes"
	"encoding/json"

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
		panic("Error building version")
	}

	return buffer.String()
}
