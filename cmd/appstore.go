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
	storePackage     string
	hostedAppPackage string
	hostedFromJSON   string
	hostedPublishSt  string
	hostedFile       string
	hostedFileType   string
	catalogStartTime string
	catalogEndTime   string
	catalogPageSize  int
)

var appStoreCmd = &cobra.Command{
	Use:   "app-store",
	Short: "Alternative app store operator APIs (hosted apps, catalog export)",
	Long: `These commands are for operators of an app store other than Google Play, not
for app developers. They cover two surfaces:

  apps     register apps your store hosts, submit their details for Google's
           policy review, and upload their APKs, images and declaration files
  catalog  read the Play catalog export: metadata about recently updated apps
           and a feed of update events

Access is granted by Google to approved app store partners; other accounts get
403 here. Everything is keyed by --store-package, the package name of the app
store application itself.`,
}

// --- Hosted apps ------------------------------------------------------------------

var hostedAppsCmd = &cobra.Command{
	Use:   "apps",
	Short: "Apps hosted by your app store",
}

var hostedCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Register a hosted app record",
	Long: `Creates the app record. This must be called before any other command for
that hosted app.`,
	Example: `  gplay app-store apps create --store-package com.example.appstore --app-package com.example.hostedapp`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAppStore(cmd, func(ctx context.Context, c *api.Client, app string) error {
			doc, err := c.Post(ctx, appStorePath("/apps:create"), map[string]any{"packageName": app})
			if err != nil {
				return err
			}
			fmt.Printf("Hosted app %s created.\n", app)
			if jsonFlag {
				return printJSON(doc)
			}
			return nil
		})
	},
}

var hostedUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Submit or update a hosted app's details for review",
	Long: `Sends the app's details, localized store listings, active APK sets and policy
declarations. The update is sent for Google's review immediately.

The body is an UpdateAppStoreHostedAppRequest. Image and APK ids in it come
from "app-store apps upload-image" and "app-store apps upload-apk".`,
	Example: `  gplay app-store apps update --store-package com.example.appstore --app-package com.example.hostedapp --from-json @hosted-app.json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAppStore(cmd, func(ctx context.Context, c *api.Client, app string) error {
			var body map[string]any
			if err := jsonFromFlag(hostedFromJSON, &body); err != nil {
				return err
			}
			body["packageName"] = app
			doc, err := c.Post(ctx, appStorePath("/apps:update"), body)
			if err != nil {
				return err
			}
			fmt.Printf("Hosted app %s submitted for review.\n", app)
			if jsonFlag {
				return printJSON(doc)
			}
			return nil
		})
	},
}

var hostedPublishCmd = &cobra.Command{
	Use:   "publish-status",
	Short: "Publish or unpublish a hosted app",
	Long: `Sets the publish state. An app is PUBLISHED by default after a successful
