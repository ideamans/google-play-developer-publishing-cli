package cmd

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ideamans/google-play-developer-publishing-cli/internal/api"
)

var (
	binaryFile          string
	binaryAckWarning    bool
	binaryDeviceTierCfg string
	binaryVersionCode   int64
	binaryFileType      string
	binaryFromJSON      string
	binaryReferences    int64
	binaryReplace       bool
)

// --- App bundles ---------------------------------------------------------------

var bundlesCmd = &cobra.Command{
	Use:   "bundles",
	Short: "Android App Bundles (.aab)",
}

var bundlesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List app bundles in the current edit",
	Long: `Lists the bundles the edit knows about. Note this reflects the edit, not the
full upload history of the app; a freshly created edit lists the bundles
already associated with the app.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runEditRead(cmd, func(ctx context.Context, e *edit) error {
			doc, err := e.c.GetMap(ctx, e.path("/bundles"))
			if err != nil {
				return err
			}
			if jsonFlag {
				return printJSON(doc)
			}
			bundles := doc.Docs("bundles")
			if len(bundles) == 0 {
				fmt.Println("No app bundles.")
				return nil
			}
			w := newTable("VERSION CODE", "SHA256")
			for _, b := range bundles {
				fmt.Fprintf(w, "%d\t%s\n", b.Int("versionCode"), b.Str("sha256"))
			}
			return w.Flush()
		})
	},
}

var bundlesUploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Upload an .aab to the current edit",
	Long: `Uploads an Android App Bundle. Files above 16 MiB use the resumable upload
protocol with progress on stderr.

The upload alone does not release anything: assign the resulting version code
to a track with "gplay tracks set", or use "gplay release" to do both in one
edit.`,
	Example: `  gplay bundles upload --file app-release.aab
  gplay bundles upload --file app-release.aab --no-commit`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runEdit(cmd, func(ctx context.Context, e *edit) error {
			code, err := uploadBundle(ctx, e, binaryFile)
			if err != nil {
				return err
			}
			reportUpload(e.c, "Uploaded %s as version code %d.\n", binaryFile, code)
			return nil
		})
	},
}

func uploadBundle(ctx context.Context, e *edit, file string) (int64, error) {
	path := e.path("/bundles")
	params := url.Values{}
	if binaryAckWarning {
		params.Set("ackBundleInstallationWarning", "true")
	}
	if binaryDeviceTierCfg != "" {
		params.Set("deviceTierConfigId", binaryDeviceTierCfg)
	}
	if len(params) > 0 {
		path += "?" + params.Encode()
	}
	doc, err := e.c.UploadFile(ctx, path, file, "application/octet-stream")
	if err != nil {
		return 0, err
	}
	return doc.Int("versionCode"), nil
}

// --- APKs -----------------------------------------------------------------------

var apksCmd = &cobra.Command{
	Use:   "apks",
	Short: "APKs (.apk)",
}

var apksListCmd = &cobra.Command{
	Use:   "list",
	Short: "List APKs in the current edit",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runEditRead(cmd, func(ctx context.Context, e *edit) error {
			doc, err := e.c.GetMap(ctx, e.path("/apks"))
			if err != nil {
				return err
			}
			if jsonFlag {
				return printJSON(doc)
			}
			apks := doc.Docs("apks")
			if len(apks) == 0 {
				fmt.Println("No APKs.")
				return nil
			}
			w := newTable("VERSION CODE", "SHA256")
			for _, a := range apks {
				fmt.Fprintf(w, "%d\t%s\n", a.Int("versionCode"), a.Doc("binary").Str("sha256"))
			}
			return w.Flush()
		})
	},
}

var apksUploadCmd = &cobra.Command{
	Use:     "upload",
	Short:   "Upload an .apk to the current edit",
	Example: `  gplay apks upload --file app-release.apk`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runEdit(cmd, func(ctx context.Context, e *edit) error {
			code, err := uploadAPK(ctx, e, binaryFile)
			if err != nil {
				return err
			}
			reportUpload(e.c, "Uploaded %s as version code %d.\n", binaryFile, code)
			return nil
		})
	},
}

func uploadAPK(ctx context.Context, e *edit, file string) (int64, error) {
	doc, err := e.c.UploadFile(ctx, e.path("/apks"), file, "application/vnd.android.package-archive")
	if err != nil {
		return 0, err
	}
	return doc.Int("versionCode"), nil
}

var apksExternalCmd = &cobra.Command{
	Use:   "add-externally-hosted",
	Short: "Register an externally hosted APK (enterprise-only)",
	Long: `Registers an APK that Google Play links to but does not host. This is only
