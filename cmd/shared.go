package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/ideamans/google-play-developer-publishing-cli/internal/api"
	"github.com/ideamans/google-play-developer-publishing-cli/internal/config"
)

// newClient resolves credentials and returns a client honoring --dry-run.
func newClient() (*api.Client, error) {
	creds, err := config.Resolve(profileFlag)
	if err != nil {
		return nil, err
	}
	c := api.New(creds)
	c.DryRun = dryRun
	return c, nil
}

// resolvePackage returns the package name from --package, then the environment
// or the profile default.
func resolvePackage(c *api.Client) (string, error) {
	if packageFlag != "" {
		return packageFlag, nil
	}
	if pkg := c.Credentials().Package; pkg != "" {
		return pkg, nil
	}
	return "", fmt.Errorf("no package name: pass --package com.example.app, set GPLAY_PACKAGE, " +
		`or store one in the profile with "gplay configure --package ..."`)
}

// resolveDeveloperID returns the Play Console developer account id, needed by
// the users/grants commands.
func resolveDeveloperID(c *api.Client) (string, error) {
	if developerIDFlag != "" {
		return developerIDFlag, nil
	}
	if id := c.Credentials().DeveloperID; id != "" {
		return id, nil
	}
	return "", fmt.Errorf("no developer account id: pass --developer-id, set GPLAY_DEVELOPER_ID, " +
		"or store one in the profile. Find it in the Play Console URL: " +
		"play.google.com/console/u/0/developers/<DEVELOPER_ID>/...")
}

// appPath builds "/applications/<package>[/suffix]".
func appPath(pkg, format string, args ...any) string {
	suffix := ""
	if format != "" {
		suffix = fmt.Sprintf(format, args...)
	}
	return "/applications/" + esc(pkg) + suffix
}

// esc escapes a single path segment. Package names, product ids and purchase
// tokens all end up in the path and may contain characters that need escaping.
func esc(s string) string { return url.PathEscape(s) }

// --- Edit sessions ------------------------------------------------------------

var (
	editIDFlag              string
	noCommitFlag            bool
	changesNotSentForReview bool
	changesInReviewBehavior string
)

// dryRunEditID stands in for a real edit id when --dry-run suppresses the
// edits.insert call, so the printed request paths stay readable.
const dryRunEditID = "DRY-RUN-EDIT"

// edit is an open Android Publisher edit transaction. Nothing an edit contains
// reaches the store until it is committed.
type edit struct {
	c     *api.Client
	pkg   string
	id    string
	owned bool // created by this invocation, so this invocation commits or deletes it
}

// openEdit joins the edit given by --edit, or creates a fresh one. Under
// --dry-run no edit is created and a placeholder id keeps the printed request
// paths readable.
func openEdit(ctx context.Context, c *api.Client, pkg string) (*edit, error) {
	if editIDFlag != "" {
		return &edit{c: c, pkg: pkg, id: editIDFlag}, nil
	}
	if c.DryRun {
		fmt.Fprintf(os.Stderr, "DRY-RUN POST %s\n", api.URL(appPath(pkg, "/edits")))
		return &edit{c: c, pkg: pkg, id: dryRunEditID, owned: true}, nil
	}
	return insertEdit(ctx, c, pkg)
}

// openEditForRead is openEdit for read-only commands. It creates a real edit
// even under --dry-run, because a read needs a usable edit id and an edit that
// is never committed changes nothing.
func openEditForRead(ctx context.Context, c *api.Client, pkg string) (*edit, error) {
	if editIDFlag != "" {
		return &edit{c: c, pkg: pkg, id: editIDFlag}, nil
	}
	return insertEdit(ctx, c, pkg)
}

func insertEdit(ctx context.Context, c *api.Client, pkg string) (*edit, error) {
	data, err := c.Do(ctx, http.MethodPost, appPath(pkg, "/edits"), nil)
	if err != nil {
		return nil, err
	}
	var doc api.Doc
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	id := doc.Str("id")
	if id == "" {
		return nil, fmt.Errorf("edits.insert returned no edit id")
	}
	return &edit{c: c, pkg: pkg, id: id, owned: true}, nil
}

// path builds a path inside the edit, e.g. e.path("/listings/%s", "ja").
func (e *edit) path(format string, args ...any) string {
	suffix := ""
	if format != "" {
		suffix = fmt.Sprintf(format, args...)
	}
	return appPath(e.pkg, "/edits/%s%s", esc(e.id), suffix)
}

// commit finalizes the edit and sends the changes to Google Play.
func (e *edit) commit(ctx context.Context) error {
	path := e.path(":commit")
	var params []string
	if changesNotSentForReview {
		params = append(params, "changesNotSentForReview=true")
	}
	if changesInReviewBehavior != "" {
		params = append(params, "changesInReviewBehavior="+url.QueryEscape(changesInReviewBehavior))
	}
	if len(params) > 0 {
		path += "?" + strings.Join(params, "&")
	}
	if _, err := e.c.Post(ctx, path, nil); err != nil {
		return err
	}
	if e.c.DryRun {
		fmt.Println("Dry run complete. Nothing was sent to Google Play.")
		return nil
	}
	fmt.Printf("Edit %s committed.\n", e.id)
	return nil
}

// abandon deletes an edit created by this invocation. Errors are ignored: the
// caller is already reporting a failure, and edits expire on their own. The
// delete bypasses --dry-run because a read-only command creates a real edit
// even then, and leaving it behind would be the surprising outcome.
func (e *edit) abandon(ctx context.Context) {
	if !e.owned || e.id == dryRunEditID {
		return
	}
	_, _ = e.c.Do(ctx, http.MethodDelete, e.path(""), nil)
}

