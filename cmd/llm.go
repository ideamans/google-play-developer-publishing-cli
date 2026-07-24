package cmd

import (
	"github.com/ideamans/go-llm-cli-kit/llmcmd"
	"github.com/spf13/cobra"

	"github.com/ideamans/google-play-developer-publishing-cli/internal/llmdocs"
)

// LLMConfig describes the `gplay llm` subcommand. main uses it to keep the
// deprecated --llm flag working at any position on the command line.
func LLMConfig() llmcmd.Config {
	return llmcmd.Config{Docs: llmdocs.Docs()}
}

func init() {
	llmcmd.AddTo(rootCmd, LLMConfig())

	rootCmd.Args = cobra.NoArgs
	rootCmd.RunE = func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	}
}
