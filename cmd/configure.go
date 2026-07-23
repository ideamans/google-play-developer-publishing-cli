package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/ideamans/google-play-developer-publishing-cli/internal/config"
)

var (
	configureKeyPath     string
	configureDeveloperID string
	configureForce       bool
)

var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Add a profile from a service account JSON key",
	Long: `Copies the service account JSON key into
~/.config/google-play-developer-publishing/keys/ (with 0600 permissions) and
registers a profile in config.toml. The first registered profile becomes the
default profile.

The key must be a service account key created in Google Cloud Console
(IAM & Admin > Service Accounts > Keys > Add key > JSON) for a project where the
"Google Play Android Developer API" is enabled. The same service account must
then be invited in Play Console > Users and permissions with access to the apps
you intend to manage.

--package and --developer-id are optional defaults; with them you can omit
--package on every later command. Re-run with --force to update an existing
profile (--key may be omitted when only changing the defaults).`,
	Example: `  gplay configure --key ~/Downloads/play-publisher-abc123.json --package com.example.app
  gplay configure --profile client-a --key ~/Downloads/client-a.json --package com.client.app
  gplay configure --force --package com.example.other`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := profileFlag
		if name == "" {
			name = "default"
		}

		cfg, err := config.Load()
		if err != nil {
			return err
		}
		existing, exists := cfg.Profiles[name]
		if exists && !configureForce {
			return fmt.Errorf("profile %q already exists; use --force to overwrite", name)
		}
		if !exists && configureKeyPath == "" {
			return fmt.Errorf("--key is required when creating profile %q", name)
		}

		profile := existing
		var key *config.ServiceAccountKey
		if configureKeyPath != "" {
			data, err := os.ReadFile(configureKeyPath)
			if err != nil {
				return err
			}
			key, err = config.ParseServiceAccount(data)
			if err != nil {
				return fmt.Errorf("%s: %w", configureKeyPath, err)
			}

			keysDir, err := config.KeysDir()
			if err != nil {
				return err
			}
			if err := os.MkdirAll(keysDir, 0o700); err != nil {
				return err
			}
			destName := name + ".json"
			if err := os.WriteFile(filepath.Join(keysDir, destName), data, 0o600); err != nil {
				return err
			}
			profile.ServiceAccount = filepath.Join("keys", destName)
		}
		if packageFlag != "" {
			profile.Package = packageFlag
		}
		if configureDeveloperID != "" {
			profile.DeveloperID = configureDeveloperID
		}

		cfg.Profiles[name] = profile
		madeDefault := false
		if cfg.DefaultProfile == "" {
			cfg.DefaultProfile = name
			madeDefault = true
		}
		if err := cfg.Save(); err != nil {
			return err
		}

		configPath, _ := config.FilePath()
		fmt.Printf("Profile %q saved to %s\n", name, configPath)
		if key != nil {
			fmt.Printf("  service account: %s\n", key.ClientEmail)
			fmt.Printf("  cloud project:   %s\n", key.ProjectID)
		}
		if profile.Package != "" {
			fmt.Printf("  package:         %s\n", profile.Package)
		}
		if profile.DeveloperID != "" {
			fmt.Printf("  developer id:    %s\n", profile.DeveloperID)
		}
		if madeDefault {
			fmt.Printf("Set %q as the default profile.\n", name)
		}
		fmt.Println("\nVerify with: gplay details get" + packageHint(profile.Package))
		return nil
	},
}

func packageHint(pkg string) string {
	if pkg != "" {
		return ""
	}
	return " --package com.example.app"
}

func init() {
	configureCmd.Flags().StringVar(&configureKeyPath, "key", "", "path to the service account JSON key")
	configureCmd.Flags().StringVar(&configureDeveloperID, "developer-id", "", "Play Console developer account id (for the users/grants commands)")
	configureCmd.Flags().BoolVar(&configureForce, "force", false, "overwrite an existing profile")
	rootCmd.AddCommand(configureCmd)
}
