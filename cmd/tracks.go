package cmd

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ideamans/google-play-developer-publishing-cli/internal/api"
)

var (
	trackName         string
	trackFrom         string
	trackTo           string
	trackVersionCodes []int64
	trackStatus       string
	trackUserFraction float64
	trackReleaseName  string
	trackNotes        []string
	trackCountries    []string
	trackRestOfWorld  bool
	trackUpdatePrio   int64
	trackKeepExisting bool
	trackFromJSON     string
	trackType         string
	trackFormFactor   string
	testerGroups      []string
	testerReplace     bool
	trackPut          bool
)

var tracksCmd = &cobra.Command{
	Use:   "tracks",
	Short: "Release tracks and staged rollouts",
	Long: `Manages the releases in a track (internal, alpha, beta, production, or a
closed testing track's own name).

A release couples version codes with a status:

  draft       staged in Play Console, not released
  inProgress  rolling out to a fraction of users (requires --user-fraction)
  halted      rollout paused
  completed   rolled out to everyone

Updating a track replaces its release list. "gplay tracks set" therefore sends
exactly the release you describe, which is what a normal deploy wants; pass
--keep-existing to merge into the releases already in the track instead.`,
}

var tracksListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List every track and its releases",
	Example: `  gplay tracks list --package com.example.app`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runEditRead(cmd, func(ctx context.Context, e *edit) error {
			doc, err := e.c.GetMap(ctx, e.path("/tracks"))
			if err != nil {
				return err
			}
			if jsonFlag {
				return printJSON(doc)
			}
			tracks := doc.Docs("tracks")
			if len(tracks) == 0 {
				fmt.Println("No tracks.")
				return nil
			}
			w := newTable("TRACK", "RELEASES")
			for _, t := range tracks {
				releases := t.Docs("releases")
				if len(releases) == 0 {
					fmt.Fprintf(w, "%s\t-\n", t.Str("track"))
					continue
				}
				for i, r := range releases {
					name := t.Str("track")
					if i > 0 {
						name = ""
					}
					fmt.Fprintf(w, "%s\t%s\n", name, releaseSummary(r))
				}
			}
			return w.Flush()
		})
	},
}

var tracksGetCmd = &cobra.Command{
	Use:     "get",
	Short:   "Show one track",
	Example: `  gplay tracks get --track production`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runEditRead(cmd, func(ctx context.Context, e *edit) error {
			doc, err := e.c.GetMap(ctx, e.path("/tracks/%s", esc(trackName)))
			if err != nil {
				return err
			}
			return printJSON(doc)
		})
	},
}

var tracksSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Put a release into a track",
	Long: `Writes one release into the track. The status defaults to "inProgress" when
--user-fraction is given and "completed" otherwise.

Release notes are per language and accept files:
  --notes ja=@notes-ja.txt --notes en-US="Bug fixes"`,
	Example: `  gplay tracks set --track internal --version-code 42
  gplay tracks set --track production --version-code 42 --user-fraction 0.1 --notes ja=@notes-ja.txt
  gplay tracks set --track production --version-code 42 --status draft
  gplay tracks set --track production --version-code 42 --country JP --country US`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runEdit(cmd, func(ctx context.Context, e *edit) error {
			release, err := buildRelease(cmd)
			if err != nil {
				return err
			}
			return writeTrack(ctx, e, trackName, release, trackKeepExisting)
		})
	},
}

