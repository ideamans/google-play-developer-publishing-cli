package cmd

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/ideamans/google-play-developer-publishing-cli/internal/api"
)

var editsCmd = &cobra.Command{
	Use:   "edits",
	Short: "Manage edit transactions",
	Long: `Every change to an app's store listing, tracks or binaries happens inside an
"edit": a staging area that is created, filled in, and then committed as one
atomic change. Nothing reaches Google Play until the edit is committed.

The dedicated commands (listings, tracks, images, bundles, ...) open and commit
an edit for you. Use these subcommands when you want to batch several commands
into one edit:

    EDIT=$(gplay edits create)
    gplay listings set --edit $EDIT --locale ja --title "..."
    gplay tracks set   --edit $EDIT --track internal --version-code 42
    gplay edits commit $EDIT

Edits expire (typically after 7 days, sooner if the app changes elsewhere) and
a commit fails once expired; just create a new one.`,
}

var editsCreateCmd = &cobra.Command{
	Use:     "create",
	Short:   "Create an edit and print its id",
	Example: `  EDIT=$(gplay edits create --package com.example.app)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			doc, err := c.Post(ctx, appPath(pkg, "/edits"), nil)
			if err != nil {
				return err
			}
			if jsonFlag {
				return printJSON(doc)
			}
			fmt.Println(doc.Str("id"))
			if expiry := doc.Int("expiryTimeSeconds"); expiry > 0 {
				fmt.Fprintf(cmd.ErrOrStderr(), "Expires %s\n", time.Unix(expiry, 0).Format(time.RFC3339))
			}
			return nil
		})
	},
}

var editsGetCmd = &cobra.Command{
	Use:   "get [edit-id]",
	Short: "Show an edit and its expiry",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			id, err := editArg(args)
			if err != nil {
				return err
			}
			doc, err := c.GetMap(ctx, appPath(pkg, "/edits/%s", esc(id)))
			if err != nil {
				return err
			}
			if jsonFlag {
				return printJSON(doc)
			}
			fmt.Printf("id:      %s\n", doc.Str("id"))
			if expiry := doc.Int("expiryTimeSeconds"); expiry > 0 {
				fmt.Printf("expires: %s\n", time.Unix(expiry, 0).Format(time.RFC3339))
			}
			return nil
		})
	},
}

var editsDeleteCmd = &cobra.Command{
	Use:   "delete [edit-id]",
	Short: "Discard an edit without publishing anything",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			id, err := editArg(args)
			if err != nil {
				return err
			}
			if err := c.Delete(ctx, appPath(pkg, "/edits/%s", esc(id))); err != nil {
				return err
			}
			fmt.Printf("Edit %s deleted.\n", id)
			return nil
		})
	},
}

var editsValidateCmd = &cobra.Command{
	Use:   "validate [edit-id]",
	Short: "Check an edit for errors without committing it",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			id, err := editArg(args)
			if err != nil {
				return err
			}
			// Validation changes nothing, so it runs even under --dry-run.
			if _, err := c.Do(ctx, http.MethodPost, appPath(pkg, "/edits/%s:validate", esc(id)), nil); err != nil {
				return err
			}
			fmt.Printf("Edit %s is valid.\n", id)
			return nil
		})
	},
}

var editsCommitCmd = &cobra.Command{
	Use:   "commit [edit-id]",
	Short: "Commit an edit and send the changes to Google Play",
	Long: `Commits the edit. Changes that require review are submitted for review unless
--changes-not-sent-for-review is given.

Use --changes-not-sent-for-review only for changes that do not require review
(for example a staged rollout percentage). Google rejects the commit when the
edit contains changes that do need review, and changes held back this way stay
pending until a later submission carries them along.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			id, err := editArg(args)
			if err != nil {
				return err
			}
			e := &edit{c: c, pkg: pkg, id: id}
			return e.commit(ctx)
		})
	},
}

// editArg takes the edit id from the positional argument or from --edit.
func editArg(args []string) (string, error) {
	if len(args) == 1 && args[0] != "" {
		return args[0], nil
	}
	if editIDFlag != "" {
		return editIDFlag, nil
	}
	return "", fmt.Errorf("no edit id: pass it as an argument or with --edit")
}

// versionCodeList renders a release's version codes for tables.
func versionCodeList(release api.Doc) string {
	return commaJoin(release.Strings("versionCodes"))
}

// releaseSummary renders one release as a single table cell.
func releaseSummary(release api.Doc) string {
	parts := []string{release.Str("status")}
	if codes := versionCodeList(release); codes != "-" {
		parts = append(parts, "codes="+codes)
	}
	if fraction := release.Float("userFraction"); fraction > 0 {
		parts = append(parts, "userFraction="+strconv.FormatFloat(fraction, 'g', -1, 64))
	}
	if name := release.Str("name"); name != "" {
		parts = append(parts, "name="+name)
	}
	return strings.Join(parts, " ")
}

func init() {
	addEditReadFlags(editsGetCmd, editsDeleteCmd, editsValidateCmd, editsCommitCmd)
	editsCommitCmd.Flags().BoolVar(&changesNotSentForReview, "changes-not-sent-for-review", false,
		"commit without sending changes for review")
	editsCommitCmd.Flags().StringVar(&changesInReviewBehavior, "changes-in-review-behavior", "",
		"CANCEL_IN_REVIEW_AND_SUBMIT or ERROR_IF_IN_REVIEW when a review is already pending")
	editsCmd.AddCommand(editsCreateCmd, editsGetCmd, editsDeleteCmd, editsValidateCmd, editsCommitCmd)
	rootCmd.AddCommand(editsCmd)
}
