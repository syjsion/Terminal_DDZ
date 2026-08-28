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

func TestExpertExactEligibilityUsesDynamicComplexity(t *testing.T) {
	tiny := rolloutState{current: 0, landlord: 0, winner: -1, hands: [3][]game.Card{
		make([]game.Card, 3), make([]game.Card, 3), make([]game.Card, 4),
	}}
	if !expertExactEligible(tiny) {
		t.Fatal("10-card endgame should always attempt exact solve")
	}

	sparse := rolloutState{
		current:  0,
		landlord: 0,
		lead:     0,
		winner:   -1,
		hands: [3][]game.Card{
			{{Rank: game.Rank3}, {Rank: game.Rank7}, {Rank: game.RankJ}, {Rank: game.Rank2}},
			{{Rank: game.Rank4}, {Rank: game.Rank8}, {Rank: game.RankQ}, {Rank: game.RankA}, {Rank: game.RankSJ}},
			{{Rank: game.Rank5}, {Rank: game.Rank9}, {Rank: game.Rank10}, {Rank: game.RankK}, {Rank: game.RankBJ}},
		},
	}
	complexity, ok := expertExactComplexity(sparse)
	if !ok {
		t.Fatal("sparse 14-card endgame complexity could not be estimated")
	}
	if complexity > expertExactComplexityLimit {
		t.Fatalf("sparse complexity = %d, want <= %d", complexity, expertExactComplexityLimit)
	}
	if !expertExactEligible(sparse) {
		t.Fatal("sparse 14-card endgame should use dynamic exact solve")
	}

	branchy := rolloutState{
		current:  0,
		landlord: 0,
		lead:     0,
		winner:   -1,
		hands: [3][]game.Card{
			{{Rank: game.Rank3}, {Rank: game.Rank4}, {Rank: game.Rank5}, {Rank: game.Rank6}, {Rank: game.Rank7}, {Rank: game.Rank8}, {Rank: game.Rank9}, {Rank: game.Rank10}},
			{{Rank: game.RankJ}, {Rank: game.RankQ}, {Rank: game.RankK}},
			{{Rank: game.RankA}, {Rank: game.Rank2}, {Rank: game.RankSJ}},
		},
	}
	branchComplexity, ok := expertExactComplexity(branchy)
	if !ok {
		t.Fatal("branchy 14-card endgame complexity could not be estimated")
	}
	if branchComplexity <= expertExactComplexityLimit {
		t.Fatalf("branchy complexity = %d, want > %d", branchComplexity, expertExactComplexityLimit)
	}
	if expertExactEligible(branchy) {
		t.Fatal("branchy 14-card endgame should stay on tactical/rollout search")
	}

	tooLarge := sparse
	tooLarge.hands[0] = append(append([]game.Card(nil), sparse.hands[0]...), game.Card{Rank: game.Rank6})
	if expertExactEligible(tooLarge) {
		t.Fatal("15-card endgame must stay outside dynamic exact solve")
	}
}

func TestExpertExactNodeBudgetFallsBackWithoutPoisoningCache(t *testing.T) {
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
	cache := newExpertTacticalCache(0, 11)
	if _, solved := solveExpertExactWithLimit(context.Background(), state, cache, 1); solved {
		t.Fatal("one-node exact budget should force a safe fallback")
	}
	if _, ok := cache.lookupExact(state); ok {
		t.Fatal("incomplete exact search must not cache a root result")
	}

	if _, solved := solveExpertExact(context.Background(), state, cache); !solved {
		t.Fatal("normal exact budget should still solve after a bounded fallback")
	}
}
