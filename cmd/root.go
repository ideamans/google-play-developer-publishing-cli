package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	profileFlag string
	packageFlag string
	dryRun      bool
	jsonFlag    bool
)

var rootCmd = &cobra.Command{
	Use:           "gplay",
	Short:         "Google Play Developer Publishing API CLI (Android Publisher v3)",
	SilenceUsage:  true,
	SilenceErrors: true,
}

// PluginVersion is the released version of this CLI. It is also the version
// recorded in plugins/google-play-developer-publishing-cli/.claude-plugin/plugin.json —
// a test enforces that the two agree, and the release workflow enforces that
// both agree with the git tag. Bump it in the same commit as the tag.
const PluginVersion = "0.2.0"

// Root returns the assembled command tree without executing it, so the catalog
// generator works from the same definition main dispatches on.
func Root() *cobra.Command { return rootCmd }

func Execute(version string) {
	rootCmd.Version = version
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&profileFlag, "profile", "", "profile name (env: GPLAY_PROFILE)")
	rootCmd.PersistentFlags().StringVarP(&packageFlag, "package", "p", "", "Android package name, e.g. com.example.app (env: GPLAY_PACKAGE)")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "print mutating requests instead of sending them")
	rootCmd.PersistentFlags().BoolVar(&jsonFlag, "json", false, "print raw API JSON instead of a table")
}
