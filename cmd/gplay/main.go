package main

import (
	"fmt"
	"os"

	"github.com/ideamans/go-llm-cli-kit/llmcmd"

	"github.com/ideamans/google-play-developer-publishing-cli/cmd"
)

// Set by GoReleaser via -ldflags "-X main.version=... -X main.commit=... -X main.date=...".
var (
	version = cmd.PluginVersion
	commit  = "none"
	date    = "unknown"
)

func main() {
	// --llm anywhere on the command line prints the reference and exits,
	// bypassing cobra so it keeps working regardless of subcommand position.
	// Deprecated in favour of `gplay llm`, but removing it would break every
	// existing caller.
	if handled, err := llmcmd.HandleLegacy(os.Args[1:], cmd.LLMConfig(), os.Stdout); handled {
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
		return
	}

	cmd.Execute(fmt.Sprintf("%s (commit %s, built %s)", version, commit, date))
}