var tracksPromoteCmd = &cobra.Command{
	Use:   "promote",
	Short: "Move a release from one track to another",
	Long: `Reads the source track's release, writes it into the target track, and
removes it from the source track (which is what promoting means in Play).

Without --version-code the version codes of the source track's newest release
are used. Release notes carry over unless --notes is given.`,
	Example: `  gplay tracks promote --from internal --to production --user-fraction 0.1
  gplay tracks promote --from beta --to production --version-code 42`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runEdit(cmd, func(ctx context.Context, e *edit) error {
			source, err := e.c.GetMap(ctx, e.path("/tracks/%s", esc(trackFrom)))
			if err != nil {
				return err
			}
			releases := source.Docs("releases")
			if len(releases) == 0 {
				return fmt.Errorf("track %q has no release to promote", trackFrom)
			}
			from := releases[len(releases)-1]

			release, err := buildRelease(cmd)
			if err != nil {
				return err
			}
			if len(trackVersionCodes) == 0 {
				codes := from.Strings("versionCodes")
				if len(codes) == 0 {
					return fmt.Errorf("the release in %q has no version codes", trackFrom)
				}
				release["versionCodes"] = codes
			}
			if !cmd.Flags().Changed("notes") {
				if notes := from.Docs("releaseNotes"); len(notes) > 0 {
					release["releaseNotes"] = notes
				}
			}
			if !cmd.Flags().Changed("release-name") {
				if name := from.Str("name"); name != "" {
					release["name"] = name
				}
			}

			if err := writeTrack(ctx, e, trackTo, release, trackKeepExisting); err != nil {
				return err
			}
			// Promoting means the build is no longer in the source track.
			if _, err := e.c.Patch(ctx, e.path("/tracks/%s", esc(trackFrom)),
				map[string]any{"track": trackFrom, "releases": []any{}}); err != nil {
				return fmt.Errorf("clear track %s after promotion: %w", trackFrom, err)
			}
			fmt.Printf("Cleared track %s.\n", trackFrom)
			return nil
		})
	},
}

var tracksRolloutCmd = &cobra.Command{
	Use:   "rollout",
	Short: "Change the staged rollout percentage of a track's release",
	Long: `Updates the user fraction of the track's in-progress (or halted) release and
resumes it. A rollout change does not require review, so it can be committed
with --changes-not-sent-for-review.`,
	Example: `  gplay tracks rollout --track production --user-fraction 0.5
  gplay tracks rollout --track production --user-fraction 0.5 --changes-not-sent-for-review`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runEdit(cmd, func(ctx context.Context, e *edit) error {
			return updateRelease(ctx, e, trackName, func(release map[string]any) error {
				release["status"] = "inProgress"
				release["userFraction"] = trackUserFraction
				return nil
			}, "rollout set to "+strconv.FormatFloat(trackUserFraction, 'g', -1, 64))
		})
	},
}

var tracksHaltCmd = &cobra.Command{
	Use:     "halt",
	Short:   "Pause a track's staged rollout",
	Example: `  gplay tracks halt --track production`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runEdit(cmd, func(ctx context.Context, e *edit) error {
			return updateRelease(ctx, e, trackName, func(release map[string]any) error {
				release["status"] = "halted"
				return nil
			}, "rollout halted")
		})
	},
}

var tracksCompleteCmd = &cobra.Command{
	Use:     "complete",
	Short:   "Roll a track's release out to 100% of users",
	Long:    `Sets the release status to "completed" and drops the user fraction.`,
	Example: `  gplay tracks complete --track production`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runEdit(cmd, func(ctx context.Context, e *edit) error {
			return updateRelease(ctx, e, trackName, func(release map[string]any) error {
				release["status"] = "completed"
				delete(release, "userFraction")
				return nil
			}, "rollout completed")
		})
	},
}

var tracksCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new closed testing track",
	Long: `Creates an additional closed testing track. The built-in tracks (internal,
alpha, beta, production) always exist and do not need to be created.`,
	Example: `  gplay tracks create --track qa-team --type CLOSED_TESTING
  gplay tracks create --track wear-qa --type CLOSED_TESTING --form-factor WEAR`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runEdit(cmd, func(ctx context.Context, e *edit) error {
			body := map[string]any{"track": trackName, "type": trackType}
			if trackFormFactor != "" {
				body["formFactor"] = trackFormFactor
			}
			doc, err := e.c.Post(ctx, e.path("/tracks"), body)
			if err != nil {
				return err
			}
			fmt.Printf("Track %s created.\n", trackName)
			if jsonFlag {
				return printJSON(doc)
			}
			return nil
		})
	},
}

