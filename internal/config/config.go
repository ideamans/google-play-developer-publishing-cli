package config

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

const dirName = "google-play-developer-publishing"

// Profile is a named set of Google Play Developer Publishing API credentials.
type Profile struct {
	// ServiceAccount is a path to the service account JSON key, absolute or
	// relative to the config directory.
	ServiceAccount string `toml:"service_account"`
	// Package is the default Android package name for this profile.
	Package string `toml:"package,omitempty"`
	// DeveloperID is the Play Console developer account id, needed only by the
	// users/grants commands.
	DeveloperID string `toml:"developer_id,omitempty"`
}

type Config struct {
	DefaultProfile string             `toml:"default_profile,omitempty"`
	Profiles       map[string]Profile `toml:"profiles,omitempty"`
}

// Dir returns the configuration directory
// (~/.config/google-play-developer-publishing, honoring XDG_CONFIG_HOME).
func Dir() (string, error) {
	if base := os.Getenv("XDG_CONFIG_HOME"); base != "" {
		return filepath.Join(base, dirName), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", dirName), nil
}

func FilePath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.toml"), nil
}

func KeysDir() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "keys"), nil
}

// Load reads config.toml. A missing file yields an empty config.
func Load() (*Config, error) {
	path, err := FilePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return &Config{Profiles: map[string]Profile{}}, nil
	}
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]Profile{}
	}
	return &cfg, nil
}

func (c *Config) Save() error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := toml.Marshal(c)
	if err != nil {
		return err
	}
	path, err := FilePath()
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func (c *Config) ProfileNames() []string {
	names := make([]string, 0, len(c.Profiles))
	for name := range c.Profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ServiceAccountKey is the subset of a Google service account JSON key that the
// JWT-bearer flow needs.
type ServiceAccountKey struct {
	Type         string `json:"type"`
	ProjectID    string `json:"project_id"`
	PrivateKeyID string `json:"private_key_id"`
	PrivateKey   string `json:"private_key"`
	ClientEmail  string `json:"client_email"`
	TokenURI     string `json:"token_uri"`
}

// DefaultTokenURI is used when the key file omits token_uri.
const DefaultTokenURI = "https://oauth2.googleapis.com/token"

// ParseServiceAccount validates a service account JSON key.
func ParseServiceAccount(data []byte) (*ServiceAccountKey, error) {
	var key ServiceAccountKey
	if err := json.Unmarshal(data, &key); err != nil {
		return nil, fmt.Errorf("parse service account JSON: %w", err)
	}
	if key.Type != "service_account" {
		kind := key.Type
		if kind == "" {
			kind = "(missing \"type\")"
		}
		return nil, fmt.Errorf(`not a service account key: type is %s.
Download the key from Google Cloud Console > IAM & Admin > Service Accounts > Keys (JSON),
then grant that service account access in Play Console > Users and permissions`, kind)
	}
	if key.ClientEmail == "" || key.PrivateKey == "" {
		return nil, errors.New("service account JSON is missing client_email or private_key")
	}
	if key.TokenURI == "" {
		key.TokenURI = DefaultTokenURI
	}
	return &key, nil
}

// Credentials is a fully resolved set of credentials ready for signing, plus the
// contextual defaults (package, developer id) attached to the profile.
type Credentials struct {
	Source      string // "env" or "profile:<name>"
	Key         *ServiceAccountKey
	Package     string
	DeveloperID string
}

// Resolve resolves credentials with the following precedence:
//  1. Environment variables GPLAY_SERVICE_ACCOUNT_JSON / GPLAY_SERVICE_ACCOUNT_BASE64,
//     else GOOGLE_APPLICATION_CREDENTIALS
//  2. The profile given by the --profile flag, then GPLAY_PROFILE, then default_profile
func Resolve(profileFlag string) (*Credentials, error) {
	envPackage := os.Getenv("GPLAY_PACKAGE")
	envDeveloper := os.Getenv("GPLAY_DEVELOPER_ID")

	if b64 := os.Getenv("GPLAY_SERVICE_ACCOUNT_BASE64"); b64 != "" {
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(b64))
		if err != nil {
			return nil, fmt.Errorf("decode GPLAY_SERVICE_ACCOUNT_BASE64: %w", err)
		}
		key, err := ParseServiceAccount(decoded)
		if err != nil {
			return nil, fmt.Errorf("GPLAY_SERVICE_ACCOUNT_BASE64: %w", err)
		}
		return &Credentials{Source: "env", Key: key, Package: envPackage, DeveloperID: envDeveloper}, nil
	}
	for _, envVar := range []string{"GPLAY_SERVICE_ACCOUNT_JSON", "GOOGLE_APPLICATION_CREDENTIALS"} {
		path := os.Getenv(envVar)
		if path == "" {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", envVar, err)
		}
		key, err := ParseServiceAccount(data)
		if err != nil {
			return nil, fmt.Errorf("%s (%s): %w", envVar, path, err)
		}
		return &Credentials{Source: "env", Key: key, Package: envPackage, DeveloperID: envDeveloper}, nil
	}

	cfg, err := Load()
	if err != nil {
		return nil, err
	}
	name := profileFlag
	if name == "" {
		name = os.Getenv("GPLAY_PROFILE")
	}
	if name == "" {
		name = cfg.DefaultProfile
	}
	if name == "" {
		return nil, errors.New(`no profile configured; run "gplay configure --key <service-account.json>" first`)
	}
	profile, ok := cfg.Profiles[name]
	if !ok {
		path, _ := FilePath()
		return nil, fmt.Errorf("profile %q not found in %s", name, path)
	}

	resolvedPath := profile.ServiceAccount
	if !filepath.IsAbs(resolvedPath) {
		dir, err := Dir()
		if err != nil {
			return nil, err
		}
		resolvedPath = filepath.Join(dir, resolvedPath)
	}
	data, err := os.ReadFile(resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("read service account key for profile %q: %w", name, err)
	}
	key, err := ParseServiceAccount(data)
	if err != nil {
		return nil, fmt.Errorf("profile %q (%s): %w", name, resolvedPath, err)
	}

	pkg := envPackage
	if pkg == "" {
		pkg = profile.Package
	}
	developer := envDeveloper
	if developer == "" {
		developer = profile.DeveloperID
	}
	return &Credentials{
		Source:      "profile:" + name,
		Key:         key,
		Package:     pkg,
		DeveloperID: developer,
	}, nil
}
