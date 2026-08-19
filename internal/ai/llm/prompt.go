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
Choose the strategically best bid. Return JSON only: {"bid": <id>}`,
		view.Seat, game.CardsString(view.OwnCards), view.HighestBid, intsString(legal))
}

func movePrompt(view player.PlayerView, legal []game.Move) string {
	var b strings.Builder
	fmt.Fprintf(&b, "You are playing Dou Dizhu (斗地主).\nYou are Seat %d.\nRole: %s.\nLandlord: Seat %d.\n", view.Seat, roleEnglish(view.Role), view.LandlordSeat)
	b.WriteString("Card rank: 3 < 4 < 5 < 6 < 7 < 8 < 9 < 10 < J < Q < K < A < 2 < SJ < BJ\n")
	fmt.Fprintf(&b, "Your cards: %s\nCards remaining:", game.CardsString(view.OwnCards))
	for seat := 0; seat < 3; seat++ {
		if seat != view.Seat {
			fmt.Fprintf(&b, " Seat %d=%d", seat, view.OtherCounts[seat])
		}
	}
	fmt.Fprintf(&b, "\nPublic bottom cards: %s\n", game.CardsString(view.BottomPublic))
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
		fmt.Fprintf(&b, "%d = %s\n", move.ID, move.Label())
	}
	b.WriteString("Choose the strategically best move. Farmers should cooperate when appropriate.\nReturn JSON only: {\"move\": <id>}")
	return b.String()
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
