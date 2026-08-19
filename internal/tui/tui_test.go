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

func humanPlayingModel(t *testing.T) *Model {
	t.Helper()
	m := testModel(t)
	if err := m.engine.StartRound(); err != nil {
		t.Fatal(err)
	}
	for m.engine.State().Phase == game.PhaseBidding {
		state := m.engine.State()
		bid := 0
		if state.CurrentSeat == 0 {
			bid = 3
		}
		if err := m.engine.ApplyBid(state.CurrentSeat, bid); err != nil {
			t.Fatal(err)
		}
	}
	m.screen = screenGame
	m.width, m.height = 100, 30
	m.refreshChoices()
	return m
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

func TestManualSelectionControlsAndLegalSubmit(t *testing.T) {
	m := humanPlayingModel(t)
	_, _ = m.handleGameKey("tab")
	if m.playMode != playModeManual {
		t.Fatal("Tab did not enter manual mode")
	}
	_, _ = m.handleGameKey("space")
	if !m.selectedHand[0] {
		t.Fatal("Space did not select focused card")
	}
	_, _ = m.handleGameKey("right")
	if m.handCursor != 1 {
		t.Fatal("Right did not move hand cursor")
	}
	_, _ = m.handleGameKey("backspace")
	if len(m.selectedHand) != 0 {
		t.Fatal("Backspace did not clear selection")
	}

	move := m.moves[0]
	wanted := game.RankCounts(move.Cards)
	for index, card := range m.engine.State().Players[0].Hand {
		if wanted[card.Rank] > 0 {
			m.selectedHand[index] = true
			wanted[card.Rank]--
		}
	}
	beforeTurn := m.engine.State().TurnID
	_, _ = m.handleGameKey("enter")
	if m.engine.State().TurnID == beforeTurn {
		t.Fatalf("manual legal move was not applied: %s", m.status)
	}
}

func TestManualInvalidSelectionKeepsTurn(t *testing.T) {
	m := humanPlayingModel(t)
	_, _ = m.handleGameKey("tab")
	hand := m.engine.State().Players[0].Hand
	first, second := 0, -1
	for i := 1; i < len(hand); i++ {
		if hand[i].Rank != hand[first].Rank {
			second = i
			break
		}
	}
	if second < 0 {
		t.Fatal("test hand does not contain two ranks")
	}
	m.selectedHand[first], m.selectedHand[second] = true, true
	before := m.engine.State().TurnID
	_, _ = m.handleGameKey("enter")
	if m.engine.State().TurnID != before || !strings.Contains(m.status, "不构成合法牌型") {
		t.Fatalf("invalid selection changed turn or lacked feedback: %s", m.status)
	}
}

func TestGameLayoutInformationOrder(t *testing.T) {
	m := humanPlayingModel(t)
	for _, size := range []tea.WindowSizeMsg{{Width: 80, Height: 24}, {Width: 110, Height: 36}} {
		_, _ = m.Update(size)
		content := m.View().Content
		bottom := strings.Index(content, "底牌：")
		counter := strings.Index(content, "记牌器：")
		history := strings.Index(content, "出牌历史：")
		action := strings.Index(content, "推荐候选")
		hand := strings.Index(content, "你的身份：")
		if !(bottom >= 0 && bottom < counter && counter < history && history < action && action < hand) {
			t.Fatalf("unexpected layout order at %dx%d: bottom=%d counter=%d history=%d action=%d hand=%d\n%s", size.Width, size.Height, bottom, counter, history, action, hand, content)
		}
	}
	m.showCounter, m.showHistory = false, false
	content := m.View().Content
	if strings.Index(content, "记牌器：[R 展开]") > strings.Index(content, "出牌历史：[H 展开]") || strings.Index(content, "出牌历史：[H 展开]") > strings.Index(content, "你的身份：") {
		t.Fatal("collapsed regions changed layout order")
	}
}
