package player

import "github.com/syjsion/Terminal_DDZ/internal/game"

type PublicMove struct {
	Seat int
	Move game.Move
}

type PlayerView struct {
	Phase           game.Phase
	Seat            int
	Role            game.Role
	OwnCards        []game.Card
	OtherCounts     map[int]int
	LandlordSeat    int
	BottomPublic    []game.Card
	PlayedCards     []game.ActionRecord
	LastMove        *PublicMove
	BidScore        int
	Multiplier      int
	CurrentSeat     int
	HighestBid      int
	SuccessfulPlays [3]int
}

func BuildView(state game.GameState, seat int) PlayerView {
	view := PlayerView{
		Phase:           state.Phase,
		Seat:            seat,
		Role:            state.Players[seat].Role,
		OwnCards:        append([]game.Card(nil), state.Players[seat].Hand...),
		OtherCounts:     make(map[int]int, 2),
		LandlordSeat:    state.LandlordSeat,
		PlayedCards:     append([]game.ActionRecord(nil), state.History...),
		BidScore:        state.BidScore,
		Multiplier:      state.Multiplier,
		CurrentSeat:     state.CurrentSeat,
		HighestBid:      state.BidState.HighestBid,
		SuccessfulPlays: state.SuccessfulPlays,
	}
	for i := range state.Players {
		if i != seat {
			view.OtherCounts[i] = len(state.Players[i].Hand)
		}
	}
	if state.LandlordSeat >= 0 {
		view.BottomPublic = append([]game.Card(nil), state.BottomCards...)
	}
	if state.CurrentTrick.LastMove != nil {
		move := *state.CurrentTrick.LastMove
		move.Cards = append([]game.Card(nil), move.Cards...)
		view.LastMove = &PublicMove{Seat: state.CurrentTrick.LeadSeat, Move: move}
	}
	return view
}
