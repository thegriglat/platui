// Package config loads and validates the platui YAML config (contract C1 v1).
package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const DefaultBaseURL = "https://api.plane.so"
const DefaultBranchTemplate = "{identifier}-{sequence}-{slug}"

// Branch configures branch, worktree and commit naming.
type Branch struct {
	Template       string `yaml:"template"`
	Base           string `yaml:"base"`
	Checkout       *bool  `yaml:"checkout"`
	Worktree       bool   `yaml:"worktree"`
	WorktreeDir    string `yaml:"worktree_dir"`
	CommitTemplate string `yaml:"commit_template"`
	Remote         string `yaml:"remote"`
}

// Config is the platui configuration (C1 v5).
type Config struct {
	BaseURL        string    `yaml:"base_url"`
	APIKey         string    `yaml:"api_key"`
	APIKeyEnv      string    `yaml:"api_key_env"`
	Workspace      string    `yaml:"workspace"`
	DefaultProject string    `yaml:"default_project"`
	WebURL         string    `yaml:"web_url"`
	Branch         Branch    `yaml:"branch"`
	Workflow       Workflow  `yaml:"workflow"`
	GitLab         GitLab    `yaml:"gitlab"`
	Filters        []Filter  `yaml:"filters"`
	Theme          Theme     `yaml:"theme"`
	Editor         string    `yaml:"editor"`
	Refresh        Refresh   `yaml:"refresh"`
	Cache          Cache     `yaml:"cache"`
	Tabs           Tabs      `yaml:"tabs"`
	UI             UI        `yaml:"ui"`
	Commands       []Command `yaml:"commands"`
	Todo           Todo      `yaml:"todo"`
	HTTP           HTTP      `yaml:"http"`
}

// ErrNoAPIKey means the API key is unset or empty (errors.Is).
var ErrNoAPIKey = errors.New("api key is not set")

func errHome(err error) error {
	return fmt.Errorf("config: cannot determine home directory: %w", err)
}

// DefaultPath returns $XDG_CONFIG_HOME/platui/config.yml or ~/.config/platui/config.yml.
func DefaultPath() (string, error) {
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "platui", "config.yml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", errHome(err)
	}
	return filepath.Join(home, ".config", "platui", "config.yml"), nil
}

// Load reads and parses the config file at path.
func Load(path string) (*Config, error) { return load(path, true) }

// LoadNoKey is like Load but skips checking and resolving the API key (todo, prompt).
func LoadNoKey(path string) (*Config, error) { return load(path, false) }

func load(path string, needKey bool) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("config file not found: %s (create it or pass --config <path>)", path)
		}
		return nil, fmt.Errorf("config: read %s: %w (use --config <path> to choose another file)", path, err)
	}
	c, err := parse(data, needKey)
	if err != nil {
		return nil, fmt.Errorf("config %s: %w", path, err)
	}
	return c, nil
}

// Parse decodes YAML, applies defaults, resolves the API key and validates.
func Parse(data []byte) (*Config, error) { return parse(data, true) }

func parse(data []byte, needKey bool) (*Config, error) {
	var c Config
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&c); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("invalid YAML: %w", err)
	}
	c.applyDefaults()
	if err := c.validate(needKey); err != nil {
		return nil, err
	}
	if !needKey {
		return &c, nil
	}
	if c.APIKeyEnv != "" {
		c.APIKey = os.Getenv(c.APIKeyEnv)
	}
	if c.APIKey == "" {
		return nil, fmt.Errorf("api key is empty: %w", ErrNoAPIKey)
	}
	return &c, nil
}
