package tui

import (
	"errors"
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/syjsion/Terminal_DDZ/internal/ai"
	"github.com/syjsion/Terminal_DDZ/internal/ai/local"
	"github.com/syjsion/Terminal_DDZ/internal/game"
	"github.com/syjsion/Terminal_DDZ/internal/player"
)

type aiResultMsg struct {
	gameID uint64
	turnID uint64
	seat   int
	isBid  bool
	choice int
	err    error
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case aiResultMsg:
		return m.handleAIResult(msg)
	case tea.KeyPressMsg:
		return m.handleKey(msg.String())
	}
	return m, nil
}

func (m *Model) handleKey(key string) (tea.Model, tea.Cmd) {
	if key == "ctrl+c" {
		m.cancel()
		return m, tea.Quit
	}
	if m.overlay != overlayNone {
		return m.handleOverlayKey(key)
	}
	if key == "q" {
		m.overlay = overlayQuit
		return m, nil
	}
	if m.width > 0 && (m.width < 80 || m.height < 24) {
		return m, nil
	}
	switch m.screen {
	case screenMenu:
		switch key {
		case "up", "k":
			if m.menuCursor > 0 {
				m.menuCursor--
			}
		case "down", "j":
			if m.menuCursor < 2 {
				m.menuCursor++
			}
		case "enter":
			switch m.menuCursor {
			case 0:
				return m, m.startRound()
			case 1:
				m.screen = screenSettings
			case 2:
				m.overlay = overlayQuit
			}
		}
	case screenSettings:
		if key == "esc" || key == "enter" {
			m.screen = screenMenu
		}
	case screenResult:
		if key == "enter" {
			return m, m.startRound()
		}
		if key == "esc" {
			m.screen = screenMenu
		}
	case screenGame:
		return m.handleGameKey(key)
	}
	return m, nil
}

func (m *Model) handleOverlayKey(key string) (tea.Model, tea.Cmd) {
	switch m.overlay {
	case overlayQuit:
		switch key {
		case "y", "Y", "enter":
			m.cancel()
			return m, tea.Quit
		case "n", "N", "esc", "q":
			m.overlay = overlayNone
		}
	case overlayHelp:
		if key == "esc" || key == "?" || key == "enter" {
			m.overlay = overlayNone
		}
	case overlayAIError:
		switch key {
		case "r", "R":
			m.overlay = overlayNone
			m.lastAIError = nil
			return m, m.requestAI()
		case "l", "L":
			difficulty, _ := local.ParseDifficulty(m.cfg.Game.LocalAIDifficulty)
			m.agents[m.failedSeat] = local.New(difficulty, time.Now().UnixNano()+int64(m.failedSeat))
			m.isLLM[m.failedSeat] = false
			m.overlay = overlayNone
			m.status = fmt.Sprintf("Seat %d 本局已切换为本地 AI，新局将恢复 LLM", m.failedSeat)
			return m, m.requestAI()
		case "q", "Q":
			m.overlay = overlayQuit
		}
	}
	return m, nil
}

func (m *Model) handleGameKey(key string) (tea.Model, tea.Cmd) {
	if key == "?" {
		m.overlay = overlayHelp
		return m, nil
	}
	if key == "r" || key == "R" {
		m.showCounter = !m.showCounter
		return m, nil
	}
	if key == "h" || key == "H" {
		m.showHistory = !m.showHistory
		return m, nil
	}
	state := m.engine.State()
	if state.CurrentSeat != 0 || m.aiThinking {
		return m, nil
	}
	if state.Phase == game.PhasePlaying && key == "tab" {
		if m.playMode == playModeCandidates {
			m.playMode = playModeManual
			m.status = "手动选牌：←/→ 移动，Space 选择，Enter 出牌"
		} else {
			m.playMode = playModeCandidates
			m.status = "已切换到推荐候选"
		}
		return m, nil
	}
	if state.Phase == game.PhasePlaying && m.playMode == playModeManual {
		return m.handleManualPlayKey(key, state)
	}
	count := len(m.moves)
	if state.Phase == game.PhaseBidding {
		count = len(m.bids)
	}
	switch key {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor+1 < count {
			m.cursor++
		}
	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		index := int(key[0] - '1')
		if index < count {
			m.cursor = index
		}
	case "space", "p", "P":
		if state.Phase == game.PhasePlaying {
			for _, move := range m.moves {
				if move.IsPass {
					return m, m.applyHumanMove(move.ID)
				}
			}
		}
	case "enter":
		if state.Phase == game.PhaseBidding && m.cursor < len(m.bids) {
			if err := m.engine.ApplyBid(0, m.bids[m.cursor]); err != nil {
				m.status = err.Error()
				return m, nil
			}
			return m, m.advance()
		}
		if state.Phase == game.PhasePlaying && m.cursor < len(m.moves) {
			return m, m.applyHumanMove(m.moves[m.cursor].ID)
		}
	}
	return m, nil
}

