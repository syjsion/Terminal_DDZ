package local

import (
	"context"
	"testing"

	"github.com/syjsion/Terminal_DDZ/internal/game"
)

func TestExpertExactSolverFindsForcedLandlordWin(t *testing.T) {
	state := rolloutState{
		current:  0,
		landlord: 0,
		lead:     0,
		winner:   -1,
		hands: [3][]game.Card{
			{{Rank: game.Rank3}, {Rank: game.Rank10}},
			{{Rank: game.Rank4}},
			{{Rank: game.Rank5}},
		},
	}
	cache := newExpertTacticalCache(0, 7)
	got, solved := solveExpertExact(context.Background(), state, cache)
	if !solved {
		t.Fatal("tiny determinization was not solved exactly")
	}
	if got != 10000 {
		t.Fatalf("forced landlord value = %d, want 10000", got)
	}
}

func TestExpertExactSolverFindsForcedLandlordLoss(t *testing.T) {
	state := rolloutState{
		current:  0,
		landlord: 0,
		lead:     0,
		winner:   -1,
		hands: [3][]game.Card{
			{{Rank: game.Rank3}, {Rank: game.Rank4}},
			{{Rank: game.Rank5}},
			{{Rank: game.Rank6}},
		},
	}
	cache := newExpertTacticalCache(0, 8)
	got, solved := solveExpertExact(context.Background(), state, cache)
	if !solved {
		t.Fatal("tiny determinization was not solved exactly")
	}
	if got != -10000 {
		t.Fatalf("forced landlord value = %d, want -10000", got)
	}
}

func TestExpertExactSolverUsesFarmerCooperation(t *testing.T) {
	state := rolloutState{
		current:  1,
		landlord: 0,
		lead:     1,
		winner:   -1,
		hands: [3][]game.Card{
			{{Rank: game.Rank10}},
			{{Rank: game.Rank3}, {Rank: game.Rank9}},
			{{Rank: game.Rank4}},
		},
	}
	cache := newExpertTacticalCache(1, 9)
	got, solved := solveExpertExact(context.Background(), state, cache)
	if !solved {
		t.Fatal("tiny farmer determinization was not solved exactly")
	}
	if got != 10000 {
		t.Fatalf("farmer team value = %d, want 10000", got)
	}
}

func TestExpertExactSolverReusesTranspositionCache(t *testing.T) {
	state := rolloutState{
		current:  0,
		landlord: 0,
		lead:     0,
		winner:   -1,
		hands: [3][]game.Card{
			{{Rank: game.Rank3}, {Rank: game.Rank10}},
			{{Rank: game.Rank4}},
			{{Rank: game.Rank5}},
		},
	}
	cache := newExpertTacticalCache(0, 10)
	first, solved := solveExpertExact(context.Background(), state, cache)
	if !solved {
		t.Fatal("first exact solve failed")
	}
	hitsBefore := cache.hits
	second, solved := solveExpertExact(context.Background(), state, cache)
	if !solved {
		t.Fatal("second exact solve failed")
	}
	if first != second {
		t.Fatalf("cached exact value changed from %d to %d", first, second)
	}
	if cache.hits <= hitsBefore {
		t.Fatal("second exact solve did not reuse transposition cache")
	}
}

func TestExpertExactEligibilityOnlyUsesTinyEndgames(t *testing.T) {
	tiny := rolloutState{winner: -1, hands: [3][]game.Card{
		make([]game.Card, 3), make([]game.Card, 3), make([]game.Card, 4),
	}}
	if !expertExactEligible(tiny) {
		t.Fatal("10-card endgame should use exact solver")
	}
	larger := tiny
	larger.hands[2] = make([]game.Card, 5)
	if expertExactEligible(larger) {
		t.Fatal("11-card endgame should stay on tactical/rollout search")
	}
}