var tracksCountriesCmd = &cobra.Command{
	Use:     "country-availability",
	Short:   "Show the countries a track's artifacts are available in",
	Example: `  gplay tracks country-availability --track production`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runEditRead(cmd, func(ctx context.Context, e *edit) error {
			doc, err := e.c.GetMap(ctx, e.path("/countryAvailability/%s", esc(trackName)))
			if err != nil {
				return err
			}
			return printJSON(doc)
		})
	},
}

var releasesListCmd = &cobra.Command{
	Use:   "releases",
	Short: "List a track's releases that are ready for review, in review, or rejected",
	Long: `Shows the review state of the track's releases. Unlike the other track
commands this reads published state directly and needs no edit.`,
	Example: `  gplay tracks releases --track production`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			doc, err := c.GetMap(ctx, appPath(pkg, "/tracks/%s/releases", esc(trackName)))
			if err != nil {
				return err
			}
			return printJSON(doc)
		})
	},
}

// --- Testers -------------------------------------------------------------------

var testersCmd = &cobra.Command{
	Use:   "testers",
	Short: "Google Groups allowed to test a closed track",
	Long: `Manages the Google Groups that can join a closed testing track. Individual
tester email addresses can only be managed in the Play Console UI; the API
works with groups.`,
}

var testersGetCmd = &cobra.Command{
	Use:     "get",
	Short:   "Show the tester groups of a track",
	Example: `  gplay testers get --track alpha`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runEditRead(cmd, func(ctx context.Context, e *edit) error {
			doc, err := e.c.GetMap(ctx, e.path("/testers/%s", esc(trackName)))
			if err != nil {
				return err
			}
			if jsonFlag {
				return printJSON(doc)
			}
			groups := doc.Strings("googleGroups")
			if len(groups) == 0 {
				fmt.Printf("Track %s has no tester groups.\n", trackName)
				return nil
			}
			for _, g := range groups {
				fmt.Println(g)
			}
			return nil
		})
	},
}

var testersSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Add tester groups to a track (or replace them)",
	Example: `  gplay testers set --track alpha --group qa@example.com
  gplay testers set --track alpha --group qa@example.com --replace`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runEdit(cmd, func(ctx context.Context, e *edit) error {
			groups := testerGroups
			if !testerReplace {
				current, err := e.c.GetMap(ctx, e.path("/testers/%s", esc(trackName)))
				if err != nil {
					return err
				}
				groups = mergeUnique(current.Strings("googleGroups"), testerGroups)
			}
			body := map[string]any{"googleGroups": groups}
			path := e.path("/testers/%s", esc(trackName))
			var err error
			if testerReplace {
				// --replace means the group list is authoritative, so send a full update.
				_, err = e.c.Put(ctx, path, body)
			} else {
				_, err = e.c.Patch(ctx, path, body)
			}
			if err != nil {
				return err
			}
			fmt.Printf("Track %s testers: %s\n", trackName, commaJoin(groups))
			return nil
		})
	},
}

// --- Helpers ---------------------------------------------------------------------

// buildRelease turns the release flags into a TrackRelease body.
func buildRelease(cmd *cobra.Command) (map[string]any, error) {
	release := map[string]any{}
	if trackFromJSON != "" {
		if err := jsonFromFlag(trackFromJSON, &release); err != nil {
			return nil, err
		}
	}
	if len(trackVersionCodes) > 0 {
		codes := make([]string, 0, len(trackVersionCodes))
		for _, code := range trackVersionCodes {
			codes = append(codes, strconv.FormatInt(code, 10))
		}
		release["versionCodes"] = codes
	}

	status := trackStatus
	if status == "" {
		if cmd.Flags().Changed("user-fraction") {
			status = "inProgress"
		} else if _, ok := release["status"]; !ok {
			status = "completed"
		}
	}
	if status != "" {
		release["status"] = status
	}
	if cmd.Flags().Changed("user-fraction") {
		if trackUserFraction <= 0 || trackUserFraction >= 1 {
			return nil, fmt.Errorf("--user-fraction must be greater than 0 and less than 1 (got %v); "+
				`use "gplay tracks complete" for a full rollout`, trackUserFraction)
		}
		release["userFraction"] = trackUserFraction
	}
	if s, _ := release["status"].(string); s != "inProgress" && s != "halted" {
		// The API rejects a user fraction on draft/completed releases.
		delete(release, "userFraction")
	}
	if cmd.Flags().Changed("release-name") {
		release["name"] = trackReleaseName
	}
	if cmd.Flags().Changed("in-app-update-priority") {
		release["inAppUpdatePriority"] = trackUpdatePrio
	}
	if cmd.Flags().Changed("notes") {
		notes, err := parseLocalized(trackNotes)
		if err != nil {
			return nil, err
		}
		release["releaseNotes"] = notes
	}
	if len(trackCountries) > 0 || trackRestOfWorld {
		targeting := map[string]any{}
		if len(trackCountries) > 0 {
			targeting["countries"] = upperAll(trackCountries)
		}
		if trackRestOfWorld {
			targeting["includeRestOfWorld"] = true
		}
		release["countryTargeting"] = targeting
	}
	if _, ok := release["versionCodes"]; !ok {
		return nil, fmt.Errorf("no version codes: pass --version-code (repeatable) or --from-json")
	}
	return release, nil
}

