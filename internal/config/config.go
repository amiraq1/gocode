package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	BaseURL  string `json:"base_url"`
	APIKey   string `json:"api_key"`
	Model    string `json:"model"`
	MaxSteps int    `json:"max_steps"`
}

// Load: القيم الافتراضية ← config.json ← متغيرات البيئة (الأولوية للأخير)
func Load() (*Config, error) {
	cfg := &Config{BaseURL: "https://api.openai.com/v1", Model: "gpt-4o-mini", MaxSteps: 20}

	path := os.Getenv("GOCODE_CONFIG")
	if path == "" {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, ".config", "gocode", "config.json")
	}
	if data, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("config %s: %w", path, err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	if v := os.Getenv("GOCODE_BASE_URL"); v != "" {
		cfg.BaseURL = v
	}
	if v := os.Getenv("GOCODE_API_KEY"); v != "" {
		cfg.APIKey = v
	} else if cfg.APIKey == "" {
		cfg.APIKey = os.Getenv("OPENAI_API_KEY")
	}
	if v := os.Getenv("GOCODE_MODEL"); v != "" {
		cfg.Model = v
	}
	if cfg.APIKey == "" {
		return nil, errors.New("no API key: set GOCODE_API_KEY or api_key in " + path)
	}
	if cfg.MaxSteps <= 0 {
		cfg.MaxSteps = 20
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	return cfg, nil
}