available to organizations using Google Play for private (enterprise) apps.

The JSON body is an ExternallyHostedApk resource; it is long enough that
--from-json @file is the practical way to pass it.`,
	Example: `  gplay apks add-externally-hosted --from-json @externally-hosted.json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runEdit(cmd, func(ctx context.Context, e *edit) error {
			var apk map[string]any
			if err := jsonFromFlag(binaryFromJSON, &apk); err != nil {
				return err
			}
			body := map[string]any{"externallyHostedApk": apk}
			doc, err := e.c.Post(ctx, e.path("/apks/externallyHosted"), body)
			if err != nil {
				return err
			}
			return printJSON(doc)
		})
	},
}

// --- Deobfuscation (mapping) files ------------------------------------------------

var mappingCmd = &cobra.Command{
	Use:   "mapping",
	Short: "ProGuard/R8 mapping and native debug symbol files",
}

var mappingUploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Upload a mapping.txt or native debug symbols for a version code",
	Long: `Attaches a deobfuscation file to an already uploaded APK or bundle so that
crash stack traces in Play Console are readable.

  --type proguard    mapping.txt produced by R8/ProGuard
  --type nativeCode  a zip of native debug symbols (.so with symbols)`,
	Example: `  gplay mapping upload --version-code 42 --file mapping.txt
  gplay mapping upload --version-code 42 --type nativeCode --file symbols.zip`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runEdit(cmd, func(ctx context.Context, e *edit) error {
			return uploadMapping(ctx, e, binaryVersionCode, binaryFile, binaryFileType)
		})
	},
}

func uploadMapping(ctx context.Context, e *edit, versionCode int64, file, fileType string) error {
	if fileType == "" {
		fileType = "proguard"
	}
	path := e.path("/apks/%d/deobfuscationFiles/%s", versionCode, esc(fileType))
	if _, err := e.c.UploadFile(ctx, path, file, "application/octet-stream"); err != nil {
		return err
	}
	reportUpload(e.c, "Uploaded %s (%s) for version code %d.\n", file, fileType, versionCode)
	return nil
}

// --- Expansion files (OBB) ---------------------------------------------------------

var expansionCmd = &cobra.Command{
	Use:   "expansion",
	Short: "APK expansion files (OBB, legacy APK-only feature)",
	Long: `Expansion files apply to APK releases only; app bundles use Play Asset
Delivery instead. The type is "main" or "patch".`,
}

var expansionGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Show the expansion file attached to a version code",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runEditRead(cmd, func(ctx context.Context, e *edit) error {
			doc, err := e.c.GetMap(ctx, e.path("/apks/%d/expansionFiles/%s", binaryVersionCode, esc(binaryFileType)))
			if err != nil {
				return err
			}
			return printJSON(doc)
		})
	},
}

var expansionUploadCmd = &cobra.Command{
	Use:     "upload",
	Short:   "Upload an expansion file for a version code",
	Example: `  gplay expansion upload --version-code 42 --type main --file main.42.com.example.app.obb`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runEdit(cmd, func(ctx context.Context, e *edit) error {
			path := e.path("/apks/%d/expansionFiles/%s", binaryVersionCode, esc(binaryFileType))
			doc, err := e.c.UploadFile(ctx, path, binaryFile, "application/octet-stream")
			if err != nil {
				return err
			}
			fmt.Printf("Uploaded %s (%s) for version code %d.\n", binaryFile, binaryFileType, binaryVersionCode)
			if jsonFlag {
				return printJSON(doc)
			}
			return nil
		})
	},
}

var expansionReuseCmd = &cobra.Command{
	Use:     "reuse",
	Short:   "Point a version code at another version's expansion file",
	Long:    `Avoids re-uploading an unchanged OBB when publishing a new APK.`,
	Example: `  gplay expansion reuse --version-code 43 --type main --references-version 42`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runEdit(cmd, func(ctx context.Context, e *edit) error {
			path := e.path("/apks/%d/expansionFiles/%s", binaryVersionCode, esc(binaryFileType))
			body := map[string]any{"referencesVersion": binaryReferences}
			var err error
			if binaryReplace {
				_, err = e.c.Put(ctx, path, body)
			} else {
				_, err = e.c.Patch(ctx, path, body)
			}
			if err != nil {
				return err
			}
			fmt.Printf("Version code %d now references the %s expansion file of version code %d.\n",
				binaryVersionCode, binaryFileType, binaryReferences)
			return nil
		})
	},
}

// --- Internal app sharing -----------------------------------------------------------