update, so this is only needed to unpublish or to re-publish.`,
	Example: `  gplay app-store apps publish-status --store-package com.example.appstore --app-package com.example.hostedapp --state UNPUBLISHED`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAppStore(cmd, func(ctx context.Context, c *api.Client, app string) error {
			state := hostedPublishSt
			if !strings.HasPrefix(state, "APP_STORE_APP_PUBLISH_STATE_") {
				state = "APP_STORE_APP_PUBLISH_STATE_" + strings.ToUpper(state)
			}
			path := appStorePath("/apps/%s:updateAppStoreHostedAppPublishStatus", esc(app))
			if _, err := c.Post(ctx, path, map[string]any{"publishState": state}); err != nil {
				return err
			}
			fmt.Printf("Hosted app %s is now %s.\n", app, state)
			return nil
		})
	},
}

var hostedUploadAPKCmd = &cobra.Command{
	Use:     "upload-apk",
	Short:   "Upload an APK for a hosted app and print its id",
	Long:    `Returns an APK id to reference from the activeApks field of "app-store apps update".`,
	Example: `  gplay app-store apps upload-apk --store-package com.example.appstore --app-package com.example.hostedapp --file app.apk`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAppStore(cmd, func(ctx context.Context, c *api.Client, app string) error {
			doc, err := c.UploadFile(ctx, appStorePath("/apps/%s/apks:upload", esc(app)),
				hostedFile, "application/vnd.android.package-archive")
			if err != nil {
				return err
			}
			if jsonFlag {
				return printJSON(doc)
			}
			fmt.Println(doc.Str("apkId"))
			return nil
		})
	},
}

var hostedUploadImageCmd = &cobra.Command{
	Use:     "upload-image",
	Short:   "Upload an icon or screenshot for a hosted app and print its id",
	Long:    `Returns an image id to reference from appIconId or screenshotId in "app-store apps update".`,
	Example: `  gplay app-store apps upload-image --store-package com.example.appstore --app-package com.example.hostedapp --file icon.png`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAppStore(cmd, func(ctx context.Context, c *api.Client, app string) error {
			doc, err := c.UploadFile(ctx, appStorePath("/apps/%s/images:upload", esc(app)),
				hostedFile, contentTypeFor(hostedFile))
			if err != nil {
				return err
			}
			if jsonFlag {
				return printJSON(doc)
			}
			fmt.Println(doc.Str("imageId"))
			return nil
		})
	},
}

var hostedUploadPolicyCmd = &cobra.Command{
	Use:   "upload-policy-file",
	Short: "Upload a policy declaration file for a hosted app",
	Long: `Uploads supporting documentation for a policy declaration and prints its id.
