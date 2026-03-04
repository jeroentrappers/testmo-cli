package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	URL   string `yaml:"url"`
	Token string `yaml:"token"`
}

const configFileName = ".testmo.yaml"

// Load returns config from env vars (priority) then config file fallback.
func Load() (*Config, error) {
	cfg := &Config{}

	// Try config file first (lower priority)
	if f, err := findConfigFile(); err == nil {
		data, err := os.ReadFile(f)
		if err == nil {
			_ = yaml.Unmarshal(data, cfg)
		}
	}

	// Env vars override
	if v := os.Getenv("TESTMO_URL"); v != "" {
		cfg.URL = v
	}
	if v := os.Getenv("TESTMO_TOKEN"); v != "" {
		cfg.Token = v
	}
	if v := os.Getenv("TESTMO_API_TOKEN"); v != "" {
		cfg.Token = v
	}

	// Normalize URL
	cfg.URL = strings.TrimRight(cfg.URL, "/")
	if cfg.URL != "" && !strings.HasPrefix(cfg.URL, "http") {
		cfg.URL = "https://" + cfg.URL
	}

	return cfg, nil
}

// Validate checks that required config is present.
func (c *Config) Validate() error {
	if c.URL == "" {
		return fmt.Errorf("testmo URL not configured. Run 'testmo init' or set TESTMO_URL")
	}
	if c.Token == "" {
		return fmt.Errorf("testmo token not configured. Run 'testmo init' or set TESTMO_TOKEN")
	}
	return nil
}

// Save writes config to .testmo.yaml in the current directory.
func (c *Config) Save() error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(configFileName, data, 0600)
}

// BaseURL returns the API base URL (e.g., https://foo.testmo.net/api/v1).
func (c *Config) BaseURL() string {
	return c.URL + "/api/v1"
}

func findConfigFile() (string, error) {
	// Check current directory first, then walk up
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		p := filepath.Join(dir, configFileName)
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("config file not found")
}
