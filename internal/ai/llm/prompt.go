package llm

import (
	"fmt"
	"strings"

	"github.com/syjsion/Terminal_DDZ/internal/game"
	"github.com/syjsion/Terminal_DDZ/internal/player"
)

func bidPrompt(view player.PlayerView, legal []int) string {
	return fmt.Sprintf(`You are playing Dou Dizhu (斗地主).
You are Seat %d.
Card rank: 3 < 4 < 5 < 6 < 7 < 8 < 9 < 10 < J < Q < K < A < 2 < SJ < BJ
Your cards: %s
Current highest bid: %d
Legal bids: %s
Evaluate rocket, bombs, jokers, 2s, triples, sequence potential, and weak isolated cards.
Bid aggressively only when the whole hand has enough control and a realistic route to finish; do not overbid a fragmented hand merely because it has a few high cards.
Choose the strategically best legal bid. Return JSON only: {"bid": <id>}`,
		view.Seat, game.CardsString(view.OwnCards), view.HighestBid, intsString(legal))
}

func movePrompt(view player.PlayerView, legal []game.Move) string {
	var b strings.Builder
	fmt.Fprintf(&b, "You are playing Dou Dizhu (斗地主).\nYou are Seat %d.\nRole: %s.\nLandlord: Seat %d.\nMultiplier: x%d.\n", view.Seat, roleEnglish(view.Role), view.LandlordSeat, view.Multiplier)
	b.WriteString("Card rank: 3 < 4 < 5 < 6 < 7 < 8 < 9 < 10 < J < Q < K < A < 2 < SJ < BJ\n")
	fmt.Fprintf(&b, "Your cards: %s\nCards remaining:", game.CardsString(view.OwnCards))
	for seat := 0; seat < 3; seat++ {
		if seat != view.Seat {
			fmt.Fprintf(&b, " Seat %d (%s)=%d", seat, seatRelationship(view, seat), view.OtherCounts[seat])
		}
	}
	fmt.Fprintf(&b, "\nPublic bottom cards: %s\n", game.CardsString(view.BottomPublic))
	writeTarget(&b, view)
	writeThreats(&b, view)
	unplayed := unplayedOutsideHand(view)
	b.WriteString("Unplayed cards outside your hand by rank:")
	for _, rank := range game.AllRanks {
		fmt.Fprintf(&b, " %s=%d", rank, unplayed[rank])
	}
	b.WriteByte('\n')
	b.WriteString("Recent public actions:\n")
	history := view.PlayedCards
	if len(history) > 12 {
		history = history[len(history)-12:]
	}
	for _, action := range history {
		if action.Kind == game.ActionBid {
			fmt.Fprintf(&b, "Seat %d: BID %d\n", action.Seat, action.Bid)
		} else {
			fmt.Fprintf(&b, "Seat %d: %s\n", action.Seat, action.Move.String())
		}
	}
	b.WriteString("Played card totals:")
	played := map[game.Rank]int{}
	for _, action := range view.PlayedCards {
		if action.Kind == game.ActionPlay && !action.Move.IsPass {
			for _, card := range action.Move.Cards {
				played[card.Rank]++
			}
		}
	}
	for _, rank := range game.AllRanks {
		if played[rank] > 0 {
			fmt.Fprintf(&b, " %s=%d", rank, played[rank])
		}
	}
	b.WriteString("\nLegal moves:\n")
	for _, move := range legal {
		moveType := move.Type.String()
		if move.IsPass {
			moveType = "PASS"
		}
		used := len(move.Cards)
		remaining := len(view.OwnCards) - used
		power := "normal"
		if move.Type == game.Bomb || move.Type == game.Rocket {
			power = "explosive"
		}
		fmt.Fprintf(&b, "%d = cards=%s; type=%s; cards_used=%d; cards_remaining=%d; finishes_hand=%s; power=%s\n",
			move.ID, move.Label(), moveType, used, remaining, yesNo(remaining == 0 && !move.IsPass), power)
	}
	b.WriteString(`Decision priorities:
1. Take an immediate win; otherwise stop an opponent with 1-2 cards when possible.
2. Prefer efficient combinations that reduce future turns and avoid creating isolated cards.
3. When following, use the cheapest sufficient control unless spending a premium card creates a decisive advantage.
4. Preserve bombs/rocket and high control cards unless they win, answer an urgent threat, or establish a clear finishing route.
5. As a farmer, usually PASS when your teammate controls the trick; overtake only to win, stop the landlord, or materially improve the team's route.
Choose the strategically best legal move. Return JSON only: {"move": <id>}`)
	return b.String()
}

func writeTarget(b *strings.Builder, view player.PlayerView) {
	if view.LastMove == nil {
		b.WriteString("Current target: none; you lead a new trick.\n")
		return
	}
	moveType := view.LastMove.Move.Type.String()
	if view.LastMove.Move.IsPass {
		moveType = "PASS"
	}
	fmt.Fprintf(b, "Current target: Seat %d (%s) played %s, type=%s.\n",
		view.LastMove.Seat, seatRelationship(view, view.LastMove.Seat), view.LastMove.Move.Label(), moveType)
}

func writeThreats(b *strings.Builder, view player.PlayerView) {
	b.WriteString("Urgent low-card threats:")
	found := false
	for seat := 0; seat < 3; seat++ {
		if seat == view.Seat || view.OtherCounts[seat] > 2 {
			continue
		}
		found = true
		fmt.Fprintf(b, " Seat %d (%s)=%d", seat, seatRelationship(view, seat), view.OtherCounts[seat])
	}
	if !found {
		b.WriteString(" none")
	}
	b.WriteByte('\n')
}

func unplayedOutsideHand(view player.PlayerView) map[game.Rank]int {
	counts := game.RankCounts(game.NewDeck())
	for _, card := range view.OwnCards {
		counts[card.Rank]--
	}
	for _, action := range view.PlayedCards {
		if action.Kind != game.ActionPlay || action.Move.IsPass {
			continue
		}
		for _, card := range action.Move.Cards {
			counts[card.Rank]--
		}
	}
	return counts
}

func seatRelationship(view player.PlayerView, seat int) string {
	if seat == view.Seat {
		return "self"
	}
	if view.Role == game.RoleFarmer && seat != view.LandlordSeat {
		return "teammate"
	}
	if seat == view.LandlordSeat {
		return "opponent/landlord"
	}
	return "opponent/farmer"
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func roleEnglish(role game.Role) string {
	if role == game.RoleLandlord {
		return "Landlord"
	}
	return "Farmer"
}

func intsString(values []int) string {
	parts := make([]string, len(values))
	for i, value := range values {
		parts[i] = fmt.Sprint(value)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}
