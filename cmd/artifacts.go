package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/ideamans/google-play-developer-publishing-cli/internal/api"
)

var (
	artifactVersionCode int64
	artifactDownloadID  string
	artifactVariantID   int64
	artifactOutput      string
	artifactFromJSON    string
)

// --- Generated APKs -------------------------------------------------------------

var generatedCmd = &cobra.Command{
	Use:   "generated-apks",
	Short: "APKs Google Play generated from an uploaded app bundle",
	Long: `Lists and downloads the split, standalone and universal APKs that Google Play
derived from an app bundle, signed with the app signing key. This is how you
get an installable artifact that matches what users receive.`,
}

var generatedListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List the APKs generated for a version code",
	Example: `  gplay generated-apks list --version-code 42`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			doc, err := c.GetMap(ctx, appPath(pkg, "/generatedApks/%d", artifactVersionCode))
			if err != nil {
				return err
			}
			if jsonFlag {
				return printJSON(doc)
			}
			groups := doc.Docs("generatedApks")
			if len(groups) == 0 {
				fmt.Println("No generated APKs. Google Play generates them a few minutes after a bundle upload.")
				return nil
			}
			w := newTable("SIGNING KEY", "KIND", "DOWNLOAD ID")
			for _, g := range groups {
				key := truncate(g.Str("certificateSha256Hash"), 16)
				for _, kind := range []string{"generatedSplitApks", "generatedStandaloneApks", "generatedAssetPackSlices", "generatedRecoveryModules"} {
					for _, apk := range g.Docs(kind) {
						fmt.Fprintf(w, "%s\t%s\t%s\n", key, kind, apk.Str("downloadId"))
					}
				}
				if universal := g.Doc("generatedUniversalApk"); universal != nil {
					fmt.Fprintf(w, "%s\t%s\t%s\n", key, "generatedUniversalApk", universal.Str("downloadId"))
				}
			}
			return w.Flush()
		})
	},
}

var generatedDownloadCmd = &cobra.Command{
	Use:     "download",
	Short:   "Download one generated APK",
	Example: `  gplay generated-apks download --version-code 42 --download-id <id> --output universal.apk`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			data, err := c.Download(ctx, appPath(pkg, "/generatedApks/%d/downloads/%s:download",
				artifactVersionCode, esc(artifactDownloadID)))
			if err != nil {
				return err
			}
			return writeArtifact(data, fmt.Sprintf("%s-%d-%s.apk", pkg, artifactVersionCode, artifactDownloadID))
		})
	},
}

// --- System APKs ------------------------------------------------------------------

var systemAPKsCmd = &cobra.Command{
	Use:   "system-apks",
	Short: "System image APK variants (device manufacturers only)",
	Long: `Creates and downloads APK variants suitable for preloading into a system
image. This is only available to accounts that Google has enabled for system
app distribution.`,
}

var systemAPKsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List the variants created for a version code",
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			doc, err := c.GetMap(ctx, appPath(pkg, "/systemApks/%d/variants", artifactVersionCode))
			if err != nil {
				return err
			}
			return printJSON(doc)
		})
	},
}

var systemAPKsGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Show one variant",
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			doc, err := c.GetMap(ctx, appPath(pkg, "/systemApks/%d/variants/%d", artifactVersionCode, artifactVariantID))
			if err != nil {
				return err
			}
			return printJSON(doc)
		})
	},
}

var systemAPKsCreateCmd = &cobra.Command{
	Use:     "create",
	Short:   "Create a variant for a device specification",
	Long:    `Takes a Variant body describing the device spec (SDK version, ABIs, screen density, locales).`,
	Example: `  gplay system-apks create --version-code 42 --from-json @variant.json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			var body map[string]any
			if err := jsonFromFlag(artifactFromJSON, &body); err != nil {
				return err
			}
			doc, err := c.Post(ctx, appPath(pkg, "/systemApks/%d/variants", artifactVersionCode), body)
			if err != nil {
				return err
			}
			return printJSON(doc)
		})
	},
}

var systemAPKsDownloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Download a variant as an APK",
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			data, err := c.Download(ctx, appPath(pkg, "/systemApks/%d/variants/%d:download",
				artifactVersionCode, artifactVariantID))
			if err != nil {
				return err
			}
			return writeArtifact(data, fmt.Sprintf("%s-%d-variant-%d.apk", pkg, artifactVersionCode, artifactVariantID))
		})
	},
}

// writeArtifact writes downloaded bytes to --output or a derived file name.
func writeArtifact(data []byte, defaultName string) error {
	path := artifactOutput
	if path == "" {
		path = defaultName
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil && filepath.Dir(path) != "." {
		return err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return err
	}
	fmt.Printf("Wrote %s (%d bytes).\n", path, len(data))
	return nil
}

func init() {
	versionScoped := []*cobra.Command{
		generatedListCmd, generatedDownloadCmd,
		systemAPKsListCmd, systemAPKsGetCmd, systemAPKsCreateCmd, systemAPKsDownloadCmd,
	}
	for _, sub := range versionScoped {
		sub.Flags().Int64Var(&artifactVersionCode, "version-code", 0, "version code (required)")
		_ = sub.MarkFlagRequired("version-code")
	}
	for _, sub := range []*cobra.Command{systemAPKsGetCmd, systemAPKsDownloadCmd} {
		sub.Flags().Int64Var(&artifactVariantID, "variant", 0, "variant id (required)")
		_ = sub.MarkFlagRequired("variant")
	}
	generatedDownloadCmd.Flags().StringVar(&artifactDownloadID, "download-id", "", "download id from \"generated-apks list\" (required)")
	_ = generatedDownloadCmd.MarkFlagRequired("download-id")
	for _, sub := range []*cobra.Command{generatedDownloadCmd, systemAPKsDownloadCmd} {
		sub.Flags().StringVarP(&artifactOutput, "output", "o", "", "output file (default: derived from the package and version code)")
	}
	systemAPKsCreateCmd.Flags().StringVar(&artifactFromJSON, "from-json", "", "Variant JSON, or @file (required)")
	_ = systemAPKsCreateCmd.MarkFlagRequired("from-json")

	generatedCmd.AddCommand(generatedListCmd, generatedDownloadCmd)
	systemAPKsCmd.AddCommand(systemAPKsListCmd, systemAPKsGetCmd, systemAPKsCreateCmd, systemAPKsDownloadCmd)
	rootCmd.AddCommand(generatedCmd, systemAPKsCmd)
}
