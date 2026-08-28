package local

import (
	"context"
	"math/rand"
	"testing"

	"github.com/syjsion/Terminal_DDZ/internal/game"
)

func TestExpertTacticalTreeAssumesEnemyBestImmediateWin(t *testing.T) {
	target := mustTacticalMove(t, game.Card{Rank: game.Rank4})
	state := rolloutState{
		current:  1,
		landlord: 0,
		lead:     0,
		target:   &target,
		winner:   -1,
		hands: [3][]game.Card{
			{{Rank: game.Rank3}, {Rank: game.Rank6}},
			{{Rank: game.Rank5}},
			{{Rank: game.Rank7}, {Rank: game.Rank8}},
		},
	}
	got := runExpertTacticalTree(context.Background(), state, 0, 1, 8, rand.New(rand.NewSource(1)))
	if got != -10000 {
		t.Fatalf("enemy immediate winning reply = %d, want -10000", got)
	}
}

func TestExpertTacticalTreeUsesTeammateWinningReply(t *testing.T) {
	target := mustTacticalMove(t, game.Card{Rank: game.Rank4})
	state := rolloutState{
		current:  2,
		landlord: 0,
		lead:     0,
		target:   &target,
		winner:   -1,
		hands: [3][]game.Card{
			{{Rank: game.Rank9}, {Rank: game.Rank10}},
			{{Rank: game.Rank3}, {Rank: game.Rank7}},
			{{Rank: game.Rank5}},
		},
	}
	got := runExpertTacticalTree(context.Background(), state, 1, 1, 8, rand.New(rand.NewSource(2)))
	if got != 10000 {
		t.Fatalf("teammate immediate winning reply = %d, want 10000", got)
	}
}

func TestExpertTacticalPliesFocusesOnCriticalEndgame(t *testing.T) {
	critical := rolloutState{
		current:  1,
		landlord: 0,
		winner:   -1,
		hands: [3][]game.Card{
			make([]game.Card, 4),
			make([]game.Card, 8),
			make([]game.Card, 9),
		},
	}
	if got := expertTacticalPlies(critical, 0); got != 2 {
		t.Fatalf("critical tactical plies = %d, want 2", got)
	}

	tiny := rolloutState{
		current:  1,
		landlord: 0,
		winner:   -1,
		hands: [3][]game.Card{
			make([]game.Card, 3),
			make([]game.Card, 3),
			make([]game.Card, 4),
		},
	}
	if got := expertTacticalPlies(tiny, 0); got != 3 {
		t.Fatalf("tiny endgame tactical plies = %d, want 3", got)
	}

	quiet := rolloutState{
		current:  1,
		landlord: 0,
		winner:   -1,
		hands: [3][]game.Card{
			make([]game.Card, 12),
			make([]game.Card, 10),
			make([]game.Card, 11),
		},
	}
	if got := expertTacticalPlies(quiet, 0); got != 0 {
		t.Fatalf("quiet tactical plies = %d, want 0", got)
	}
}

func TestTacticalReplyCandidatesKeepPassOverTeammate(t *testing.T) {
	target := mustTacticalMove(t, game.Card{Rank: game.Rank4})
	state := rolloutState{
		current:  2,
		landlord: 0,
		lead:     1,
		target:   &target,
		winner:   -1,
		hands: [3][]game.Card{
			{{Rank: game.Rank3}, {Rank: game.Rank9}},
			{{Rank: game.Rank10}, {Rank: game.RankJ}},
			{{Rank: game.Rank5}, {Rank: game.Rank6}, {Rank: game.Rank7}, {Rank: game.Rank8}},
		},
	}
	legal := game.GenerateLegalMoves(state.hands[state.current], state.target, true)
	selected := selectTacticalReplyCandidates(state, legal, 3)
	for _, move := range selected {
		if move.IsPass {
			return
		}
	}
	t.Fatal("tactical reply beam dropped pass while teammate controlled trick")
}

func TestExpertTacticalStateKeyIgnoresSuitsButTracksSearchState(t *testing.T) {
	target := mustTacticalMove(t, game.Card{Rank: game.Rank7})
	stateA := rolloutState{
		current:  1,
		landlord: 0,
		lead:     0,
		target:   &target,
		winner:   -1,
		hands: [3][]game.Card{
			{{Rank: game.Rank3, Suit: game.Club}, {Rank: game.Rank3, Suit: game.Diamond}},
			{{Rank: game.Rank8, Suit: game.Heart}},
			{{Rank: game.Rank9, Suit: game.Spade}},
		},
	}
	stateB := stateA
	stateB.hands[0] = []game.Card{{Rank: game.Rank3, Suit: game.Heart}, {Rank: game.Rank3, Suit: game.Spade}}
	stateB.hands[1] = []game.Card{{Rank: game.Rank8, Suit: game.Club}}
	stateB.hands[2] = []game.Card{{Rank: game.Rank9, Suit: game.Diamond}}

	keyA := expertTacticalStateKey(stateA, 0, 2, 10)
	keyB := expertTacticalStateKey(stateB, 0, 2, 10)
	if keyA != keyB {
		t.Fatal("rank-equivalent states should share a tactical cache key")
	}
	if keyA == expertTacticalStateKey(stateA, 0, 1, 10) {
		t.Fatal("different tactical horizons must not share a cache key")
	}
	stateB.current = 2
	if keyA == expertTacticalStateKey(stateB, 0, 2, 10) {
		t.Fatal("different side-to-move must not share a cache key")
	}
}

func TestExpertTacticalCacheReusesRepeatedState(t *testing.T) {
	state := rolloutState{
		current:  0,
		landlord: 0,
		winner:   -1,
		hands: [3][]game.Card{
			{{Rank: game.Rank3}, {Rank: game.Rank4}, {Rank: game.Rank5}},
			{{Rank: game.Rank6}, {Rank: game.Rank7}, {Rank: game.Rank8}},
			{{Rank: game.Rank9}, {Rank: game.Rank10}, {Rank: game.RankJ}},
		},
	}
	cache := newExpertTacticalCache(0, 42)
	first := runExpertTacticalTreeCached(context.Background(), state, 2, 8, cache)
	hitsBefore := cache.hits
	second := runExpertTacticalTreeCached(context.Background(), state, 2, 8, cache)
	if first != second {
		t.Fatalf("cached tactical value changed from %d to %d", first, second)
	}
	if cache.hits <= hitsBefore {
		t.Fatal("second tactical evaluation did not hit transposition cache")
	}
}

func mustTacticalMove(t *testing.T, cards ...game.Card) game.Move {
	t.Helper()
	move, err := game.AnalyzeMove(cards)
	if err != nil {
		t.Fatal(err)
	}
	return move
}