func (m *Model) handleManualPlayKey(key string, state game.GameState) (tea.Model, tea.Cmd) {
	hand := state.Players[0].Hand
	switch key {
	case "left":
		if m.handCursor > 0 {
			m.handCursor--
		}
	case "right":
		if m.handCursor+1 < len(hand) {
			m.handCursor++
		}
	case "space":
		if len(hand) > 0 {
			if m.selectedHand[m.handCursor] {
				delete(m.selectedHand, m.handCursor)
			} else {
				m.selectedHand[m.handCursor] = true
			}
		}
	case "backspace", "ctrl+h":
		clear(m.selectedHand)
		m.status = "已清空手选牌"
	case "p", "P":
		for _, move := range m.moves {
			if move.IsPass {
				return m, m.applyHumanMove(move.ID)
			}
		}
		m.status = "当前是首出，不能 PASS"
	case "enter":
		return m, m.applyHumanCards(m.selectedCards())
	}
	return m, nil
}

func (m *Model) applyHumanMove(id int) tea.Cmd {
	if err := m.engine.ApplyMove(0, id); err != nil {
		m.status = err.Error()
		return nil
	}
	return m.advance()
}

func (m *Model) applyHumanCards(cards []game.Card) tea.Cmd {
	if err := m.engine.ApplyCards(0, cards); err != nil {
		m.status = err.Error()
		return nil
	}
	m.status = "出牌成功"
	return m.advance()
}

func (m *Model) requestAI() tea.Cmd {
	state := m.engine.State()
	seat := state.CurrentSeat
	if seat == 0 || state.Finished || m.agents[seat] == nil {
		return nil
	}
	m.aiThinking = true
	view := player.BuildView(state, seat)
	agent := m.agents[seat]
	isLLM := m.isLLM[seat]
	delay := time.Duration(m.cfg.General.AIDelayMS) * time.Millisecond
	gameID, turnID := state.GameID, state.TurnID
	if state.Phase == game.PhaseBidding {
		legal := append([]int(nil), m.engine.LegalBids(seat)...)
		return func() tea.Msg {
			if !isLLM && delay > 0 {
				timer := time.NewTimer(delay)
				defer timer.Stop()
				select {
				case <-m.ctx.Done():
					return aiResultMsg{gameID: gameID, turnID: turnID, seat: seat, isBid: true, err: m.ctx.Err()}
				case <-timer.C:
				}
			}
			choice, err := agent.ChooseBid(m.ctx, view, legal)
			return aiResultMsg{gameID: gameID, turnID: turnID, seat: seat, isBid: true, choice: choice, err: err}
		}
	}
	legal := append([]game.Move(nil), m.engine.LegalMoves(seat)...)
	return func() tea.Msg {
		if !isLLM && delay > 0 {
			timer := time.NewTimer(delay)
			defer timer.Stop()
			select {
			case <-m.ctx.Done():
				return aiResultMsg{gameID: gameID, turnID: turnID, seat: seat, err: m.ctx.Err()}
			case <-timer.C:
			}
		}
		choice, err := agent.ChooseMove(m.ctx, view, legal)
		return aiResultMsg{gameID: gameID, turnID: turnID, seat: seat, choice: choice, err: err}
	}
}

func (m *Model) handleAIResult(result aiResultMsg) (tea.Model, tea.Cmd) {
	state := m.engine.State()
	if result.gameID != state.GameID || result.turnID != state.TurnID || result.seat != state.CurrentSeat {
		return m, nil
	}
	m.aiThinking = false
	if result.err != nil {
		var fallback *ai.FallbackError
		if errors.As(result.err, &fallback) {
			m.status = fallback.Error()
			if m.logger != nil {
				m.logger.Print(fallback.Error())
			}
		} else {
			m.lastAIError = result.err
			m.failedSeat = result.seat
			m.overlay = overlayAIError
			if m.logger != nil {
				m.logger.Printf("Seat %d AI error: %v", result.seat, result.err)
			}
			return m, nil
		}
	}
	var err error
	if result.isBid {
		err = m.engine.ApplyBid(result.seat, result.choice)
	} else {
		err = m.engine.ApplyMove(result.seat, result.choice)
	}
	if err != nil {
		m.lastAIError = err
		m.failedSeat = result.seat
		m.overlay = overlayAIError
		return m, nil
	}
	return m, m.advance()
}
