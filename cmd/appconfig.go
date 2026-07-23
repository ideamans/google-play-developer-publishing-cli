package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/ideamans/google-play-developer-publishing-cli/internal/api"
)

var (
	dtcID            string
	dtcFromJSON      string
	dtcAllowUnknown  bool
	safetyLabelsFile string
	recoveryID       string
	recoveryVersion  string
	recoveryFromJSON string
)

// --- Device tier configs ------------------------------------------------------

var dtcCmd = &cobra.Command{
	Use:     "device-tier-configs",
	Aliases: []string{"dtc"},
	Short:   "Device tier configs for Play Asset Delivery / device targeting",
	Long: `Device tier configs describe device groups (by RAM, SoC, system feature) and
tiers, so an app bundle can ship different assets to different devices. A
config is immutable: creating one returns a new id, which you then pass to
"gplay bundles upload --device-tier-config-id".`,
}

var dtcListCmd = &cobra.Command{
	Use:   "list",
	Short: "List device tier configs",
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			items, err := c.ListAll(ctx, appPath(pkg, "/deviceTierConfigs?pageSize=100"), "deviceTierConfigs")
			if err != nil {
				return err
			}
			if jsonFlag {
				return printJSON(items)
			}
			if len(items) == 0 {
				fmt.Println("No device tier configs.")
				return nil
			}
			w := newTable("ID", "DEVICE GROUPS", "TIERS", "COUNTRY SETS")
			for _, d := range items {
				fmt.Fprintf(w, "%s\t%d\t%d\t%d\n", d.Str("deviceTierConfigId"),
					len(d.Docs("deviceGroups")), len(d.Doc("deviceTierSet").Docs("deviceTiers")),
					len(d.Docs("userCountrySets")))
			}
			return w.Flush()
		})
	},
}

var dtcGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Show one device tier config",
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			doc, err := c.GetMap(ctx, appPath(pkg, "/deviceTierConfigs/%s", esc(dtcID)))
			if err != nil {
				return err
			}
			return printJSON(doc)
		})
	},
}

var dtcCreateCmd = &cobra.Command{
	Use:     "create",
	Short:   "Create a device tier config",
	Example: `  gplay device-tier-configs create --from-json @device-tiers.json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			var body map[string]any
			if err := jsonFromFlag(dtcFromJSON, &body); err != nil {
				return err
			}
			path := appPath(pkg, "/deviceTierConfigs")
			if dtcAllowUnknown {
				path += "?allowUnknownDevices=true"
			}
			doc, err := c.Post(ctx, path, body)
			if err != nil {
				return err
			}
			if jsonFlag {
				return printJSON(doc)
			}
			fmt.Println(doc.Str("deviceTierConfigId"))
			return nil
		})
	},
}

// --- Data safety ----------------------------------------------------------------

var dataSafetyCmd = &cobra.Command{
	Use:   "data-safety",
	Short: "Upload the Data safety (privacy) declaration",
	Long: `Replaces the app's Data safety section from a CSV export.

The API takes the exact CSV that Play Console's Data safety page exports and
imports; export it once from the console, edit it, and submit it here. The CSV
content is sent as the safetyLabels string, so this command reads a file rather
than taking individual answers.

This is a whole-section replacement: anything missing from the CSV is cleared.`,
	Example: `  gplay data-safety --file data-safety.csv`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			data, err := os.ReadFile(safetyLabelsFile)
			if err != nil {
				return err
			}
			doc, err := c.Post(ctx, appPath(pkg, "/dataSafety"), map[string]any{"safetyLabels": string(data)})
			if err != nil {
				return err
			}
			fmt.Println("Data safety labels updated.")
			if jsonFlag {
				return printJSON(doc)
			}
			return nil
		})
	},
}

// --- App recovery ------------------------------------------------------------------

var recoveryCmd = &cobra.Command{
	Use:   "app-recovery",
	Short: "App recovery actions for broken releases",
	Long: `App recovery lets you push a remote in-app update to users stuck on a broken
version, without a full release. Create a draft action, then deploy it; cancel
stops an action that is already running.`,
}

var recoveryListCmd = &cobra.Command{
	Use:   "list",
	Short: "List recovery actions",
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			path := appPath(pkg, "/appRecoveries")
			if recoveryVersion != "" {
				path += "?versionCode=" + esc(recoveryVersion)
			}
			doc, err := c.GetMap(ctx, path)
			if err != nil {
				return err
			}
			if jsonFlag {
				return printJSON(doc)
			}
			actions := doc.Docs("recoveryActions")
			if len(actions) == 0 {
				fmt.Println("No recovery actions.")
				return nil
			}
			w := newTable("ID", "STATUS", "CREATE TIME", "CANCEL TIME")
			for _, a := range actions {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", a.Str("appRecoveryId"), a.Str("status"),
					dash(a.Str("createTime")), dash(a.Str("cancelTime")))
			}
			return w.Flush()
		})
	},
}

var recoveryCreateCmd = &cobra.Command{
	Use:     "create",
	Short:   "Create a draft recovery action",
	Long:    `Takes a CreateDraftAppRecoveryRequest body: the targeting and the remote in-app update to apply.`,
	Example: `  gplay app-recovery create --from-json @recovery.json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			var body map[string]any
			if err := jsonFromFlag(recoveryFromJSON, &body); err != nil {
				return err
			}
			doc, err := c.Post(ctx, appPath(pkg, "/appRecoveries"), body)
			if err != nil {
				return err
			}
			return printJSON(doc)
		})
	},
}

