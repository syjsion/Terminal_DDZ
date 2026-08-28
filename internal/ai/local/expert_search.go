package local

import (
	"context"
	"math"
	"math/rand"
	"sort"

	"github.com/syjsion/Terminal_DDZ/internal/game"
	"github.com/syjsion/Terminal_DDZ/internal/player"
)

type rootISMCTSStat struct {
	move      game.Move
	baseScore int
	visits    int
	total     int
}

// chooseExpertV2 performs a root information-set Monte Carlo search. Each
// simulation re-determinizes hidden cards from public information, while UCB
// concentrates the rollout budget on moves that are both promising and still
// under-explored. Tactical positions expand the next one or two replies before
// falling back to the rollout policy.
func (a *Agent) chooseExpertV2(ctx context.Context, view player.PlayerView, legal []game.Move) (int, error) {
	unseen := unseenRankCounts(view)
	memo := make(map[string]int, 256)
	stats := make([]rootISMCTSStat, 0, len(legal))
	for _, move := range legal {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		base := hardV2Score(ctx, view, move, unseen, memo)
		if !move.IsPass && len(move.Cards) == len(view.OwnCards) {
			return move.ID, nil
		}
		stats = append(stats, rootISMCTSStat{move: move, baseScore: base})
	}
	sort.SliceStable(stats, func(i, j int) bool { return stats[i].baseScore > stats[j].baseScore })

	limit, simulations, depth := expertV2Budget(view, len(stats))
	if limit > len(stats) {
		limit = len(stats)
	}
	if limit <= 1 || simulations <= 0 {
		return stats[0].move.ID, nil
	}
	stats = selectExpertRootCandidates(stats, limit, view)

	a.mu.Lock()
	seed := a.rng.Int63()
	a.mu.Unlock()
	rng := rand.New(rand.NewSource(seed))

	// Seed every root action against the same determinizations. Paired hidden
	// deals reduce variance between candidate estimates before UCB starts
	// allocating the remaining budget adaptively.
	for n := 0; n < 2; n++ {
		baseState, ok := sampleRolloutState(view, rng)
		if !ok {
			continue
		}
		for i := range stats {
			if err := ctx.Err(); err != nil {
				return 0, err
			}
			state := baseState
			applyRolloutMove(&state, stats[i].move)
			reward := runExpertSimulationSearch(ctx, state, view.Seat, depth, rng)
			stats[i].visits++
			stats[i].total += reward
		}
	}

	totalVisits := 0
	for i := range stats {
		totalVisits += stats[i].visits
	}
	remaining := simulations - totalVisits
	if remaining < 0 {
		remaining = 0
	}
	for step := 0; step < remaining; step++ {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		index := selectRootISMCTSAction(stats, totalVisits)
		reward, ok, err := expertV2Simulation(ctx, view, stats[index].move, depth, rng)
		if err != nil {
			return 0, err
		}
		if !ok {
			continue
		}
		stats[index].visits++
		stats[index].total += reward
		totalVisits++
	}

	best := 0
	bestValue := expertV2FinalValue(stats[0])
	for i := 1; i < len(stats); i++ {
		if value := expertV2FinalValue(stats[i]); value > bestValue {
			best, bestValue = i, value
		}
	}
	return stats[best].move.ID, nil
}

func expertV2Budget(view player.PlayerView, candidateCount int) (limit, simulations, depth int) {
	if candidateCount == 0 {
		return 0, 0, 0
	}
	enemyCards := enemyMinCards(view)
	switch {
	case len(view.OwnCards) <= 7 || enemyCards <= 2:
		return 7, 125, 42
	case len(view.OwnCards) <= 11 || enemyCards <= 5:
		return 6, 92, 34
	default:
		return 5, 60, 25
	}
}

func selectExpertRootCandidates(stats []rootISMCTSStat, limit int, view player.PlayerView) []rootISMCTSStat {
	if limit >= len(stats) {
		return append([]rootISMCTSStat(nil), stats...)
	}
	selected := append([]rootISMCTSStat(nil), stats[:limit]...)

	// Tactical candidates must not disappear merely because a deterministic
	// heuristic ranked them just outside the normal root beam.
	if view.LastMove != nil && sameTeam(view, view.LastMove.Seat) {
		if pass, ok := findRootCandidate(stats, func(move game.Move) bool { return move.IsPass }); ok {
			selected = ensureRootCandidate(selected, pass)
		}
	}
	if enemyMinCards(view) <= 2 {
		if bomb, ok := findRootCandidate(stats, func(move game.Move) bool {
			return move.Type == game.Rocket || move.Type == game.Bomb
		}); ok {
			selected = ensureRootCandidate(selected, bomb)
		}
	}
	return selected
}

func findRootCandidate(stats []rootISMCTSStat, match func(game.Move) bool) (rootISMCTSStat, bool) {
	for _, stat := range stats {
		if match(stat.move) {
			return stat, true
		}
	}
	return rootISMCTSStat{}, false
}

func ensureRootCandidate(selected []rootISMCTSStat, candidate rootISMCTSStat) []rootISMCTSStat {
	for _, stat := range selected {
		if stat.move.ID == candidate.move.ID {
			return selected
		}
	}
	return append(selected, candidate)
}

func selectRootISMCTSAction(stats []rootISMCTSStat, totalVisits int) int {
	best := 0
	bestValue := math.Inf(-1)
	logVisits := math.Log(float64(totalVisits + 2))
	for i, stat := range stats {
		if stat.visits == 0 {
			return i
		}
		mean := float64(stat.total) / float64(stat.visits)
		exploration := 1250.0 * math.Sqrt(logVisits/float64(stat.visits))
		// Hard-v2 acts as a bounded policy prior. Its influence decays as real
		// rollout evidence accumulates and cannot overwhelm repeated outcomes.
		prior := float64(clampExpertPrior(stat.baseScore)) * 0.08 / math.Sqrt(float64(stat.visits+1))
		value := mean + exploration + prior
		if value > bestValue {
			best, bestValue = i, value
		}
	}
	return best
}

func expertV2Simulation(ctx context.Context, view player.PlayerView, move game.Move, depth int, rng *rand.Rand) (int, bool, error) {
	if err := ctx.Err(); err != nil {
		return 0, false, err
	}
	state, ok := sampleRolloutState(view, rng)
	if !ok {
		return 0, false, nil
	}
	applyRolloutMove(&state, move)
	return runExpertSimulationSearch(ctx, state, view.Seat, depth, rng), true, nil
}

func expertV2FinalValue(stat rootISMCTSStat) float64 {
	if stat.visits == 0 {
		return float64(stat.baseScore)
	}
	mean := float64(stat.total) / float64(stat.visits)
	// Rollout evidence dominates; the bounded deterministic prior stabilizes
	// close calls without freezing the search onto its initial ranking.
	return mean*2.45 + float64(clampExpertPrior(stat.baseScore))*0.40
}

func clampExpertPrior(score int) int {
	if score > 8000 {
		return 8000
	}
	if score < -8000 {
		return -8000
	}
	return score
}
