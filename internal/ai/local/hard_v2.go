package local

import (
	"context"
	"sort"

	"github.com/syjsion/Terminal_DDZ/internal/game"
	"github.com/syjsion/Terminal_DDZ/internal/player"
)

// chooseHardV2 combines hand-structure search, card counting, opponent danger
// awareness and farmer teamwork. It intentionally uses only PlayerView data,
// so the AI never sees hidden opponent cards.
func chooseHardV2(ctx context.Context, view player.PlayerView, legal []game.Move) (int, error) {
	unseen := unseenRankCounts(view)
	memo := make(map[string]int, 256)
	bestID, bestScore := legal[0].ID, -1<<30
	for _, move := range legal {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		score := hardV2Score(ctx, view, move, unseen, memo)
		if score > bestScore {
			bestID, bestScore = move.ID, score
		}
	}
	return bestID, nil
}

func hardV2Score(ctx context.Context, view player.PlayerView, move game.Move, unseen map[game.Rank]int, memo map[string]int) int {
	score := normalScore(view, move)
	enemyCards := enemyMinCards(view)
	if move.IsPass {
		return score + hardPassScoreV2(view, enemyCards)
	}

	remaining := removeByRank(view.OwnCards, move.Cards)
	turns := remainingTurnEstimate(ctx, remaining, memo)
	score += remainderTurnScore(turns, len(remaining))
	score += controlScoreV2(move, unseen)
	score += dangerScoreV2(view, move, unseen, enemyCards)
	score += teamworkScoreV2(view, move, enemyCards)
	score += bombUsageScoreV2(view, move, enemyCards, turns)
	return score
}

func hardPassScoreV2(view player.PlayerView, enemyCards int) int {
	if view.LastMove == nil {
		return -100000
	}
	score := 0
	if sameTeam(view, view.LastMove.Seat) {
		score += 1800
		if count, ok := view.OtherCounts[view.LastMove.Seat]; ok && count <= 2 {
			score += 1600
		}
		if enemyCards <= 2 {
			score -= 1200
		}
	} else if enemyCards <= 2 {
		score -= 2200
	}
	return score
}

func teamworkScoreV2(view player.PlayerView, move game.Move, enemyCards int) int {
	if view.LastMove == nil || !sameTeam(view, view.LastMove.Seat) {
		return 0
	}
	penalty := -1700
	if enemyCards <= 2 {
		penalty = -350
	}
	if count, ok := view.OtherCounts[view.LastMove.Seat]; ok && count <= 2 {
		penalty -= 1200
	}
	if move.Type == game.Bomb || move.Type == game.Rocket {
		penalty -= 1600
	}
	return penalty
}

func dangerScoreV2(view player.PlayerView, move game.Move, unseen map[game.Rank]int, enemyCards int) int {
	if enemyCards > 2 {
		return 0
	}
	score := 0
	if view.LastMove != nil && !sameTeam(view, view.LastMove.Seat) {
		score += 700
	}
	switch enemyCards {
	case 1:
		if move.Type == game.Single {
			if higherUnseenRanks(move.MainRank, 1, unseen) == 0 {
				score += 350
			} else {
				score -= 1400
			}
		} else if len(move.Cards) >= 2 {
			score += 650
		}
	case 2:
		if move.Type == game.Pair {
			if higherUnseenRanks(move.MainRank, 2, unseen) == 0 {
				score += 200
			} else {
				score -= 500
			}
		} else if len(move.Cards) >= 3 {
			score += 300
		}
	}
	return score
}

func controlScoreV2(move game.Move, unseen map[game.Rank]int) int {
	need := 0
	switch move.Type {
	case game.Single:
		need = 1
	case game.Pair:
		need = 2
	case game.Triple, game.TripleWithSingle, game.TripleWithPair:
		need = 3
	case game.Bomb:
		need = 4
	default:
		return 0
	}
	higher := higherUnseenRanks(move.MainRank, need, unseen)
	switch higher {
	case 0:
		return 650
	case 1:
		return 220
	default:
		if higher > 6 {
			higher = 6
		}
		return -higher * 35
	}
}

func bombUsageScoreV2(view player.PlayerView, move game.Move, enemyCards, turns int) int {
	if move.Type != game.Bomb && move.Type != game.Rocket {
		return 0
	}
	score := 0
	if enemyCards <= 2 {
		if move.Type == game.Bomb {
			score += 750
		} else {
			score += 900
		}
	}
	if turns <= 2 {
		if move.Type == game.Bomb {
			score += 1300
		} else {
			score += 1500
		}
	}
	if turns == 1 {
		score += 1200
	}
	if view.LastMove != nil && sameTeam(view, view.LastMove.Seat) {
		score -= 1200
	}
	return score
}

