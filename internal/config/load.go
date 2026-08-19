package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/pelletier/go-toml/v2"
)

func Load(path string) (Config, string, error) {
	explicit := path != ""
	if path == "" {
		path = "config.toml"
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) && !explicit {
			cfg := Default()
			return cfg, "", cfg.Validate()
		}
		return Config{}, path, fmt.Errorf("读取配置 %s: %w", path, err)
	}
	cfg := Default()
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return Config{}, path, fmt.Errorf("解析配置 %s: %w", path, err)
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, path, fmt.Errorf("config error: %w", err)
	}
	return cfg, path, nil
}