The file type is sent alongside the bytes, so this uses a multipart upload.`,
	Example: `  gplay app-store apps upload-policy-file --store-package com.example.appstore --app-package com.example.hostedapp --file declaration.pdf`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAppStore(cmd, func(ctx context.Context, c *api.Client, app string) error {
			fileType := hostedFileType
			if !strings.HasPrefix(fileType, "DECLARATION_FILE_TYPE_") {
				fileType = "DECLARATION_FILE_TYPE_" + strings.ToUpper(fileType)
			}
			doc, err := c.UploadMultipart(ctx,
				appStorePath("/apps/%s/policyDeclarationFiles:upload", esc(app)),
				hostedFile, policyContentType(hostedFile),
				map[string]any{"fileType": fileType})
			if err != nil {
				return err
			}
			return printJSON(doc)
		})
	},
}

// --- Catalog export ------------------------------------------------------------------

var catalogCmd = &cobra.Command{
	Use:   "catalog",
	Short: "Google Play catalog export for app stores",
}

var catalogAppViewCmd = &cobra.Command{
	Use:     "app",
	Short:   "Show catalog metadata for one Play app",
	Example: `  gplay app-store catalog app --store-package com.example.appstore --app-package com.example.playapp`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAppStore(cmd, func(ctx context.Context, c *api.Client, app string) error {
			doc, err := c.GetMap(ctx, catalogPath("/recentAppViews/%s", esc(app)))
			if err != nil {
				return err
			}
			return printJSON(doc)
		})
	},
}

var catalogEventsCmd = &cobra.Command{
	Use:     "updates",
	Short:   "List update events for eligible apps in a time range",
	Example: `  gplay app-store catalog updates --store-package com.example.appstore --start 2026-07-01T00:00:00Z`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		if storePackage == "" {
			return errStorePackage()
		}
		params := url.Values{}
		if catalogStartTime != "" {
			params.Set("startTime", catalogStartTime)
		}
		if catalogEndTime != "" {
			params.Set("endTime", catalogEndTime)
		}
		if catalogPageSize > 0 {
			params.Set("pageSize", fmt.Sprint(catalogPageSize))
		}
		path := catalogPath("/recentUpdateEvents")
		if len(params) > 0 {
			path += "?" + params.Encode()
		}
		items, err := c.ListAll(cmd.Context(), path, "recentUpdateEvents")
		if err != nil {
			return err
		}
		if jsonFlag {
			return printJSON(items)
		}
		if len(items) == 0 {
			fmt.Println("No update events in the given range.")
			return nil
		}
		w := newTable("EVENT TIME", "TYPE", "PACKAGE NAME")
		for _, e := range items {
			fmt.Fprintf(w, "%s\t%s\t%s\n", e.Str("eventTime"), e.Str("updateType"), e.Str("playAppPackageName"))
		}
		return w.Flush()
	},
}

// --- Helpers ----------------------------------------------------------------------------

func appStorePath(format string, args ...any) string {
	return "/appstore/" + esc(storePackage) + fmt.Sprintf(format, args...)
}

func catalogPath(format string, args ...any) string {
	return "/appstorecatalog/" + esc(storePackage) + fmt.Sprintf(format, args...)
}

// runAppStore resolves the client and the hosted app's package name. Unlike the
// developer commands, the package here is the app the store hosts, while
// --store-package identifies the store itself.
func runAppStore(cmd *cobra.Command, fn func(ctx context.Context, c *api.Client, app string) error) error {
	c, err := newClient()
	if err != nil {
		return err
	}
	if storePackage == "" {
		return errStorePackage()
	}
	app := hostedAppPackage
	if app == "" {
		app, err = resolvePackage(c)
		if err != nil {
			return fmt.Errorf("no hosted app package: pass --app-package")
		}
	}
	return fn(cmd.Context(), c, app)
}

func errStorePackage() error {
	return fmt.Errorf("--store-package is required: the package name of your app store application")
}

func policyContentType(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".pdf":
		return "application/pdf"
	default:
		return contentTypeFor(path)
	}
}

func init() {
	storeScoped := []*cobra.Command{
		hostedCreateCmd, hostedUpdateCmd, hostedPublishCmd,
		hostedUploadAPKCmd, hostedUploadImageCmd, hostedUploadPolicyCmd,
		catalogAppViewCmd, catalogEventsCmd,
	}
	for _, sub := range storeScoped {
		sub.Flags().StringVar(&storePackage, "store-package", "", "package name of your app store application (required)")
		_ = sub.MarkFlagRequired("store-package")
	}
	for _, sub := range storeScoped {
		if sub == catalogEventsCmd {
			continue
		}
		sub.Flags().StringVar(&hostedAppPackage, "app-package", "", "package name of the app (defaults to --package)")
	}
	hostedUpdateCmd.Flags().StringVar(&hostedFromJSON, "from-json", "", "UpdateAppStoreHostedAppRequest JSON, or @file (required)")
	_ = hostedUpdateCmd.MarkFlagRequired("from-json")
	hostedPublishCmd.Flags().StringVar(&hostedPublishSt, "state", "", "PUBLISHED or UNPUBLISHED (required)")
	_ = hostedPublishCmd.MarkFlagRequired("state")
	for _, sub := range []*cobra.Command{hostedUploadAPKCmd, hostedUploadImageCmd, hostedUploadPolicyCmd} {
		sub.Flags().StringVar(&hostedFile, "file", "", "file to upload (required)")
		_ = sub.MarkFlagRequired("file")
	}
	hostedUploadPolicyCmd.Flags().StringVar(&hostedFileType, "type", "DOCUMENT", "policy declaration file type")

	catalogEventsCmd.Flags().StringVar(&catalogStartTime, "start", "", "start of the range as an RFC 3339 timestamp")
	catalogEventsCmd.Flags().StringVar(&catalogEndTime, "end", "", "end of the range as an RFC 3339 timestamp")
	catalogEventsCmd.Flags().IntVar(&catalogPageSize, "page-size", 0, "events per page")

	hostedAppsCmd.AddCommand(hostedCreateCmd, hostedUpdateCmd, hostedPublishCmd,
		hostedUploadAPKCmd, hostedUploadImageCmd, hostedUploadPolicyCmd)
	catalogCmd.AddCommand(catalogAppViewCmd, catalogEventsCmd)
	appStoreCmd.AddCommand(hostedAppsCmd, catalogCmd)
	rootCmd.AddCommand(appStoreCmd)
}
