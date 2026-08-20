package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

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
	if err := resolveAPIKeys(&cfg, os.LookupEnv); err != nil {
		return Config{}, path, fmt.Errorf("config error: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, path, fmt.Errorf("config error: %w", err)
	}
	return cfg, path, nil
}

func resolveAPIKeys(cfg *Config, lookup func(string) (string, bool)) error {
	resolved := make(map[string]bool, 2)
	for _, aiConfig := range cfg.AIConfigs() {
		if aiConfig.Type != "llm" || resolved[aiConfig.Provider] {
			continue
		}
		resolved[aiConfig.Provider] = true
		provider, ok := cfg.Providers[aiConfig.Provider]
		if !ok {
			continue
		}
		envName := strings.TrimSpace(provider.APIKeyEnv)
		if envName == "" {
			continue
		}
		if strings.TrimSpace(provider.APIKey) != "" {
			return fmt.Errorf("providers.%s: api_key 和 api_key_env 只能配置一个", aiConfig.Provider)
		}
		if !validEnvName(envName) {
			return fmt.Errorf("providers.%s.api_key_env: %q 不是有效的环境变量名", aiConfig.Provider, envName)
		}
		apiKey, exists := lookup(envName)
		if !exists || strings.TrimSpace(apiKey) == "" {
			return fmt.Errorf("providers.%s.api_key_env: 环境变量 %s 未设置或为空", aiConfig.Provider, envName)
		}
		provider.APIKey = apiKey
		provider.APIKeyEnv = ""
		cfg.Providers[aiConfig.Provider] = provider
	}
	return nil
}
