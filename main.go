package main

import (
	"github.com/szaffarano/gotas/cmd"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
	builtBy = "unknown"
)

func main() {
	cmd.Execute(cmd.Version{Version: version, Commit: commit, Date: date, BuiltBy: builtBy})
}