// writeTrack sends a release to a track, either replacing its release list or
// merging into it.
func writeTrack(ctx context.Context, e *edit, track string, release map[string]any, keepExisting bool) error {
	releases := []any{release}
	if keepExisting {
		current, err := e.c.GetMap(ctx, e.path("/tracks/%s", esc(track)))
		if err != nil {
			return err
		}
		merged := make([]any, 0, len(current.Docs("releases"))+1)
		replaced := false
		for _, existing := range current.Docs("releases") {
			if existing.Str("status") == release["status"] {
				merged = append(merged, release)
				replaced = true
				continue
			}
			merged = append(merged, map[string]any(existing))
		}
		if !replaced {
			merged = append(merged, release)
		}
		releases = merged
	}
	body := map[string]any{"track": track, "releases": releases}
	path := e.path("/tracks/%s", esc(track))
	var err error
	if trackPut {
		_, err = e.c.Put(ctx, path, body)
	} else {
		_, err = e.c.Patch(ctx, path, body)
	}
	if err != nil {
		return err
	}
	fmt.Printf("Track %s set to %s.\n", track, describeRelease(release))
	return nil
}

// updateRelease reads a track, applies mutate to the release that is currently
// rolling out, and writes the track back.
func updateRelease(ctx context.Context, e *edit, track string, mutate func(map[string]any) error, what string) error {
	current, err := e.c.GetMap(ctx, e.path("/tracks/%s", esc(track)))
	if err != nil {
		return err
	}
	docs := current.Docs("releases")
	if len(docs) == 0 {
		return fmt.Errorf("track %q has no releases", track)
	}
	target := -1
	for i, r := range docs {
		if status := r.Str("status"); status == "inProgress" || status == "halted" {
			target = i
			break
		}
	}
	if target < 0 {
		// No staged rollout in flight: fall back to the newest release.
		target = len(docs) - 1
	}
	releases := make([]any, 0, len(docs))
	for i, r := range docs {
		release := map[string]any(r)
		if i == target {
			if err := mutate(release); err != nil {
				return err
			}
		}
		releases = append(releases, release)
	}
	body := map[string]any{"track": track, "releases": releases}
	if _, err := e.c.Patch(ctx, e.path("/tracks/%s", esc(track)), body); err != nil {
		return err
	}
	fmt.Printf("Track %s: %s.\n", track, what)
	return nil
}

func describeRelease(release map[string]any) string {
	parts := []string{}
	if status, ok := release["status"].(string); ok {
		parts = append(parts, status)
	}
	if codes, ok := release["versionCodes"].([]string); ok {
		parts = append(parts, "version codes "+strings.Join(codes, ","))
	}
	if fraction, ok := release["userFraction"].(float64); ok {
		parts = append(parts, fmt.Sprintf("%.0f%% of users", fraction*100))
	}
	if len(parts) == 0 {
		return "updated"
	}
	return strings.Join(parts, ", ")
}

func mergeUnique(existing, added []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(existing)+len(added))
	for _, value := range append(append([]string{}, existing...), added...) {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func upperAll(values []string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		out = append(out, strings.ToUpper(v))
	}
	return out
}

