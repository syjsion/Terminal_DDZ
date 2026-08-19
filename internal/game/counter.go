package game

type CounterView struct {
	UnknownByRank  map[Rank]int
	OtherCounts    map[int]int
	BombsPlayed    int
	RocketPossible bool
}

func BuildCounter(state GameState, seat int) CounterView {
	counts := RankCounts(NewDeck())
	for _, card := range state.Players[seat].Hand {
		counts[card.Rank]--
	}
	for _, action := range state.History {
		if action.Kind != ActionPlay || action.Move.IsPass {
			continue
		}
		for _, card := range action.Move.Cards {
			counts[card.Rank]--
		}
	}
	other := make(map[int]int, 2)
	for i := range state.Players {
		if i != seat {
			other[i] = len(state.Players[i].Hand)
		}
	}
	return CounterView{
		UnknownByRank:  counts,
		OtherCounts:    other,
		BombsPlayed:    state.Bombs,
		RocketPossible: counts[RankSJ] > 0 && counts[RankBJ] > 0,
	}
}
