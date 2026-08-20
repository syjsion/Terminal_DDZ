package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMissingDefaultUsesDefaults(t *testing.T) {
	dir := t.TempDir()
	old, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)
	cfg, used, err := Load("")
	if err != nil || used != "" || cfg.AI1.Type != "local" || cfg.Game.MaxLLMLegalMoves != 80 {
		t.Fatalf("Load defaults = %+v, %q, %v", cfg, used, err)
	}
}

func TestExplicitMissingIsError(t *testing.T) {
	if _, _, err := Load(filepath.Join(t.TempDir(), "missing.toml")); err == nil {
		t.Fatal("expected missing config error")
	}
}

func TestLLMProviderValidation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	data := []byte("[ai1]\ntype='llm'\nprovider='x'\n")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "providers.x") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUnusedEmptyProviderDoesNotFail(t *testing.T) {
	cfg := Default()
	cfg.Providers["unused"] = ProviderConfig{}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("unused provider rejected: %v", err)
	}
}

func TestLLMAPIKeyCanLoadFromEnvironment(t *testing.T) {
	t.Setenv("TERMINAL_DDZ_TEST_KEY", "secret-from-env")
	path := filepath.Join(t.TempDir(), "config.toml")
	data := []byte(`
[ai1]
type = "llm"
provider = "x"

[providers.x]
base_url = "https://example.com/v1"
api_key_env = "TERMINAL_DDZ_TEST_KEY"
model = "test-model"
timeout_seconds = 10
`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, _, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	provider := cfg.Providers["x"]
	if provider.APIKey != "secret-from-env" || provider.APIKeyEnv != "" {
		t.Fatalf("environment key was not resolved safely: %+v", provider)
	}
}

func TestLLMAPIKeyEnvironmentErrorsAreActionable(t *testing.T) {
	tests := []struct {
		name       string
		apiKeyLine string
		want       string
	}{
		{name: "missing environment", apiKeyLine: `api_key_env = "TERMINAL_DDZ_MISSING_TEST_KEY"`, want: "未设置或为空"},
		{name: "invalid environment name", apiKeyLine: `api_key_env = "BAD-NAME"`, want: "不是有效的环境变量名"},
		{name: "ambiguous source", apiKeyLine: "api_key = \"direct\"\napi_key_env = \"TERMINAL_DDZ_TEST_KEY\"", want: "只能配置一个"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.toml")
			data := []byte("[ai1]\ntype='llm'\nprovider='x'\n[providers.x]\nbase_url='https://example.com/v1'\n" + tc.apiKeyLine + "\nmodel='test'\ntimeout_seconds=10\n")
			if err := os.WriteFile(path, data, 0o600); err != nil {
				t.Fatal(err)
			}
			_, _, err := Load(path)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Load error = %v, want %q", err, tc.want)
			}
		})
	}
}