func enemyMinCards(view player.PlayerView) int {
	best := 1 << 30
	for seat, count := range view.OtherCounts {
		if !sameTeam(view, seat) && count < best {
			best = count
		}
	}
	return best
}

// unseenRankCounts reconstructs the cards that may still be in opponents'
// hands from the public history and our own hand. This is card counting, not
// hidden-information access.
func unseenRankCounts(view player.PlayerView) map[game.Rank]int {
	counts := make(map[game.Rank]int, len(game.AllRanks))
	for _, rank := range game.AllRanks {
		counts[rank] = 4
	}
	counts[game.RankSJ], counts[game.RankBJ] = 1, 1
	for _, card := range view.OwnCards {
		counts[card.Rank]--
	}
	for _, action := range view.PlayedCards {
		if action.Kind != game.ActionPlay || action.Move.IsPass {
			continue
		}
		for _, card := range action.Move.Cards {
			counts[card.Rank]--
		}
	}
	for rank, count := range counts {
		if count < 0 {
			counts[rank] = 0
		}
	}
	return counts
}

func higherUnseenRanks(rank game.Rank, need int, unseen map[game.Rank]int) int {
	count := 0
	for _, other := range game.AllRanks {
		if other > rank && unseen[other] >= need {
			count++
		}
	}
	return count
}

func remainderTurnScore(turns, remaining int) int {
	if remaining == 0 {
		return 100000
	}
	if turns <= 0 {
		return -10000
	}
	score := -turns * 260
	if remaining <= 10 {
		score -= turns * 220
	}
	switch turns {
	case 1:
		score += 6500
	case 2:
		score += 2600
	case 3:
		score += 700
	}
	return score
}

// For ten cards or fewer, search every legal decomposition and memoize the
// minimum number of turns required to empty the hand. Larger hands use a fast
// structural estimate to keep terminal play responsive.
func remainingTurnEstimate(ctx context.Context, hand []game.Card, memo map[string]int) int {
	if len(hand) == 0 {
		return 0
	}
	if len(hand) <= 10 {
		return minTurnsToFinish(ctx, hand, memo)
	}
	return greedyGroupEstimateV2(hand)
}

func minTurnsToFinish(ctx context.Context, hand []game.Card, memo map[string]int) int {
	if len(hand) == 0 {
		return 0
	}
	key := handKey(hand)
	if value, ok := memo[key]; ok {
		return value
	}
	if ctx.Err() != nil {
		return greedyGroupEstimateV2(hand)
	}
	moves := game.GenerateLegalMoves(hand, nil, false)
	if len(moves) == 0 {
		return len(hand)
	}
	sort.SliceStable(moves, func(i, j int) bool {
		if len(moves[i].Cards) != len(moves[j].Cards) {
			return len(moves[i].Cards) > len(moves[j].Cards)
		}
		ib := moves[i].Type == game.Bomb || moves[i].Type == game.Rocket
		jb := moves[j].Type == game.Bomb || moves[j].Type == game.Rocket
		if ib != jb {
			return !ib
		}
		return moves[i].MainRank < moves[j].MainRank
	})
	best := len(hand)
	for _, move := range moves {
		if ctx.Err() != nil {
			return greedyGroupEstimateV2(hand)
		}
		turns := 1 + minTurnsToFinish(ctx, removeByRank(hand, move.Cards), memo)
		if turns < best {
			best = turns
			if best == 1 {
				break
			}
		}
	}
	memo[key] = best
	return best
}

func handKey(hand []game.Card) string {
	counts := game.RankCounts(hand)
	key := make([]byte, len(game.AllRanks))
	for i, rank := range game.AllRanks {
		key[i] = byte(counts[rank])
	}
	return string(key)
}

func greedyGroupEstimateV2(hand []game.Card) int {
	left := append([]game.Card(nil), hand...)
	groups := 0
	for len(left) > 0 && groups < 20 {
		candidates := game.GenerateLegalMoves(left, nil, false)
		if len(candidates) == 0 {
			return groups + len(left)
		}
		best, bestValue := candidates[0], greedyMoveValueV2(candidates[0])
		for _, candidate := range candidates[1:] {
			if value := greedyMoveValueV2(candidate); value > bestValue {
				best, bestValue = candidate, value
			}
		}
		left = removeByRank(left, best.Cards)
		groups++
	}
	return groups
}

func greedyMoveValueV2(move game.Move) int {
	value := len(move.Cards)*100 - int(move.MainRank)
	if move.Type == game.Straight || move.Type == game.PairStraight || move.Type == game.Plane || move.Type == game.PlaneWithSingles || move.Type == game.PlaneWithPairs {
		value += len(move.Cards) * 20
	}
	if move.Type == game.Bomb {
		value -= 350
	}
	if move.Type == game.Rocket {
		value -= 500
	}
	return value
}
