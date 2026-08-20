package game

import "fmt"

func ValidateInvariants(state GameState) error {
	if state.CurrentSeat < 0 || state.CurrentSeat > 2 {
		return fmt.Errorf("CurrentSeat 越界: %d", state.CurrentSeat)
	}
	if state.Phase == PhaseBoot {
		if len(state.BottomCards) != 0 || len(state.History) != 0 {
			return fmt.Errorf("未开始状态仍包含底牌或历史")
		}
		for seat := range state.Players {
			if len(state.Players[seat].Hand) != 0 {
				return fmt.Errorf("未开始状态 Seat %d 仍有手牌", seat)
			}
		}
		return nil
	}
	physical := make([]Card, 0, 54)
	for seat := range state.Players {
		physical = append(physical, state.Players[seat].Hand...)
	}
	if state.Phase == PhaseBidding {
		physical = append(physical, state.BottomCards...)
	} else if state.Phase == PhasePlaying || state.Phase == PhaseFinished {
		if state.LandlordSeat < 0 || state.LandlordSeat > 2 {
			return fmt.Errorf("LandlordSeat 越界: %d", state.LandlordSeat)
		}
		for _, action := range state.History {
			if action.Kind == ActionPlay && !action.Move.IsPass {
				physical = append(physical, action.Move.Cards...)
			}
		}
	}
	if len(physical) != 54 {
		return fmt.Errorf("实体牌总数为 %d，不是 54", len(physical))
	}
	want := make(map[int]int, 54)
	key := func(card Card) int { return int(card.Rank)*10 + int(card.Suit) }
	for _, card := range NewDeck() {
		want[key(card)]++
	}
	for _, card := range physical {
		want[key(card)]--
	}
	for card, count := range want {
		if count != 0 {
			return fmt.Errorf("实体牌 %d 的守恒计数为 %d", card, count)
		}
	}
	if state.Phase == PhasePlaying && !state.Finished {
		target := state.CurrentTrick.LastMove
		if len(GenerateLegalMoves(state.Players[state.CurrentSeat].Hand, target, target != nil)) == 0 {
			return fmt.Errorf("当前玩家 Seat %d 没有合法动作", state.CurrentSeat)
		}
	}
	return nil
}
