package tui

import (
	"context"
	"errors"
	"fmt"
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
		seats := strings.Index(content, "Seat 1 ")
		history := strings.Index(content, "出牌历史：")
		action := strings.Index(content, "推荐候选")
		hand := strings.Index(content, "你的身份：")
		if !(bottom >= 0 && bottom < counter && counter < seats && seats < history && history < action && action < hand) {
			t.Fatalf("unexpected layout order at %dx%d: bottom=%d counter=%d seats=%d history=%d action=%d hand=%d\n%s", size.Width, size.Height, bottom, counter, seats, history, action, hand, content)
		}
	}
	m.showCounter, m.showHistory = false, false
	content := m.View().Content
	if strings.Index(content, "记牌器：[R 展开]") > strings.Index(content, "出牌历史：[H 展开]") || strings.Index(content, "出牌历史：[H 展开]") > strings.Index(content, "你的身份：") {
		t.Fatal("collapsed regions changed layout order")
	}
}

func TestHistoryKeepsSixAndAdaptsToTerminalHeight(t *testing.T) {
	m := testModel(t)
	state := game.GameState{Phase: game.PhasePlaying}
	state.Players[0].Name = "你"
	state.Players[1].Name = "AI-1"
	state.Players[2].Name = "AI-2"
	for number := 1; number <= 8; number++ {
		state.History = append(state.History, game.ActionRecord{
			Number: number,
			Kind:   game.ActionPlay,
			Seat:   number % 3,
			Move:   game.Move{Type: game.Single, Cards: []game.Card{{Rank: game.Rank3}}},
		})
	}

	m.height = 30
	tall := m.viewHistory(state)
	if strings.Contains(tall, "#01") || strings.Contains(tall, "#02") {
		t.Fatalf("tall history retained old rows:\n%s", tall)
	}
	for number := 3; number <= 8; number++ {
		if !strings.Contains(tall, fmt.Sprintf("#%02d", number)) {
			t.Fatalf("tall history missing #%02d:\n%s", number, tall)
		}
	}

	m.height = 24
	compact := m.viewHistory(state)
	if !strings.Contains(compact, "显示 3/6，增高窗口查看") {
		t.Fatalf("compact history missing indicator:\n%s", compact)
	}
	for number := 3; number <= 5; number++ {
		if strings.Contains(compact, fmt.Sprintf("#%02d", number)) {
			t.Fatalf("compact history displayed old #%02d:\n%s", number, compact)
		}
	}
	for number := 6; number <= 8; number++ {
		if !strings.Contains(compact, fmt.Sprintf("#%02d", number)) {
			t.Fatalf("compact history missing #%02d:\n%s", number, compact)
		}
	}
	if len(state.History) != 8 {
		t.Fatalf("viewHistory mutated engine history: %d", len(state.History))
	}
}

func TestStartRoundRestoresConfiguredLLMAfterManualDowngrade(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	cfg := config.Default()
	cfg.General.AIDelayMS = 0
	original := local.New(local.Normal, 11)
	agents := [3]ai.Agent{nil, original, local.New(local.Normal, 12)}
	isLLM := [3]bool{false, true, false}
	m := NewModel(ctx, cancel, cfg, game.NewEngine(game.WithSeed(1)), agents, isLLM, nil)

	m.agents[1] = local.New(local.Easy, 99)
	m.isLLM[1] = false
	m.lastAIError = errors.New("old LLM error")
	m.failedSeat = 1

	_ = m.startRound()
	if m.agents[1] != original || !m.isLLM[1] {
		t.Fatal("new round did not restore the configured LLM agent")
	}
	if m.lastAIError != nil || m.failedSeat != 0 {
		t.Fatalf("new round retained stale AI error: %v, seat=%d", m.lastAIError, m.failedSeat)
	}
}

func TestManualDowngradeMessageIsLimitedToCurrentRound(t *testing.T) {
	m := testModel(t)
	m.isLLM[1] = true
	m.overlay = overlayAIError
	m.failedSeat = 1
	_, _ = m.handleOverlayKey("L")
	if m.isLLM[1] || !strings.Contains(m.status, "本局已切换") || !strings.Contains(m.status, "新局将恢复 LLM") {
		t.Fatalf("manual downgrade state or message is unclear: isLLM=%v status=%q", m.isLLM[1], m.status)
	}
}
