package config

import (
	"fmt"
	"net/url"
	"strings"
)

type Config struct {
	General   GeneralConfig             `toml:"general"`
	Game      GameConfig                `toml:"game"`
	Player    PlayerConfig              `toml:"player"`
	AI1       AIConfig                  `toml:"ai1"`
	AI2       AIConfig                  `toml:"ai2"`
	Providers map[string]ProviderConfig `toml:"providers"`
}

type GeneralConfig struct {
	Language         string `toml:"language"`
	Theme            string `toml:"theme"`
	ShowCardCounter  bool   `toml:"show_card_counter"`
	ShowHistoryPanel bool   `toml:"show_history_panel"`
	AIDelayMS        int    `toml:"ai_delay_ms"`
}

type GameConfig struct {
	LocalAIDifficulty string `toml:"local_ai_difficulty"`
	LLMFallback       bool   `toml:"llm_fallback"`
	MaxLLMLegalMoves  int    `toml:"max_llm_legal_moves"`
}

type PlayerConfig struct {
	Name string `toml:"name"`
}

type AIConfig struct {
	Name       string `toml:"name"`
	Type       string `toml:"type"`
	Difficulty string `toml:"difficulty"`
	Provider   string `toml:"provider"`
}

type ProviderConfig struct {
	BaseURL        string `toml:"base_url"`
	APIKey         string `toml:"api_key"`
	Model          string `toml:"model"`
	TimeoutSeconds int    `toml:"timeout_seconds"`
}

func Default() Config {
	return Config{
		General:   GeneralConfig{Language: "zh-CN", Theme: "work", ShowCardCounter: true, ShowHistoryPanel: true, AIDelayMS: 350},
		Game:      GameConfig{LocalAIDifficulty: "normal", LLMFallback: true, MaxLLMLegalMoves: 80},
		Player:    PlayerConfig{Name: "你"},
		AI1:       AIConfig{Name: "Worker-01", Type: "local", Difficulty: "normal"},
		AI2:       AIConfig{Name: "Worker-02", Type: "local", Difficulty: "normal"},
		Providers: map[string]ProviderConfig{},
	}
}

func (c Config) Validate() error {
	if c.General.Language != "zh-CN" {
		return fmt.Errorf("general.language: MVP 仅支持 zh-CN")
	}
	if c.General.Theme != "work" {
		return fmt.Errorf("general.theme: MVP 仅支持 work")
	}
	if c.General.AIDelayMS < 0 {
		return fmt.Errorf("general.ai_delay_ms: 不能小于 0")
	}
	if !validDifficulty(c.Game.LocalAIDifficulty) {
		return fmt.Errorf("game.local_ai_difficulty: 必须是 easy、normal 或 hard")
	}
	if c.Game.MaxLLMLegalMoves < 1 {
		return fmt.Errorf("game.max_llm_legal_moves: 必须大于 0")
	}
	if strings.TrimSpace(c.Player.Name) == "" {
		return fmt.Errorf("player.name: 不能为空")
	}
	for name, ai := range map[string]AIConfig{"ai1": c.AI1, "ai2": c.AI2} {
		if strings.TrimSpace(ai.Name) == "" {
			return fmt.Errorf("%s.name: 不能为空", name)
		}
		switch ai.Type {
		case "local":
			if !validDifficulty(ai.Difficulty) {
				return fmt.Errorf("%s.difficulty: 必须是 easy、normal 或 hard", name)
			}
		case "llm":
			if ai.Provider == "" {
				return fmt.Errorf("%s.provider: LLM 玩家必须指定 provider", name)
			}
			provider, ok := c.Providers[ai.Provider]
			if !ok {
				return fmt.Errorf("%s.provider: providers.%s 不存在", name, ai.Provider)
			}
			if err := validateProvider(ai.Provider, provider); err != nil {
				return err
			}
		default:
			return fmt.Errorf("%s.type: 必须是 local 或 llm", name)
		}
	}
	return nil
}

func validDifficulty(value string) bool {
	return value == "easy" || value == "normal" || value == "hard"
}

func validateProvider(name string, p ProviderConfig) error {
	prefix := "providers." + name
	if strings.TrimSpace(p.BaseURL) == "" {
		return fmt.Errorf("%s.base_url: 不能为空", prefix)
	}
	u, err := url.Parse(p.BaseURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("%s.base_url: 必须是有效的 HTTP(S) URL", prefix)
	}
	if strings.TrimSpace(p.APIKey) == "" {
		return fmt.Errorf("%s.api_key: 不能为空", prefix)
	}
	if strings.TrimSpace(p.Model) == "" {
		return fmt.Errorf("%s.model: 不能为空", prefix)
	}
	if p.TimeoutSeconds <= 0 {
		return fmt.Errorf("%s.timeout_seconds: 必须大于 0", prefix)
	}
	return nil
}

func (c Config) AIConfigs() [2]AIConfig { return [2]AIConfig{c.AI1, c.AI2} }

func (c Config) Summary() []string {
	return []string{aiSummary("AI-1", c.AI1, c), aiSummary("AI-2", c.AI2, c)}
}

func aiSummary(label string, a AIConfig, c Config) string {
	if a.Type == "local" {
		return fmt.Sprintf("%s: 本地 / %s", label, a.Difficulty)
	}
	p := c.Providers[a.Provider]
	return fmt.Sprintf("%s: LLM / %s / %s / Key 已配置", label, a.Provider, p.Model)
}
