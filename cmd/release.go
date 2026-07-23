package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var (
	releaseAAB      []string
	releaseAPK      []string
	releaseMapping  string
	releaseSymbols  string
	releaseValidate bool
)

var releaseCmd = &cobra.Command{
	Use:   "release",
	Short: "Upload a build and release it to a track in one edit",
	Long: `Does the whole publishing round trip in a single edit: upload the artifacts,
attach the mapping file, write the release into the track, and commit.

Version codes come from the uploads; pass --version-code as well to include
builds that were uploaded earlier (for example one APK per ABI).

The status defaults to "inProgress" when --user-fraction is given and
"completed" otherwise. Use --status draft to stage a release without
publishing it.`,
	Example: `  gplay release --track internal --aab app-release.aab
  gplay release --track production --aab app-release.aab --mapping mapping.txt \
    --user-fraction 0.1 --notes ja=@notes-ja.txt --notes en-US=@notes-en.txt
  gplay release --track production --aab app-release.aab --status draft --dry-run`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(releaseAAB) == 0 && len(releaseAPK) == 0 && len(trackVersionCodes) == 0 {
			return fmt.Errorf("nothing to release: pass --aab, --apk or --version-code")
		}
		return runEdit(cmd, func(ctx context.Context, e *edit) error {
			var uploaded []int64
			for _, file := range releaseAAB {
				code, err := uploadBundle(ctx, e, file)
				if err != nil {
					return err
				}
				reportUpload(e.c, "Uploaded %s as version code %d.\n", file, code)
				uploaded = append(uploaded, code)
			}
			for _, file := range releaseAPK {
				code, err := uploadAPK(ctx, e, file)
				if err != nil {
					return err
				}
				reportUpload(e.c, "Uploaded %s as version code %d.\n", file, code)
				uploaded = append(uploaded, code)
			}
			trackVersionCodes = append(trackVersionCodes, uploaded...)
			if e.c.DryRun {
				// The uploads were skipped, so the API returned no version code;
				// the placeholder 0 keeps the preview readable.
				fmt.Fprintln(cmd.ErrOrStderr(),
					"DRY-RUN: version code 0 stands in for the code the upload would have returned")
			}

			if releaseMapping != "" || releaseSymbols != "" {
				if len(uploaded) != 1 {
					return fmt.Errorf("--mapping/--symbols needs exactly one uploaded artifact to attach to; " +
						`upload the rest separately with "gplay mapping upload --version-code N"`)
				}
				if releaseMapping != "" {
					if err := uploadMapping(ctx, e, uploaded[0], releaseMapping, "proguard"); err != nil {
						return err
					}
				}
				if releaseSymbols != "" {
					if err := uploadMapping(ctx, e, uploaded[0], releaseSymbols, "nativeCode"); err != nil {
						return err
					}
				}
			}

			release, err := buildRelease(cmd)
			if err != nil {
				return err
			}
			if err := writeTrack(ctx, e, trackName, release, trackKeepExisting); err != nil {
				return err
			}
			if releaseValidate && !e.c.DryRun {
				if _, err := e.c.Do(ctx, "POST", e.path(":validate"), nil); err != nil {
					return fmt.Errorf("validate edit: %w", err)
				}
				fmt.Println("Edit validated.")
			}
			return nil
		})
	},
}

func init() {
	releaseCmd.Flags().StringVar(&trackName, "track", "", "target track: internal, alpha, beta, production or a closed track (required)")
	_ = releaseCmd.MarkFlagRequired("track")
	releaseCmd.Flags().StringArrayVar(&releaseAAB, "aab", nil, "app bundle to upload; repeatable")
	releaseCmd.Flags().StringArrayVar(&releaseAPK, "apk", nil, "APK to upload; repeatable")
	releaseCmd.Flags().StringVar(&releaseMapping, "mapping", "", "ProGuard/R8 mapping.txt for the uploaded artifact")
	releaseCmd.Flags().StringVar(&releaseSymbols, "symbols", "", "native debug symbols zip for the uploaded artifact")
	releaseCmd.Flags().BoolVar(&releaseValidate, "validate", false, "validate the edit before committing")
	addReleaseFlags(releaseCmd)
	addEditFlags(releaseCmd)
	rootCmd.AddCommand(releaseCmd)
}
