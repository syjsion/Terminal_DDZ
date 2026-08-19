package tui

import (
	"context"
	"log"

	tea "charm.land/bubbletea/v2"

	"github.com/syjsion/Terminal_DDZ/internal/ai"
	"github.com/syjsion/Terminal_DDZ/internal/config"
	"github.com/syjsion/Terminal_DDZ/internal/game"
)

type screen uint8

const (
	screenMenu screen = iota
	screenSettings
	screenGame
	screenResult
)

type overlay uint8

const (
	overlayNone overlay = iota
	overlayHelp
	overlayQuit
	overlayAIError
)

type SessionStats struct {
	Rounds        int
	HumanWins     int
	LandlordGames int
	LandlordWins  int
	FarmerGames   int
	FarmerWins    int
	Bombs         int
}

type Model struct {
	ctx    context.Context
	cancel context.CancelFunc
	cfg    config.Config
	engine *game.Engine
	agents [3]ai.Agent
	isLLM  [3]bool
	logger *log.Logger

	screen      screen
	overlay     overlay
	menuCursor  int
	cursor      int
	width       int
	height      int
	moves       []game.Move
	bids        []int
	showCounter bool
	showHistory bool
	aiThinking  bool
	status      string
	lastAIError error
	failedSeat  int
	stats       SessionStats
	countedGame uint64
}

func NewModel(ctx context.Context, cancel context.CancelFunc, cfg config.Config, engine *game.Engine, agents [3]ai.Agent, isLLM [3]bool, logger *log.Logger) *Model {
	return &Model{
		ctx: ctx, cancel: cancel, cfg: cfg, engine: engine, agents: agents, isLLM: isLLM, logger: logger,
		screen: screenMenu, showCounter: cfg.General.ShowCardCounter, showHistory: cfg.General.ShowHistoryPanel,
	}
}

func (m *Model) Init() tea.Cmd { return nil }

func (m *Model) refreshChoices() {
	m.cursor = 0
	m.moves = nil
	m.bids = nil
	state := m.engine.State()
	if state.CurrentSeat != 0 {
		return
	}
	if state.Phase == game.PhaseBidding {
		m.bids = m.engine.LegalBids(0)
	} else if state.Phase == game.PhasePlaying {
		m.moves = m.engine.LegalMoves(0)
	}
}

func (m *Model) startRound() tea.Cmd {
	if err := m.engine.StartRound(); err != nil {
		m.status = err.Error()
		return nil
	}
	m.screen = screenGame
	m.overlay = overlayNone
	m.status = "新一局开始"
	m.refreshChoices()
	return m.advance()
}

func (m *Model) advance() tea.Cmd {
	state := m.engine.State()
	if state.Finished {
		m.recordResult(state)
		m.screen = screenResult
		m.aiThinking = false
		return nil
	}
	m.refreshChoices()
	if state.CurrentSeat != 0 {
		return m.requestAI()
	}
	return nil
}

func (m *Model) recordResult(state game.GameState) {
	if m.countedGame == state.GameID {
		return
	}
	m.countedGame = state.GameID
	m.stats.Rounds++
	m.stats.Bombs += state.Bombs
	humanWon := (state.LandlordSeat == 0 && state.WinnerTeam == game.TeamLandlord) || (state.LandlordSeat != 0 && state.WinnerTeam == game.TeamFarmers)
	if humanWon {
		m.stats.HumanWins++
	}
	if state.LandlordSeat == 0 {
		m.stats.LandlordGames++
		if humanWon {
			m.stats.LandlordWins++
		}
	} else {
		m.stats.FarmerGames++
		if humanWon {
			m.stats.FarmerWins++
		}
	}
}
