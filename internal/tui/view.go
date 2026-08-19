package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/syjsion/Terminal_DDZ/internal/game"
)

var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	dimStyle    = lipgloss.NewStyle().Faint(true)
	selectStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2"))
	warnStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("3"))
	errorStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("1"))
)

func (m *Model) View() tea.View {
	var content string
	if m.width > 0 && (m.width < 80 || m.height < 24) {
		content = fmt.Sprintf("终端窗口太小。\n最小尺寸：80x24\n当前尺寸：%dx%d\n请调整窗口大小后继续。", m.width, m.height)
	} else if m.overlay == overlayHelp {
		content = m.viewHelp()
	} else {
		switch m.screen {
		case screenMenu:
			content = m.viewMenu()
		case screenSettings:
			content = m.viewSettings()
		case screenGame:
			content = m.viewGame()
		case screenResult:
			content = m.viewResult()
		}
		if m.overlay == overlayQuit {
			content += "\n\n" + warnStyle.Render("确认退出？ [Y/Enter] 退出  [N/Esc] 返回")
		}
		if m.overlay == overlayAIError {
			content += "\n\n" + errorStyle.Render(fmt.Sprintf("AI 请求失败：%v", m.lastAIError))
			content += "\n[R] 重试  [L] 本局切换本地 AI  [Q] 退出"
		}
	}
	view := tea.NewView(content)
	view.AltScreen = true
	view.WindowTitle = "Terminal 斗地主"
	return view
}

func (m *Model) divider() string {
	width := m.width
	if width <= 0 || width > 96 {
		width = 96
	}
	return dimStyle.Render(strings.Repeat("-", width))
}

func (m *Model) viewMenu() string {
	items := []string{"开始游戏", "配置信息", "退出"}
	var b strings.Builder
	b.WriteString(titleStyle.Render("Terminal 斗地主"))
	b.WriteString("\n\n纯终端三人斗地主 · Human vs AI vs AI\n\n")
	for i, item := range items {
		prefix := "  "
		line := item
		if i == m.menuCursor {
			prefix = "> "
			line = selectStyle.Render(item)
		}
		fmt.Fprintf(&b, "%s%s\n", prefix, line)
	}
	fmt.Fprintf(&b, "\n%s\n↑/↓ 或 j/k 选择 · Enter 确认 · Q 退出", m.divider())
	if m.stats.Rounds > 0 {
		fmt.Fprintf(&b, "\n本次运行：%d 局，胜 %d 局，炸弹 %d 次", m.stats.Rounds, m.stats.HumanWins, m.stats.Bombs)
		fmt.Fprintf(&b, "\n地主：%d/%d（%.0f%%） · 农民：%d/%d（%.0f%%）", m.stats.LandlordWins, m.stats.LandlordGames, winRate(m.stats.LandlordWins, m.stats.LandlordGames), m.stats.FarmerWins, m.stats.FarmerGames, winRate(m.stats.FarmerWins, m.stats.FarmerGames))
	}
	return b.String()
}

func (m *Model) viewSettings() string {
	lines := m.cfg.Summary()
	return fmt.Sprintf("%s\n\n%s\n%s\n记牌器：%s\n历史面板：%s\n主题：work\n界面语言：简体中文\nAI 延迟：%dms\nLLM 失败回退：%s\n\nAPI Key 仅显示配置状态，不会在界面或日志中输出。\n\n[Enter/Esc] 返回",
		titleStyle.Render("配置信息"), lines[0], lines[1], onOff(m.cfg.General.ShowCardCounter), onOff(m.cfg.General.ShowHistoryPanel), m.cfg.General.AIDelayMS, onOff(m.cfg.Game.LLMFallback))
}