// finish commits an edit this invocation created, unless --no-commit was given.
// An edit joined via --edit is always left open for the caller to chain.
func (e *edit) finish(ctx context.Context) error {
	if !e.owned {
		fmt.Fprintf(os.Stderr, "Edit %s left open (--edit); commit it with: gplay edits commit --edit %s\n", e.id, e.id)
		return nil
	}
	if noCommitFlag {
		fmt.Printf("%s\n", e.id)
		fmt.Fprintf(os.Stderr, "Edit not committed (--no-commit). Continue with --edit %s, then: gplay edits commit --edit %s\n", e.id, e.id)
		return nil
	}
	return e.commit(ctx)
}

// runEdit resolves the client and package, opens an edit, runs fn, and commits.
// A failure inside fn abandons the edit so nothing half-applied is left behind.
func runEdit(cmd *cobra.Command, fn func(ctx context.Context, e *edit) error) error {
	c, err := newClient()
	if err != nil {
		return err
	}
	pkg, err := resolvePackage(c)
	if err != nil {
		return err
	}
	ctx := cmd.Context()
	e, err := openEdit(ctx, c, pkg)
	if err != nil {
		return err
	}
	if err := fn(ctx, e); err != nil {
		e.abandon(ctx)
		return err
	}
	return e.finish(ctx)
}

// runEditRead is runEdit for read-only commands: reads also need an open edit,
// but it is discarded instead of committed.
func runEditRead(cmd *cobra.Command, fn func(ctx context.Context, e *edit) error) error {
	c, err := newClient()
	if err != nil {
		return err
	}
	pkg, err := resolvePackage(c)
	if err != nil {
		return err
	}
	ctx := cmd.Context()
	e, err := openEditForRead(ctx, c, pkg)
	if err != nil {
		return err
	}
	defer e.abandon(ctx)
	return fn(ctx, e)
}

// run is the non-edit equivalent: resolve client and package, then call fn.
func run(cmd *cobra.Command, fn func(ctx context.Context, c *api.Client, pkg string) error) error {
	c, err := newClient()
	if err != nil {
		return err
	}
	pkg, err := resolvePackage(c)
	if err != nil {
		return err
	}
	return fn(cmd.Context(), c, pkg)
}

// addEditFlags wires the edit-transaction flags onto mutating commands.
func addEditFlags(cmds ...*cobra.Command) {
	for _, c := range cmds {
		c.Flags().StringVar(&editIDFlag, "edit", "", "apply to an existing edit id instead of creating one (leaves it open)")
		c.Flags().BoolVar(&noCommitFlag, "no-commit", false, "create the edit and apply changes but do not commit; prints the edit id")
		c.Flags().BoolVar(&changesNotSentForReview, "changes-not-sent-for-review", false,
			"commit without sending changes for review (only for changes that do not require review)")
		c.Flags().StringVar(&changesInReviewBehavior, "changes-in-review-behavior", "",
			"CANCEL_IN_REVIEW_AND_SUBMIT or ERROR_IF_IN_REVIEW when a review is already pending")
	}
}

// addEditReadFlags wires only --edit onto read-only edit-scoped commands.
func addEditReadFlags(cmds ...*cobra.Command) {
	for _, c := range cmds {
		c.Flags().StringVar(&editIDFlag, "edit", "", "read from an existing edit id instead of creating a throwaway one")
	}
}

// --- Output -------------------------------------------------------------------

// reportUpload announces an upload result. Under --dry-run the version code
// would be a fabricated zero, so the DRY-RUN line printed by the client is left
// to speak for itself.
func reportUpload(c *api.Client, format string, args ...any) {
	if c.DryRun {
		return
	}
	fmt.Printf(format, args...)
}

// printJSON pretty-prints a value to stdout.
func printJSON(v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}

// newTable returns a tab writer with the given header row already written.
func newTable(headers ...string) *tabwriter.Writer {
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, strings.Join(headers, "\t"))
	return w
}

// --- Input helpers ------------------------------------------------------------

// valueOrFile returns s, or the file contents when s starts with "@".
func valueOrFile(s string) (string, error) {
	if strings.HasPrefix(s, "@") {
		b, err := os.ReadFile(s[1:])
		if err != nil {
			return "", err
		}
		return strings.TrimRight(string(b), "\n"), nil
	}
	return s, nil
}

// jsonFromFlag decodes a JSON value given inline or as @file.
func jsonFromFlag(s string, v any) error {
	raw, err := valueOrFile(s)
	if err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(raw), v); err != nil {
		return fmt.Errorf("parse JSON: %w", err)
	}
	return nil
}

// parseLocalized turns repeated "<language>=<text|@file>" flags into the
// LocalizedText array shape the API uses for release notes.
func parseLocalized(values []string) ([]map[string]string, error) {
	out := make([]map[string]string, 0, len(values))
	for _, v := range values {
		lang, text, ok := strings.Cut(v, "=")
		if !ok || lang == "" {
			return nil, fmt.Errorf("invalid value %q: use <language>=<text> or <language>=@file.txt, e.g. ja=@notes-ja.txt", v)
		}
		resolved, err := valueOrFile(text)
		if err != nil {
			return nil, err
		}
		out = append(out, map[string]string{"language": lang, "text": resolved})
	}
	return out, nil
}

// splitKeyValue parses a "key=value" flag.
func splitKeyValue(v string) (string, string, error) {
	key, value, ok := strings.Cut(v, "=")
	if !ok || key == "" {
		return "", "", fmt.Errorf("invalid value %q: expected key=value", v)
	}
	return key, value, nil
}

// commaJoin renders a string slice for table cells.
func commaJoin(values []string) string {
	if len(values) == 0 {
		return "-"
	}
	return strings.Join(values, ",")
}

// dash renders empty strings as "-" in tables.
func dash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