var internalSharingCmd = &cobra.Command{
	Use:   "internal-sharing",
	Short: "Internal app sharing uploads (shareable link, no edit or review)",
	Long: `Uploads an artifact for internal app sharing and prints a download URL that
testers with access can install directly. This bypasses tracks, review and
version code rules entirely, so it is the quickest way to hand a build to
someone. It is not an edit operation.`,
}

var internalSharingUploadCmd = &cobra.Command{
	Use:     "upload",
	Short:   "Upload an .aab or .apk for internal app sharing",
	Example: `  gplay internal-sharing upload --file app-debug.apk`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			kind := "bundle"
			contentType := "application/octet-stream"
			if strings.EqualFold(filepath.Ext(binaryFile), ".apk") {
				kind = "apk"
				contentType = "application/vnd.android.package-archive"
			}
			path := "/applications/internalappsharing/" + esc(pkg) + "/artifacts/" + kind
			doc, err := c.UploadFile(ctx, path, binaryFile, contentType)
			if err != nil {
				return err
			}
			if jsonFlag {
				return printJSON(doc)
			}
			fmt.Println(doc.Str("downloadUrl"))
			return nil
		})
	},
}

func init() {
	bundlesUploadCmd.Flags().StringVar(&binaryFile, "file", "", "path to the .aab (required)")
	bundlesUploadCmd.Flags().BoolVar(&binaryAckWarning, "ack-installation-warning", false,
		"acknowledge that the bundle is installable only via Play (large bundles)")
	bundlesUploadCmd.Flags().StringVar(&binaryDeviceTierCfg, "device-tier-config-id", "", "device tier config id to build the bundle against")
	_ = bundlesUploadCmd.MarkFlagRequired("file")

	apksUploadCmd.Flags().StringVar(&binaryFile, "file", "", "path to the .apk (required)")
	_ = apksUploadCmd.MarkFlagRequired("file")
	apksExternalCmd.Flags().StringVar(&binaryFromJSON, "from-json", "", "ExternallyHostedApk JSON, or @file (required)")
	_ = apksExternalCmd.MarkFlagRequired("from-json")

	mappingUploadCmd.Flags().StringVar(&binaryFile, "file", "", "path to mapping.txt or the native symbols zip (required)")
	mappingUploadCmd.Flags().Int64Var(&binaryVersionCode, "version-code", 0, "version code the file belongs to (required)")
	mappingUploadCmd.Flags().StringVar(&binaryFileType, "type", "proguard", "proguard or nativeCode")
	_ = mappingUploadCmd.MarkFlagRequired("file")
	_ = mappingUploadCmd.MarkFlagRequired("version-code")

	for _, sub := range []*cobra.Command{expansionGetCmd, expansionUploadCmd, expansionReuseCmd} {
		sub.Flags().Int64Var(&binaryVersionCode, "version-code", 0, "APK version code (required)")
		sub.Flags().StringVar(&binaryFileType, "type", "main", "main or patch")
		_ = sub.MarkFlagRequired("version-code")
	}
	expansionUploadCmd.Flags().StringVar(&binaryFile, "file", "", "path to the .obb (required)")
	_ = expansionUploadCmd.MarkFlagRequired("file")
	expansionReuseCmd.Flags().Int64Var(&binaryReferences, "references-version", 0, "version code whose expansion file to reuse (required)")
	expansionReuseCmd.Flags().BoolVar(&binaryReplace, "replace", false, "send a full update (PUT), clearing omitted fields")
	_ = expansionReuseCmd.MarkFlagRequired("references-version")

	internalSharingUploadCmd.Flags().StringVar(&binaryFile, "file", "", "path to the .aab or .apk (required)")
	_ = internalSharingUploadCmd.MarkFlagRequired("file")

	addEditReadFlags(bundlesListCmd, apksListCmd, expansionGetCmd)
	addEditFlags(bundlesUploadCmd, apksUploadCmd, apksExternalCmd, mappingUploadCmd, expansionUploadCmd, expansionReuseCmd)

	bundlesCmd.AddCommand(bundlesListCmd, bundlesUploadCmd)
	apksCmd.AddCommand(apksListCmd, apksUploadCmd, apksExternalCmd)
	mappingCmd.AddCommand(mappingUploadCmd)
	expansionCmd.AddCommand(expansionGetCmd, expansionUploadCmd, expansionReuseCmd)
	internalSharingCmd.AddCommand(internalSharingUploadCmd)
	rootCmd.AddCommand(bundlesCmd, apksCmd, mappingCmd, expansionCmd, internalSharingCmd)
}