func (m *Model) viewGame() string {
	state := m.engine.State()
	var b strings.Builder
	fmt.Fprintf(&b, "%s  第 %d 局  倍率 x%d  轮到：Seat %d %s\n", titleStyle.Render("Terminal 斗地主"), state.Round, state.Multiplier, state.CurrentSeat, state.Players[state.CurrentSeat].Name)
	if state.LandlordSeat >= 0 {
		fmt.Fprintf(&b, "底牌：%s\n", game.CardsString(state.BottomCards))
		if m.showCounter {
			b.WriteString(m.viewCounter(state))
		} else {
			b.WriteString("记牌器：[R 展开]\n")
		}
	} else {
		b.WriteString("底牌：叫地主后公开\n记牌器：叫地主后启用\n")
	}
	for seat := 1; seat < 3; seat++ {
		p := state.Players[seat]
		thinking := ""
		if m.aiThinking && state.CurrentSeat == seat {
			if m.isLLM[seat] {
				thinking = " [LLM 思考中...]"
			} else {
				thinking = " [思考中...]"
			}
		}
		separator := "    |    "
		if seat == 2 {
			separator = "\n"
		}
		fmt.Fprintf(&b, "Seat %d %s [%s] %d 张%s%s", seat, p.Name, p.Role, len(p.Hand), thinking, separator)
	}
	b.WriteString(m.divider() + "\n")
	if state.CurrentTrick.LastMove != nil {
		lead := state.CurrentTrick.LeadSeat
		fmt.Fprintf(&b, "当前目标：%s  ·  %s\n", state.CurrentTrick.LastMove.Label(), state.Players[lead].Name)
	} else if state.Phase == game.PhaseBidding {
		b.WriteString("当前阶段：叫地主\n")
	} else {
		b.WriteString("当前目标：新一轮首出\n")
	}
	if m.showHistory {
		b.WriteString(m.viewHistory(state))
	} else {
		b.WriteString("出牌历史：[H 展开]\n")
	}
	b.WriteString(m.divider() + "\n")

	if state.Phase == game.PhaseBidding {
		b.WriteString("叫地主：\n")
		if state.CurrentSeat == 0 {
			for i, bid := range m.bids {
				label := fmt.Sprintf("[%d] %s", i+1, bidLabel(bid))
				if i == m.cursor {
					fmt.Fprintf(&b, "> %s\n", selectStyle.Render(label))
				} else {
					fmt.Fprintf(&b, "  %s\n", label)
				}
			}
		} else {
			b.WriteString("等待 AI 叫分...\n")
		}
	} else if state.Phase == game.PhasePlaying {
		if state.CurrentSeat == 0 && m.playMode == playModeManual {
			b.WriteString("手动选牌：" + m.manualSelectionSummary(state) + "\n")
			b.WriteString("←/→ 移动 · Space 选择 · Enter 出牌 · Backspace 清空 · P PASS\n")
		} else if state.CurrentSeat == 0 {
			b.WriteString("推荐候选（Tab 切换手选）：\n")
			pageSize := 6
			if m.height <= 26 {
				pageSize = 3
			}
			start, end := visibleRange(m.cursor, len(m.moves), pageSize)
			for i := start; i < end; i++ {
				move := m.moves[i]
				typeLabel := handTypeLabel(move.Type)
				if move.IsPass {
					typeLabel = "过牌"
				}
				label := fmt.Sprintf("[%d] %s（%s）", i+1, move.Label(), typeLabel)
				if i == 0 && !move.IsPass {
					label += " [推荐]"
				}
				if i == m.cursor {
					fmt.Fprintf(&b, "> %s\n", selectStyle.Render(label))
				} else {
					fmt.Fprintf(&b, "  %s\n", label)
				}
			}
			fmt.Fprintf(&b, "候选 %d-%d / %d\n", start+1, end, len(m.moves))
		} else {
			b.WriteString("等待 AI 选择合法动作...\n")
		}
	}
	b.WriteString(m.divider() + "\n")
	fmt.Fprintf(&b, "你的身份：%s  ·  手牌 [%d]\n", state.Players[0].Role, len(state.Players[0].Hand))
	b.WriteString(m.renderHumanHand(state) + "\n")
	if m.status != "" {
		fmt.Fprintf(&b, "%s\n", warnStyle.Render(m.status))
	}
	if state.Phase == game.PhasePlaying {
		b.WriteString("Tab 候选/手选 · ↑↓/jk 候选 · P PASS · R 记牌 · H 历史 · ? 帮助 · Q 退出")
	} else {
		b.WriteString("↑↓/jk 选择 · Enter 确认 · R 记牌 · H 历史 · ? 帮助 · Q 退出")
	}
	return b.String()
}

func (m *Model) viewCounter(state game.GameState) string {
	counter := game.BuildCounter(state, 0)
	var b strings.Builder
	b.WriteString("记牌器：")
	for i, rank := range game.AllRanks {
		if m.width < 100 && i == 8 {
			b.WriteString("\n        ")
		}
		fmt.Fprintf(&b, " %s:%d", rank, counter.UnknownByRank[rank])
	}
	fmt.Fprintf(&b, "\n炸弹：%d · 王炸可能：%s\n", counter.BombsPlayed, yesNo(counter.RocketPossible))
	return b.String()
}

