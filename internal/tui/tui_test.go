package tui

import (
	"context"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/syjsion/Terminal_DDZ/internal/ai"
	"github.com/syjsion/Terminal_DDZ/internal/ai/local"
	"github.com/syjsion/Terminal_DDZ/internal/config"
	"github.com/syjsion/Terminal_DDZ/internal/game"
)

func testModel(t *testing.T) *Model {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	cfg := config.Default()
	cfg.General.AIDelayMS = 0
	agents := [3]ai.Agent{nil, local.New(local.Normal, 2), local.New(local.Normal, 3)}
	return NewModel(ctx, cancel, cfg, game.NewEngine(game.WithSeed(1)), agents, [3]bool{}, nil)
}

func TestMinimumTerminalMessage(t *testing.T) {
	m := testModel(t)
	_, _ = m.Update(tea.WindowSizeMsg{Width: 70, Height: 20})
	content := m.View().Content
	if !strings.Contains(content, "80x24") || !strings.Contains(content, "70x20") {
		t.Fatalf("small terminal view = %q", content)
	}
}

func TestMenuAndGameRender(t *testing.T) {
	m := testModel(t)
	_, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	if !strings.Contains(m.View().Content, "开始游戏") {
		t.Fatal("menu missing")
	}
	_ = m.startRound()
	view := m.View().Content
	if !strings.Contains(view, "手牌") || !strings.Contains(view, "叫地主") {
		t.Fatalf("game view missing core content: %s", view)
	}
	if strings.Contains(view, " T ") {
		t.Fatal("view uses T for rank 10")
	}
}

func TestStaleAIResultIsDiscarded(t *testing.T) {
	m := testModel(t)
	_ = m.startRound()
	before := m.engine.State()
	_, _ = m.handleAIResult(aiResultMsg{gameID: before.GameID + 1, turnID: before.TurnID, seat: before.CurrentSeat, isBid: true, choice: 3})
	after := m.engine.State()
	if after.TurnID != before.TurnID || len(after.History) != len(before.History) {
		t.Fatal("stale AI result mutated game")
	}
}

func TestHelpAndPanelToggles(t *testing.T) {
	m := testModel(t)
	_ = m.startRound()
	beforeCounter := m.showCounter
	_, _ = m.handleGameKey("r")
	if m.showCounter == beforeCounter {
		t.Fatal("counter did not toggle")
	}
	_, _ = m.handleGameKey("?")
	if m.overlay != overlayHelp || !strings.Contains(m.View().Content, "Space") {
		t.Fatal("help overlay missing")
	}
	_, _ = m.handleOverlayKey("esc")
	if m.overlay != overlayNone {
		t.Fatal("help did not close")
	}
}
