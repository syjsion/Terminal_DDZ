package tui

import (
	"testing"

	"github.com/syjsion/Terminal_DDZ/internal/game"
)

func orderTestCards(ranks ...game.Rank) []game.Card {
	seen := map[game.Rank]int{}
	cards := make([]game.Card, 0, len(ranks))
	for _, rank := range ranks {
		cards = append(cards, game.Card{Rank: rank, Suit: game.Suit(seen[rank])})
		seen[rank]++
	}
	return cards
}

func TestLeadOrderingPromotesTripleWithSingle(t *testing.T) {
	hand := orderTestCards(game.Rank3, game.Rank3, game.Rank3, game.Rank4, game.Rank6, game.Rank8, game.Rank10, game.RankQ, game.RankA, game.Rank2)
	ordered := rankHumanMoves(hand, game.GenerateLegalMoves(hand, nil, false), nil)
	if len(ordered) == 0 || ordered[0].Type != game.TripleWithSingle {
		t.Fatalf("first recommendation = %v %q", ordered[0].Type, ordered[0].String())
	}
	indexTriple, indexSingle := -1, -1
	for i, move := range ordered {
		if indexTriple < 0 && move.Type == game.Triple {
			indexTriple = i
		}
		if indexSingle < 0 && move.Type == game.Single {
			indexSingle = i
		}
	}
	if indexTriple < 0 || indexSingle < 0 || indexTriple <= 0 || indexSingle <= indexTriple {
		t.Fatalf("unexpected lead order: triple=%d single=%d", indexTriple, indexSingle)
	}
}

func TestFollowingOrderingUsesMinimumBeforePassAndBombs(t *testing.T) {
	hand := orderTestCards(game.Rank10, game.Rank10, game.RankJ, game.RankJ, game.Rank3, game.Rank3, game.Rank3, game.Rank3, game.RankSJ, game.RankBJ)
	targetCards := orderTestCards(game.Rank9, game.Rank9)
	target, err := game.AnalyzeMove(targetCards)
	if err != nil {
		t.Fatal(err)
	}
	ordered := rankHumanMoves(hand, game.GenerateLegalMoves(hand, &target, true), &target)
	if ordered[0].Type != game.Pair || ordered[0].MainRank != game.Rank10 {
		t.Fatalf("first response = %v %q", ordered[0].Type, ordered[0].String())
	}
	pass, bomb := -1, -1
	for i, move := range ordered {
		if move.IsPass {
			pass = i
		}
		if move.Type == game.Bomb {
			bomb = i
		}
	}
	if pass <= 0 || bomb <= pass {
		t.Fatalf("unexpected follow order: pass=%d bomb=%d", pass, bomb)
	}
}