func (m *Model) viewHistory(state game.GameState) string {
	var b strings.Builder
	history := make([]game.ActionRecord, 0, len(state.History))
	for _, action := range state.History {
		if state.Phase == game.PhaseBidding || action.Kind == game.ActionPlay {
			history = append(history, action)
		}
	}
	if len(history) > 6 {
		history = history[len(history)-6:]
	}
	retained := len(history)
	visible := retained
	if m.height <= 26 && visible > 3 {
		visible = 3
		history = history[retained-visible:]
		fmt.Fprintf(&b, "出牌历史（显示 %d/%d，增高窗口查看）：", visible, retained)
	} else {
		b.WriteString("出牌历史：")
	}
	if len(history) == 0 {
		b.WriteString("暂无记录\n")
		return b.String()
	}
	b.WriteByte('\n')
	for i, action := range history {
		name := state.Players[action.Seat].Name
		if action.Kind == game.ActionBid {
			fmt.Fprintf(&b, "#%02d %-10s 叫分 %s", action.Number, name, bidLabel(action.Bid))
		} else {
			fmt.Fprintf(&b, "#%02d %-10s %s", action.Number, name, action.Move.Label())
		}
		if i+1 < len(history) {
			b.WriteByte('\n')
		}
	}
	b.WriteByte('\n')
	return b.String()
}

func (m *Model) viewHelp() string {
	return fmt.Sprintf("%s\n\n候选模式\n  ↑/↓ 或 j/k  移动候选\n  Enter        出牌\n  Space / P    PASS（仅跟牌时）\n  1-9          快速定位候选\n\n手选模式\n  Tab          切换候选/手选\n  ←/→          移动手牌光标\n  Space        选择/取消牌\n  Enter        提交所选牌\n  Backspace    清空选择\n  P            PASS\n\n通用\n  R 记牌器 · H 历史 · ? 帮助 · Q 退出 · Esc 返回\n\n牌力：3 < 4 < 5 < 6 < 7 < 8 < 9 < 10 < J < Q < K < A < 2 < SJ < BJ\n\n[Esc/?/Enter] 返回游戏", titleStyle.Render("帮助"))
}

func (m *Model) renderHumanHand(state game.GameState) string {
	hand := state.Players[0].Hand
	if m.playMode != playModeManual || state.CurrentSeat != 0 || state.Phase != game.PhasePlaying {
		return game.CardsString(hand)
	}
	parts := make([]string, len(hand))
	for index, card := range hand {
		label := card.String()
		if m.selectedHand[index] {
			label = "[" + label + "]"
		}
		if index == m.handCursor {
			label = selectStyle.Render(">" + label + "<")
		} else if m.selectedHand[index] {
			label = selectStyle.Render(label)
		}
		parts[index] = label
	}
	return strings.Join(parts, " ")
}

func (m *Model) manualSelectionSummary(state game.GameState) string {
	cards := m.selectedCards()
	if len(cards) == 0 {
		return "尚未选择"
	}
	move, err := game.AnalyzeMove(cards)
	if err != nil {
		return fmt.Sprintf("%s · 未形成合法牌型", game.CardsString(cards))
	}
	if state.CurrentTrick.LastMove != nil && !game.Beats(move, *state.CurrentTrick.LastMove) {
		return fmt.Sprintf("%s · %s · 无法压过当前牌", game.CardsString(cards), handTypeLabel(move.Type))
	}
	return fmt.Sprintf("%s · %s", game.CardsString(cards), handTypeLabel(move.Type))
}

func (m *Model) viewResult() string {
	state := m.engine.State()
	r := state.Result
	winner := "农民胜利"
	if r.WinnerTeam == game.TeamLandlord {
		winner = "地主胜利"
	}
	spring := "无"
	if r.Spring && r.WinnerTeam == game.TeamLandlord {
		spring = "春天"
	} else if r.Spring {
		spring = "反春天"
	}
	var scores strings.Builder
	for seat := 0; seat < 3; seat++ {
		fmt.Fprintf(&scores, "\n%-12s %+d", state.Players[seat].Name, r.Scores[seat])
	}
	return fmt.Sprintf("%s\n\n%s\n\n叫分：%d\n炸弹：%d\n春天：%s\n倍率：x%d\n单份积分：%d\n%s\n\n本次运行：%d 局，胜 %d 局\n\n[Enter] 再来一局  [Esc] 主菜单  [Q] 退出",
		titleStyle.Render("本局结束"), selectStyle.Render(winner), r.BidScore, r.Bombs, spring, r.Multiplier, r.BaseScore, scores.String(), m.stats.Rounds, m.stats.HumanWins)
}

func visibleRange(cursor, total, size int) (int, int) {
	if total <= size {
		return 0, total
	}
	start := cursor - size/2
	if start < 0 {
		start = 0
	}
	if start+size > total {
		start = total - size
	}
	return start, start + size
}

func bidLabel(bid int) string {
	if bid == 0 {
		return "不叫"
	}
	return fmt.Sprintf("%d 分", bid)
}

func onOff(value bool) string {
	if value {
		return "开启"
	}
	return "关闭"
}

func yesNo(value bool) string {
	if value {
		return "是"
	}
	return "否"
}

func winRate(wins, games int) float64 {
	if games == 0 {
		return 0
	}
	return float64(wins) * 100 / float64(games)
}
