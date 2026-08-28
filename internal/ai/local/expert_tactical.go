package local

import (
	"context"
	"math/rand"
	"sort"

	"github.com/syjsion/Terminal_DDZ/internal/game"
)

type expertSimulationSearch struct {
	cache *expertTacticalCache
}

func newExpertSimulationSearch(rootSeat int, rng *rand.Rand) *expertSimulationSearch {
	return &expertSimulationSearch{cache: newExpertTacticalCache(rootSeat, rng.Int63())}
}

// runExpertSimulationSearch adds exact solving for tiny determinizations and a
// shallow adversarial/cooperative reply tree before falling back to rollout.
func runExpertSimulationSearch(ctx context.Context, state rolloutState, rootSeat, maxDepth int, rng *rand.Rand) int {
	return newExpertSimulationSearch(rootSeat, rng).evaluate(ctx, state, maxDepth)
}

func (s *expertSimulationSearch) evaluate(ctx context.Context, state rolloutState, maxDepth int) int {
	if value, done := expertTerminalValue(state, s.cache.rootSeat); done {
		return value
	}
	if expertExactEligible(state) {
		if value, solved := solveExpertExact(ctx, state, s.cache); solved {
			return value
		}
	}
	plies := expertTacticalPlies(state, s.cache.rootSeat)
	if plies == 0 || maxDepth <= 0 {
		if value, ok := s.cache.lookup(state, 0, maxDepth); ok {
			return value
		}
		value := runExpertRollout(ctx, state, s.cache.rootSeat, maxDepth, s.cache.rolloutRNG(state, 0, maxDepth))
		if ctx.Err() == nil {
			s.cache.store(state, 0, maxDepth, value)
		}
		return value
	}
	return runExpertTacticalTreeCached(ctx, state, plies, maxDepth, s.cache)
}

func expertTacticalPlies(state rolloutState, rootSeat int) int {
	if state.winner >= 0 || rootSeat < 0 || rootSeat >= 3 || state.current < 0 || state.current >= 3 {
		return 0
	}
	rootCards := len(state.hands[rootSeat])
	actorCards := len(state.hands[state.current])
	enemyCards := rolloutEnemyMinCards(state, rootSeat)
	totalCards := rolloutTotalCards(state)

	// In very small determinizations, one extra explicit reply is affordable
	// and catches setup moves that a two-ply search can miss. Positions at or
	// below expertExactCardLimit are normally intercepted by the exact solver.
	if totalCards <= 10 || rootCards <= 2 || actorCards <= 2 || enemyCards <= 1 {
		return 3
	}
	if canFinishOnTurn(state, state.current) || rootCards <= 4 || actorCards <= 3 {
		return 2
	}
	if rootCards <= 7 || actorCards <= 5 || enemyCards <= 2 {
		return 1
	}
	if state.target != nil && (state.target.Type == game.Bomb || state.target.Type == game.Rocket) && rootCards <= 10 {
		return 1
	}
	return 0
}

func rolloutTotalCards(state rolloutState) int {
	return len(state.hands[0]) + len(state.hands[1]) + len(state.hands[2])
}

func runExpertTacticalTree(ctx context.Context, state rolloutState, rootSeat, plies, maxDepth int, rng *rand.Rand) int {
	cache := newExpertTacticalCache(rootSeat, rng.Int63())
	return runExpertTacticalTreeCached(ctx, state, plies, maxDepth, cache)
}

func runExpertTacticalTreeCached(ctx context.Context, state rolloutState, plies, maxDepth int, cache *expertTacticalCache) int {
	if err := ctx.Err(); err != nil {
		return evaluateRolloutPosition(state, cache.rootSeat)
	}
	if value, ok := cache.lookup(state, plies, maxDepth); ok {
		return value
	}
	if value, done := expertTerminalValue(state, cache.rootSeat); done {
		cache.store(state, plies, maxDepth, value)
		return value
	}
	if plies <= 0 || maxDepth <= 0 {
		value := runExpertRollout(ctx, state, cache.rootSeat, maxDepth, cache.rolloutRNG(state, plies, maxDepth))
		if ctx.Err() == nil {
			cache.store(state, plies, maxDepth, value)
		}
		return value
	}

	legal := game.GenerateLegalMoves(state.hands[state.current], state.target, state.target != nil)
	if len(legal) == 0 {
		value := evaluateRolloutPosition(state, cache.rootSeat)
		cache.store(state, plies, maxDepth, value)
		return value
	}
	candidates := selectTacticalReplyCandidates(state, legal, tacticalReplyBeam(plies))
	allyTurn := sameRolloutTeam(state.landlord, cache.rootSeat, state.current)

	best := 1 << 30
	if allyTurn {
		best = -1 << 30
	}
	for _, move := range candidates {
		if ctx.Err() != nil {
			return evaluateRolloutPosition(state, cache.rootSeat)
		}
		child := state
		applyRolloutMove(&child, move)

		var value int
		if terminal, done := expertTerminalValue(child, cache.rootSeat); done {
			value = terminal
		} else {
			value = runExpertTacticalTreeCached(ctx, child, plies-1, maxDepth-1, cache)
		}

		if allyTurn {
			if value > best {
				best = value
			}
		} else if value < best {
			best = value
		}
	}
	if ctx.Err() == nil {
		cache.store(state, plies, maxDepth, best)
	}
	return best
}

func tacticalReplyBeam(plies int) int {
	if plies >= 2 {
		return 2
	}
	return 3
}

type tacticalReplyCandidate struct {
	move  game.Move
	score int
}

func selectTacticalReplyCandidates(state rolloutState, legal []game.Move, limit int) []game.Move {
	if limit <= 0 || len(legal) <= limit {
		return append([]game.Move(nil), legal...)
	}
	scored := make([]tacticalReplyCandidate, 0, len(legal))
	for _, move := range legal {
		scored = append(scored, tacticalReplyCandidate{move: move, score: rolloutMoveScore(state, move)})
	}
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		if len(scored[i].move.Cards) != len(scored[j].move.Cards) {
			return len(scored[i].move.Cards) > len(scored[j].move.Cards)
		}
		return scored[i].move.MainRank < scored[j].move.MainRank
	})

	selected := make([]game.Move, 0, limit)
	add := func(move game.Move) {
		if len(selected) >= limit {
			return
		}
		for _, existing := range selected {
			if existing.ID == move.ID {
				return
			}
		}
		selected = append(selected, move)
	}

	// Immediate wins must always be searched. After that, use the rollout
	// policy as the beam prior and reserve a slot for a strategically important
	// pass when a teammate currently controls the trick.
	for _, candidate := range scored {
		if !candidate.move.IsPass && len(candidate.move.Cards) == len(state.hands[state.current]) {
			add(candidate.move)
		}
	}
	for _, candidate := range scored {
		add(candidate.move)
	}
	if state.target != nil && sameRolloutTeam(state.landlord, state.current, state.lead) {
		for _, move := range legal {
			if move.IsPass {
				if !containsRolloutMove(selected, move.ID) {
					selected[len(selected)-1] = move
				}
				break
			}
		}
	}
	return selected
}

func containsRolloutMove(moves []game.Move, id int) bool {
	for _, move := range moves {
		if move.ID == id {
			return true
		}
	}
	return false
}

func expertTerminalValue(state rolloutState, rootSeat int) (int, bool) {
	if state.winner < 0 {
		return 0, false
	}
	if sameRolloutTeam(state.landlord, rootSeat, state.winner) {
		return 10000, true
	}
	return -10000, true
}
