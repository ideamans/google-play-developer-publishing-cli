package cmd

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ideamans/google-play-developer-publishing-cli/internal/api"
)

var (
	developerIDFlag string
	userEmail       string
	userPermissions []string
	userExpiry      string
	grantPackage    string
	grantPermission []string
	userFromJSON    string
)

var usersCmd = &cobra.Command{
	Use:   "users",
	Short: "Developer account users and their permissions",
	Long: `Manages who has access to the developer account. These commands work on the
developer account rather than a single app, so they need the developer account
id: the number in the Play Console URL
(play.google.com/console/u/0/developers/<DEVELOPER_ID>/...). Store it once with
"gplay configure --developer-id ...".

Account-wide permissions go on the user; per-app permissions are grants, one
per package (see "gplay grants").`,
}

var usersListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List the users of the developer account",
	Example: `  gplay users list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDeveloper(cmd, func(ctx context.Context, c *api.Client, dev string) error {
			items, err := c.ListAll(ctx, "/developers/"+esc(dev)+"/users?pageSize=100", "users")
			if err != nil {
				return err
			}
			if jsonFlag {
				return printJSON(items)
			}
			if len(items) == 0 {
				fmt.Println("No users.")
				return nil
			}
			w := newTable("EMAIL", "ACCESS", "ACCOUNT PERMISSIONS", "APP GRANTS")
			for _, u := range items {
				grants := []string{}
				for _, g := range u.Docs("grants") {
					grants = append(grants, g.Str("packageName"))
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", u.Str("email"), u.Str("accessState"),
					commaJoin(u.Strings("developerAccountPermissions")), commaJoin(grants))
			}
			return w.Flush()
		})
	},
}

var usersCreateCmd = &cobra.Command{
	Use:   "invite",
	Short: "Invite a user to the developer account",
	Long: `Sends an invitation. Account-wide permissions are enum values such as
CAN_VIEW_FINANCIAL_DATA_GLOBAL, CAN_MANAGE_PERMISSIONS_GLOBAL or
CAN_REPLY_TO_REVIEWS_GLOBAL; per-app access is granted separately with
"gplay grants create".`,
	Example: `  gplay users invite --email dev@example.com --permission CAN_REPLY_TO_REVIEWS_GLOBAL
  gplay users invite --email dev@example.com --expires 2026-12-31T00:00:00Z`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDeveloper(cmd, func(ctx context.Context, c *api.Client, dev string) error {
			body := map[string]any{"email": userEmail}
			if userFromJSON != "" {
				if err := jsonFromFlag(userFromJSON, &body); err != nil {
					return err
				}
				body["email"] = userEmail
			}
			if len(userPermissions) > 0 {
				body["developerAccountPermissions"] = userPermissions
			}
			if userExpiry != "" {
				body["expirationTime"] = userExpiry
			}
			doc, err := c.Post(ctx, "/developers/"+esc(dev)+"/users", body)
			if err != nil {
				return err
			}
			fmt.Printf("Invited %s.\n", userEmail)
			if jsonFlag {
				return printJSON(doc)
			}
			return nil
		})
	},
}

var usersUpdateCmd = &cobra.Command{
	Use:     "update",
	Short:   "Change a user's account-wide permissions or expiry",
	Example: `  gplay users update --email dev@example.com --permission CAN_REPLY_TO_REVIEWS_GLOBAL`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDeveloper(cmd, func(ctx context.Context, c *api.Client, dev string) error {
			body := map[string]any{}
			mask := []string{}
			if len(userPermissions) > 0 {
				body["developerAccountPermissions"] = userPermissions
				mask = append(mask, "developerAccountPermissions")
			}
			if userExpiry != "" {
				body["expirationTime"] = userExpiry
				mask = append(mask, "expirationTime")
			}
			if len(mask) == 0 {
				return fmt.Errorf("nothing to update: pass --permission and/or --expires")
			}
			path := fmt.Sprintf("/developers/%s/users/%s?updateMask=%s",
				esc(dev), esc(userEmail), url.QueryEscape(strings.Join(mask, ",")))
			if _, err := c.Patch(ctx, path, body); err != nil {
				return err
			}
			fmt.Printf("User %s updated.\n", userEmail)
			return nil
		})
	},
}

var usersDeleteCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a user from the developer account",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDeveloper(cmd, func(ctx context.Context, c *api.Client, dev string) error {
			if err := c.Delete(ctx, "/developers/"+esc(dev)+"/users/"+esc(userEmail)); err != nil {
				return err
			}
			fmt.Printf("User %s removed.\n", userEmail)
			return nil
		})
	},
}

// --- Grants ---------------------------------------------------------------------

var grantsCmd = &cobra.Command{
	Use:   "grants",
	Short: "Per-app permissions for a user",
	Long: `A grant is one user's access to one app. App-level permissions are enum