// addReleaseFlags wires the TrackRelease flags onto commands that build one.
func addReleaseFlags(cmds ...*cobra.Command) {
	for _, c := range cmds {
		c.Flags().Int64SliceVar(&trackVersionCodes, "version-code", nil, "version code to release; repeat for several (e.g. an APK per ABI)")
		c.Flags().StringVar(&trackStatus, "status", "", "draft, inProgress, halted or completed")
		c.Flags().Float64Var(&trackUserFraction, "user-fraction", 0, "staged rollout fraction, 0 < f < 1 (implies --status inProgress)")
		c.Flags().StringVar(&trackReleaseName, "release-name", "", "release name shown in Play Console")
		c.Flags().StringArrayVar(&trackNotes, "notes", nil, "release notes as <language>=<text> or <language>=@file; repeatable")
		c.Flags().StringSliceVar(&trackCountries, "country", nil, "restrict the release to these CLDR country codes, e.g. JP,US")
		c.Flags().BoolVar(&trackRestOfWorld, "include-rest-of-world", false, `also target "rest of world" alongside --country`)
		c.Flags().Int64Var(&trackUpdatePrio, "in-app-update-priority", 0, "in-app update priority, 0-5")
		c.Flags().BoolVar(&trackKeepExisting, "keep-existing", false, "merge into the track's existing releases instead of replacing them")
		c.Flags().StringVar(&trackFromJSON, "from-json", "", "TrackRelease JSON, or @file; other flags override it")
		c.Flags().BoolVar(&trackPut, "put", false, "send a full track update (PUT) instead of a patch")
	}
}

func init() {
	for _, sub := range []*cobra.Command{
		tracksGetCmd, tracksSetCmd, tracksRolloutCmd, tracksHaltCmd, tracksCompleteCmd,
		tracksCreateCmd, tracksCountriesCmd, releasesListCmd, testersGetCmd, testersSetCmd,
	} {
		sub.Flags().StringVar(&trackName, "track", "", "track name: internal, alpha, beta, production or a closed track (required)")
		_ = sub.MarkFlagRequired("track")
	}
	tracksPromoteCmd.Flags().StringVar(&trackFrom, "from", "", "source track (required)")
	tracksPromoteCmd.Flags().StringVar(&trackTo, "to", "", "target track (required)")
	_ = tracksPromoteCmd.MarkFlagRequired("from")
	_ = tracksPromoteCmd.MarkFlagRequired("to")

	addReleaseFlags(tracksSetCmd, tracksPromoteCmd)
	tracksRolloutCmd.Flags().Float64Var(&trackUserFraction, "user-fraction", 0, "staged rollout fraction, 0 < f < 1 (required)")
	_ = tracksRolloutCmd.MarkFlagRequired("user-fraction")

	tracksCreateCmd.Flags().StringVar(&trackType, "type", "CLOSED_TESTING", "track type")
	tracksCreateCmd.Flags().StringVar(&trackFormFactor, "form-factor", "", "DEFAULT, WEAR or AUTOMOTIVE")

	testersSetCmd.Flags().StringArrayVar(&testerGroups, "group", nil, "Google Group email address; repeatable (required)")
	testersSetCmd.Flags().BoolVar(&testerReplace, "replace", false, "replace the group list instead of adding to it")
	_ = testersSetCmd.MarkFlagRequired("group")

	addEditReadFlags(tracksListCmd, tracksGetCmd, tracksCountriesCmd, testersGetCmd)
	addEditFlags(tracksSetCmd, tracksPromoteCmd, tracksRolloutCmd, tracksHaltCmd, tracksCompleteCmd, tracksCreateCmd, testersSetCmd)

	tracksCmd.AddCommand(tracksListCmd, tracksGetCmd, tracksSetCmd, tracksPromoteCmd, tracksRolloutCmd,
		tracksHaltCmd, tracksCompleteCmd, tracksCreateCmd, tracksCountriesCmd, releasesListCmd)
	testersCmd.AddCommand(testersGetCmd, testersSetCmd)
	rootCmd.AddCommand(tracksCmd, testersCmd)
}