var recoveryDeployCmd = &cobra.Command{
	Use:     "deploy",
	Short:   "Deploy a draft recovery action to users",
	Example: `  gplay app-recovery deploy --id 123456`,
	RunE:    recoveryAction("deploy", "deployed"),
}

var recoveryCancelCmd = &cobra.Command{
	Use:   "cancel",
	Short: "Cancel a running recovery action",
	RunE:  recoveryAction("cancel", "cancelled"),
}

var recoveryTargetingCmd = &cobra.Command{
	Use:   "add-targeting",
	Short: "Widen the targeting of a running recovery action",
	Long:  `Takes an AddTargetingRequest body; targeting can only be added, never removed.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			var body map[string]any
			if err := jsonFromFlag(recoveryFromJSON, &body); err != nil {
				return err
			}
			doc, err := c.Post(ctx, appPath(pkg, "/appRecoveries/%s:addTargeting", esc(recoveryID)), body)
			if err != nil {
				return err
			}
			return printJSON(doc)
		})
	},
}

func recoveryAction(verb, done string) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			if _, err := c.Post(ctx, appPath(pkg, "/appRecoveries/%s:%s", esc(recoveryID), verb),
				map[string]any{"packageName": pkg, "appRecoveryId": recoveryID}); err != nil {
				return err
			}
			fmt.Printf("Recovery action %s %s.\n", recoveryID, done)
			return nil
		})
	}
}

func init() {
	dtcGetCmd.Flags().StringVar(&dtcID, "id", "", "device tier config id (required)")
	_ = dtcGetCmd.MarkFlagRequired("id")
	dtcCreateCmd.Flags().StringVar(&dtcFromJSON, "from-json", "", "DeviceTierConfig JSON, or @file (required)")
	dtcCreateCmd.Flags().BoolVar(&dtcAllowUnknown, "allow-unknown-devices", false, "accept device models Google does not recognise")
	_ = dtcCreateCmd.MarkFlagRequired("from-json")

	dataSafetyCmd.Flags().StringVar(&safetyLabelsFile, "file", "", "Data safety CSV exported from Play Console (required)")
	_ = dataSafetyCmd.MarkFlagRequired("file")

	recoveryListCmd.Flags().StringVar(&recoveryVersion, "version-code", "", "only actions targeting this version code")
	for _, sub := range []*cobra.Command{recoveryDeployCmd, recoveryCancelCmd, recoveryTargetingCmd} {
		sub.Flags().StringVar(&recoveryID, "id", "", "app recovery action id (required)")
		_ = sub.MarkFlagRequired("id")
	}
	for _, sub := range []*cobra.Command{recoveryCreateCmd, recoveryTargetingCmd} {
		sub.Flags().StringVar(&recoveryFromJSON, "from-json", "", "request JSON, or @file (required)")
		_ = sub.MarkFlagRequired("from-json")
	}

	dtcCmd.AddCommand(dtcListCmd, dtcGetCmd, dtcCreateCmd)
	recoveryCmd.AddCommand(recoveryListCmd, recoveryCreateCmd, recoveryDeployCmd, recoveryCancelCmd, recoveryTargetingCmd)
	rootCmd.AddCommand(dtcCmd, dataSafetyCmd, recoveryCmd)
}
