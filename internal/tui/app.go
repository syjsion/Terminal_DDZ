package tui

import (
	"context"
	"fmt"
	"log"

	tea "charm.land/bubbletea/v2"

	"github.com/syjsion/Terminal_DDZ/internal/ai"
	"github.com/syjsion/Terminal_DDZ/internal/ai/llm"
	"github.com/syjsion/Terminal_DDZ/internal/ai/local"
	"github.com/syjsion/Terminal_DDZ/internal/config"
	"github.com/syjsion/Terminal_DDZ/internal/game"
)

func Run(ctx context.Context, cfg config.Config, seed int64, logger *log.Logger) error {
	names := [3]string{cfg.Player.Name, cfg.AI1.Name, cfg.AI2.Name}
	engine := game.NewEngine(game.WithSeed(seed), game.WithNames(names))
	agents := [3]ai.Agent{}
	isLLM := [3]bool{}
	configs := [2]config.AIConfig{cfg.AI1, cfg.AI2}
	for i, aiConfig := range configs {
		seat := i + 1
		fallbackDifficulty, _ := local.ParseDifficulty(cfg.Game.LocalAIDifficulty)
		fallback := local.New(fallbackDifficulty, seed+int64(seat)*1009)
		if aiConfig.Type == "local" {
			difficulty, err := local.ParseDifficulty(aiConfig.Difficulty)
			if err != nil {
				return fmt.Errorf("ai%d: %w", seat, err)
			}
			agents[seat] = local.New(difficulty, seed+int64(seat)*1009)
			continue
		}
		provider := cfg.Providers[aiConfig.Provider]
		agents[seat] = llm.NewAgent(aiConfig.Provider, provider, cfg.Game.MaxLLMLegalMoves, fallback, cfg.Game.LLMFallback, nil)
		isLLM[seat] = true
	}
	modelCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	model := NewModel(modelCtx, cancel, cfg, engine, agents, isLLM, logger)
	_, err := tea.NewProgram(model, tea.WithContext(modelCtx)).Run()
	return err
}
