package local

import (
	"context"
	"sort"

	"github.com/syjsion/Terminal_DDZ/internal/game"
)

const (
	expertExactGuaranteedCardLimit = 10
	expertExactDynamicCardLimit    = 14
	expertExactComplexityLimit     = 72
)

// expertExactEligible dynamically enables full determinized solving. Tiny
// endgames are always attempted; slightly larger positions are admitted only
// when their observed move branching is small enough. Hidden cards are still
// sampled from PlayerView before this point, so the solver never sees the
// engine's real hidden hands.
func expertExactEligible(state rolloutState) bool {
	if state.winner >= 0 {
		return false
	}
	total := rolloutTotalCards(state)
	if total <= expertExactGuaranteedCardLimit {
		return true
	}
	if total > expertExactDynamicCardLimit {
		return false
	}
	complexity, ok := expertExactComplexity(state)
	return ok && complexity <= expertExactComplexityLimit
}

// expertExactComplexity estimates how expensive a full solve is likely to be.
// The current legal reply count captures immediate tactical branching, while
// each hand's lead-move count approximates how wide future reset tricks can
// become. It intentionally favors sparse, low-combination 11-14 card endings.
func expertExactComplexity(state rolloutState) (int, bool) {
	if state.current < 0 || state.current >= 3 {
		return 0, false
	}
	currentMoves := game.GenerateLegalMoves(state.hands[state.current], state.target, state.target != nil)
	if len(currentMoves) == 0 {
		return 0, false
	}

	score := rolloutTotalCards(state)*2 + len(currentMoves)*5
	for seat := 0; seat < 3; seat++ {
		leadMoves := game.GenerateLegalMoves(state.hands[seat], nil, false)
		if len(leadMoves) > 24 {
			return expertExactComplexityLimit + 1, true
		}
		score += len(leadMoves)
		if score > expertExactComplexityLimit {
			return score, true
		}
	}
	return score, true
}

func expertExactNodeBudget(state rolloutState) int {
	total := rolloutTotalCards(state)
	switch {
	case total <= 8:
		return 30000
	case total <= expertExactGuaranteedCardLimit:
		return 18000
	case total <= 12:
		return 7000
	case total <= expertExactDynamicCardLimit:
		return 3500
	default:
		return 0
	}
}

type expertExactSearch struct {
	cache *expertTacticalCache
	nodes int
	limit int
}

// solveExpertExact determines the game-theoretic team outcome for one sampled
// determinization. The result is exact for that hypothetical deal: +10000 when
// rootSeat's team can force a win, -10000 when the opposing team can force one.
// A false solved result means the context was cancelled or the per-solve node
// budget was exhausted; callers then safely fall back to tactical/rollout play.
func solveExpertExact(ctx context.Context, state rolloutState, cache *expertTacticalCache) (value int, solved bool) {
	return solveExpertExactWithLimit(ctx, state, cache, expertExactNodeBudget(state))
}

func solveExpertExactWithLimit(ctx context.Context, state rolloutState, cache *expertTacticalCache, limit int) (value int, solved bool) {
	if limit <= 0 {
		return 0, false
	}
	search := expertExactSearch{cache: cache, limit: limit}
	return search.solve(ctx, state)
}

func (s *expertExactSearch) solve(ctx context.Context, state rolloutState) (value int, solved bool) {
	if err := ctx.Err(); err != nil {
		return 0, false
	}
	if value, done := expertTerminalValue(state, s.cache.rootSeat); done {
		s.cache.storeExact(state, value)
		return value, true
	}
	if value, ok := s.cache.lookupExact(state); ok {
		return value, true
	}
	if s.nodes >= s.limit {
		return 0, false
	}
	s.nodes++

	legal := game.GenerateLegalMoves(state.hands[state.current], state.target, state.target != nil)
	if len(legal) == 0 {
		return 0, false
	}
	orderExpertExactMoves(state, legal)
	allyTurn := sameRolloutTeam(state.landlord, s.cache.rootSeat, state.current)

	if allyTurn {
		for _, move := range legal {
			if ctx.Err() != nil {
				return 0, false
			}
			child := state
			applyRolloutMove(&child, move)
			childValue, ok := s.solve(ctx, child)
			if !ok {
				return 0, false
			}
			if childValue > 0 {
				s.cache.storeExact(state, 10000)
				return 10000, true
			}
		}
		s.cache.storeExact(state, -10000)
		return -10000, true
	}

	for _, move := range legal {
		if ctx.Err() != nil {
			return 0, false
		}
		child := state
		applyRolloutMove(&child, move)
		childValue, ok := s.solve(ctx, child)
		if !ok {
			return 0, false
		}
		if childValue < 0 {
			s.cache.storeExact(state, -10000)
			return -10000, true
		}
	}
	s.cache.storeExact(state, 10000)
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