values such as CAN_MANAGE_PUBLIC_APKS, CAN_VIEW_FINANCIAL_DATA,
CAN_REPLY_TO_REVIEWS or CAN_MANAGE_STORE_PRESENCE.`,
}

var grantsCreateCmd = &cobra.Command{
	Use:     "create",
	Short:   "Grant a user access to an app",
	Example: `  gplay grants create --email dev@example.com --grant-package com.example.app --permission CAN_MANAGE_PUBLIC_APKS`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDeveloper(cmd, func(ctx context.Context, c *api.Client, dev string) error {
			pkg, err := grantPackageName(c)
			if err != nil {
				return err
			}
			body := map[string]any{
				"packageName":         pkg,
				"appLevelPermissions": grantPermission,
			}
			doc, err := c.Post(ctx, "/developers/"+esc(dev)+"/users/"+esc(userEmail)+"/grants", body)
			if err != nil {
				return err
			}
			fmt.Printf("Granted %s access to %s.\n", userEmail, pkg)
			if jsonFlag {
				return printJSON(doc)
			}
			return nil
		})
	},
}

var grantsUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Change a user's permissions on an app",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDeveloper(cmd, func(ctx context.Context, c *api.Client, dev string) error {
			pkg, err := grantPackageName(c)
			if err != nil {
				return err
			}
			body := map[string]any{"appLevelPermissions": grantPermission}
			path := fmt.Sprintf("/developers/%s/users/%s/grants/%s?updateMask=appLevelPermissions",
				esc(dev), esc(userEmail), esc(pkg))
			if _, err := c.Patch(ctx, path, body); err != nil {
				return err
			}
			fmt.Printf("Updated %s's permissions on %s.\n", userEmail, pkg)
			return nil
		})
	},
}

var grantsDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Remove a user's access to an app",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDeveloper(cmd, func(ctx context.Context, c *api.Client, dev string) error {
			pkg, err := grantPackageName(c)
			if err != nil {
				return err
			}
			if err := c.Delete(ctx, "/developers/"+esc(dev)+"/users/"+esc(userEmail)+"/grants/"+esc(pkg)); err != nil {
				return err
			}
			fmt.Printf("Removed %s's access to %s.\n", userEmail, pkg)
			return nil
		})
	},
}

// runDeveloper resolves the client and developer account id, then calls fn.
func runDeveloper(cmd *cobra.Command, fn func(ctx context.Context, c *api.Client, dev string) error) error {
	c, err := newClient()
	if err != nil {
		return err
	}
	dev, err := resolveDeveloperID(c)
	if err != nil {
		return err
	}
	return fn(cmd.Context(), c, dev)
}

// grantPackageName resolves the app a grant applies to, preferring the
// dedicated flag over the global --package.
func grantPackageName(c *api.Client) (string, error) {
	if grantPackage != "" {
		return grantPackage, nil
	}
	return resolvePackage(c)
}

func init() {
	for _, sub := range []*cobra.Command{usersListCmd, usersCreateCmd, usersUpdateCmd, usersDeleteCmd,
		grantsCreateCmd, grantsUpdateCmd, grantsDeleteCmd} {
		sub.Flags().StringVar(&developerIDFlag, "developer-id", "", "Play Console developer account id (env: GPLAY_DEVELOPER_ID)")
	}
	for _, sub := range []*cobra.Command{usersCreateCmd, usersUpdateCmd, usersDeleteCmd,
		grantsCreateCmd, grantsUpdateCmd, grantsDeleteCmd} {
		sub.Flags().StringVar(&userEmail, "email", "", "user email address (required)")
		_ = sub.MarkFlagRequired("email")
	}
	for _, sub := range []*cobra.Command{usersCreateCmd, usersUpdateCmd} {
		sub.Flags().StringArrayVar(&userPermissions, "permission", nil, "account-wide permission enum; repeatable")
		sub.Flags().StringVar(&userExpiry, "expires", "", "access expiry as an RFC 3339 timestamp")
	}
	usersCreateCmd.Flags().StringVar(&userFromJSON, "from-json", "", "User JSON, or @file")
	for _, sub := range []*cobra.Command{grantsCreateCmd, grantsUpdateCmd, grantsDeleteCmd} {
		sub.Flags().StringVar(&grantPackage, "grant-package", "", "package the grant applies to (defaults to --package)")
	}
	for _, sub := range []*cobra.Command{grantsCreateCmd, grantsUpdateCmd} {
		sub.Flags().StringArrayVar(&grantPermission, "permission", nil, "app-level permission enum; repeatable (required)")
		_ = sub.MarkFlagRequired("permission")
	}

	usersCmd.AddCommand(usersListCmd, usersCreateCmd, usersUpdateCmd, usersDeleteCmd)
	grantsCmd.AddCommand(grantsCreateCmd, grantsUpdateCmd, grantsDeleteCmd)
	rootCmd.AddCommand(usersCmd, grantsCmd)
}
