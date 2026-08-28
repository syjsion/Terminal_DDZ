package local

import (
	"context"
	"sort"

	"github.com/syjsion/Terminal_DDZ/internal/game"
)

const expertExactCardLimit = 10

// expertExactEligible restricts full determinized solving to very small
// endgames. Hidden cards are still sampled from PlayerView before this point;
// the solver never sees the engine's real hidden hands.
func expertExactEligible(state rolloutState) bool {
	return state.winner < 0 && rolloutTotalCards(state) <= expertExactCardLimit
}

// solveExpertExact determines the game-theoretic team outcome for one sampled
// determinization. The result is exact for that hypothetical deal: +10000 when
// rootSeat's team can force a win, -10000 when the opposing team can force one.
// A false solved result means the context was cancelled before completion.
func solveExpertExact(ctx context.Context, state rolloutState, cache *expertTacticalCache) (value int, solved bool) {
	if err := ctx.Err(); err != nil {
		return 0, false
	}
	if value, done := expertTerminalValue(state, cache.rootSeat); done {
		cache.storeExact(state, value)
		return value, true
	}
	if value, ok := cache.lookupExact(state); ok {
		return value, true
	}

	legal := game.GenerateLegalMoves(state.hands[state.current], state.target, state.target != nil)
	if len(legal) == 0 {
		return 0, false
	}
	orderExpertExactMoves(state, legal)
	allyTurn := sameRolloutTeam(state.landlord, cache.rootSeat, state.current)

	if allyTurn {
		for _, move := range legal {
			if ctx.Err() != nil {
				return 0, false
			}
			child := state
			applyRolloutMove(&child, move)
			childValue, ok := solveExpertExact(ctx, child, cache)
			if !ok {
				return 0, false
			}
			if childValue > 0 {
				cache.storeExact(state, 10000)
				return 10000, true
			}
		}
		cache.storeExact(state, -10000)
		return -10000, true
	}

	for _, move := range legal {
		if ctx.Err() != nil {
			return 0, false
		}
		child := state
		applyRolloutMove(&child, move)
		childValue, ok := solveExpertExact(ctx, child, cache)
		if !ok {
			return 0, false
		}
		if childValue < 0 {
			cache.storeExact(state, -10000)
			return -10000, true
		}
	}
	cache.storeExact(state, 10000)
	return 10000, true
}

func orderExpertExactMoves(state rolloutState, moves []game.Move) {
	sort.SliceStable(moves, func(i, j int) bool {
		si := rolloutMoveScore(state, moves[i])
		sj := rolloutMoveScore(state, moves[j])
		if si != sj {
			return si > sj
		}
		if len(moves[i].Cards) != len(moves[j].Cards) {
			return len(moves[i].Cards) > len(moves[j].Cards)
		}
		return moves[i].MainRank < moves[j].MainRank
	})
}

func (c *expertTacticalCache) lookupExact(state rolloutState) (int, bool) {
	value, ok := c.values[expertExactStateKey(state, c.rootSeat)]
	if ok {
		c.hits++
	}
	return value, ok
}

func (c *expertTacticalCache) storeExact(state rolloutState, value int) {
	c.values[expertExactStateKey(state, c.rootSeat)] = value
}

func expertExactStateKey(state rolloutState, rootSeat int) string {
	// 0xff is outside every real tactical horizon used by Expert, keeping exact
	// terminal values separate from depth-limited tactical/rollout estimates.
	return expertTacticalStateKey(state, rootSeat, 0xff, 0xff)
}
