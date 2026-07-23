package main

import (
	"fmt"

	"github.com/ideamans/google-play-developer-publishing-cli/cmd"
)

// Set by GoReleaser via -ldflags "-X main.version=... -X main.commit=... -X main.date=...".
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cmd.Execute(fmt.Sprintf("%s (commit %s, built %s)", version, commit, date))
}
