package local

import (
	"context"
	"math/rand"
	"sort"

	"github.com/syjsion/Terminal_DDZ/internal/game"
)

// runExpertSimulationSearch adds a shallow adversarial/cooperative reply tree
// before falling back to the rollout policy. The tree is only enabled in
// tactical positions so Expert gains lookahead without making every move slow.
func runExpertSimulationSearch(ctx context.Context, state rolloutState, rootSeat, maxDepth int, rng *rand.Rand) int {
	if value, done := expertTerminalValue(state, rootSeat); done {
		return value
	}
	plies := expertTacticalPlies(state, rootSeat)
	if plies == 0 || maxDepth <= 0 {
		return runExpertRollout(ctx, state, rootSeat, maxDepth, rng)
	}
	return runExpertTacticalTree(ctx, state, rootSeat, plies, maxDepth, rng)
}

func expertTacticalPlies(state rolloutState, rootSeat int) int {
	if state.winner >= 0 || rootSeat < 0 || rootSeat >= 3 || state.current < 0 || state.current >= 3 {
		return 0
	}
	rootCards := len(state.hands[rootSeat])
	actorCards := len(state.hands[state.current])
	enemyCards := rolloutEnemyMinCards(state, rootSeat)

	if canFinishOnTurn(state, state.current) || rootCards <= 4 || actorCards <= 3 || enemyCards <= 1 {
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

func runExpertTacticalTree(ctx context.Context, state rolloutState, rootSeat, plies, maxDepth int, rng *rand.Rand) int {
	if err := ctx.Err(); err != nil {
		return evaluateRolloutPosition(state, rootSeat)
	}
	if value, done := expertTerminalValue(state, rootSeat); done {
		return value
	}
	if plies <= 0 || maxDepth <= 0 {
		return runExpertRollout(ctx, state, rootSeat, maxDepth, rng)
	}

	legal := game.GenerateLegalMoves(state.hands[state.current], state.target, state.target != nil)
	if len(legal) == 0 {
		return evaluateRolloutPosition(state, rootSeat)
	}
	candidates := selectTacticalReplyCandidates(state, legal, tacticalReplyBeam(plies))
	allyTurn := sameRolloutTeam(state.landlord, rootSeat, state.current)

	best := 1 << 30
	if allyTurn {
		best = -1 << 30
	}
	branchSeed := rng.Int63()
	for _, move := range candidates {
		child := state
		applyRolloutMove(&child, move)
		branchRNG := rand.New(rand.NewSource(branchSeed))

		var value int
		if terminal, done := expertTerminalValue(child, rootSeat); done {
			value = terminal
		} else if plies > 1 {
			value = runExpertTacticalTree(ctx, child, rootSeat, plies-1, maxDepth-1, branchRNG)
		} else {
			value = runExpertRollout(ctx, child, rootSeat, maxDepth-1, branchRNG)
		}

		if allyTurn {
			if value > best {
				best = value
			}
		} else if value < best {
			best = value
		}
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
