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
