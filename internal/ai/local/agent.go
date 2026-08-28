package local

import (
	"context"
	"errors"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/syjsion/Terminal_DDZ/internal/game"
	"github.com/syjsion/Terminal_DDZ/internal/player"
)

type Difficulty string

const (
	Easy   Difficulty = "easy"
	Normal Difficulty = "normal"
	Hard   Difficulty = "hard"
	Expert Difficulty = "expert"
)

func ParseDifficulty(value string) (Difficulty, error) {
	d := Difficulty(strings.ToLower(value))
	if d != Easy && d != Normal && d != Hard && d != Expert {
		return "", errors.New("难度必须是 easy、normal、hard 或 expert")
	}
	return d, nil
}

type Agent struct {
	difficulty Difficulty
	rng        *rand.Rand
	mu         sync.Mutex
}

func New(difficulty Difficulty, seed int64) *Agent {
	if difficulty != Easy && difficulty != Normal && difficulty != Hard && difficulty != Expert {
		difficulty = Normal
	}
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	return &Agent{difficulty: difficulty, rng: rand.New(rand.NewSource(seed))}
}

func (a *Agent) ChooseBid(ctx context.Context, view player.PlayerView, legal []int) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if len(legal) == 0 {
		return 0, errors.New("没有合法叫分")
	}
	strength := handStrength(view.OwnCards)
	want := 0
	switch a.difficulty {
	case Easy:
		a.mu.Lock()
		if strength >= 6 && a.rng.Intn(100) < 45 {
			want = 1 + a.rng.Intn(3)
		}
		a.mu.Unlock()
	case Normal:
		want = bidForStrength(strength, 8, 12, 16)
	case Hard:
		want = bidForStrength(strength, 7, 11, 15)
	case Expert:
		want = bidForStrength(strength, 7, 10, 14)
	}
	return highestLegalAtMost(legal, want), nil
}

func handStrength(cards []game.Card) int {
	counts := game.RankCounts(cards)
	score := 0
	score += counts[game.RankBJ]*5 + counts[game.RankSJ]*4
	score += counts[game.Rank2]*2 + counts[game.RankA]
	for rank, count := range counts {
		if count == 4 {
			score += 7
		} else if count == 3 {
			score += 2
		}
		if rank >= game.RankK && count >= 2 {
			score++
		}
	}
	if counts[game.RankSJ] > 0 && counts[game.RankBJ] > 0 {
		score += 4
	}
	return score
}

func bidForStrength(score, one, two, three int) int {
	switch {
	case score >= three:
		return 3
	case score >= two:
		return 2
	case score >= one:
		return 1
	default:
		return 0
	}
}

func highestLegalAtMost(legal []int, want int) int {
	best := legal[0]
	for _, bid := range legal {
		if bid <= want && bid > best {
			best = bid
		}
	}
	return best
}

func (a *Agent) ChooseMove(ctx context.Context, view player.PlayerView, legal []game.Move) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if len(legal) == 0 {
		return 0, errors.New("没有合法出牌")
	}
	if a.difficulty == Easy {
		return a.chooseEasy(view, legal), nil
	}
	if a.difficulty == Hard {
		return chooseHardV2(ctx, view, legal)
	}
	if a.difficulty == Expert {
		return a.chooseExpertV2(ctx, view, legal)
	}
	bestID, bestScore := legal[0].ID, -1<<30
	for _, move := range legal {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		score := normalScore(view, move)
		if score > bestScore {
			bestID, bestScore = move.ID, score
		}
	}
	return bestID, nil
}

func (a *Agent) chooseEasy(view player.PlayerView, legal []game.Move) int {
	a.mu.Lock()
	defer a.mu.Unlock()
	var ordinary []game.Move
	passID := -1
	for _, move := range legal {
		if move.IsPass {
			passID = move.ID
		} else if move.Type != game.Bomb && move.Type != game.Rocket {
			ordinary = append(ordinary, move)
		}
	}
	if passID >= 0 && len(ordinary) > 0 && a.rng.Intn(100) < 35 {
		return passID
	}
	if len(ordinary) > 0 {
		return ordinary[a.rng.Intn(len(ordinary))].ID
	}
	return legal[a.rng.Intn(len(legal))].ID
}

func normalScore(view player.PlayerView, move game.Move) int {
	if move.IsPass {
		score := 0
		if view.LastMove != nil && sameTeam(view, view.LastMove.Seat) {
			score += 500
		}
		for seat, count := range view.OtherCounts {
			if !sameTeam(view, seat) && count <= 2 {
				score -= 800
			}
		}
		return score
	}
	if len(move.Cards) == len(view.OwnCards) {
		return 100000
	}
	score := len(move.Cards)*80 - int(move.MainRank)*3
	if view.LastMove != nil {
		score = 500 - int(move.MainRank)*8 - len(move.Cards)
	}
	if move.Type == game.Straight || move.Type == game.PairStraight || move.Type == game.Plane || move.Type == game.PlaneWithSingles || move.Type == game.PlaneWithPairs {
		score += len(move.Cards) * 25
	}
	if move.Type == game.Bomb {
		score -= 900
	}
	if move.Type == game.Rocket {
		score -= 1100
	}
	for seat, count := range view.OtherCounts {
		if !sameTeam(view, seat) && count <= 2 {
			score += 700
		}
	}
	if len(view.OwnCards) <= 5 {
		score += len(move.Cards) * 100
	}
	return score
}

func sameTeam(view player.PlayerView, otherSeat int) bool {
	if view.LandlordSeat < 0 {
		return false
	}
	return (view.Seat == view.LandlordSeat) == (otherSeat == view.LandlordSeat)
}

func hardRemainderScore(hand []game.Card, move game.Move) int {
	remaining := removeByRank(hand, move.Cards)
	if len(remaining) == 0 {
		return 100000
	}
	moves := game.GenerateLegalMoves(remaining, nil, false)
	if len(moves) == 0 {
		return -10000
	}
	sort.SliceStable(moves, func(i, j int) bool { return len(moves[i].Cards) > len(moves[j].Cards) })
	left := append([]game.Card(nil), remaining...)
	groups := 0
	for len(left) > 0 && groups < 20 {
		candidates := game.GenerateLegalMoves(left, nil, false)
		if len(candidates) == 0 {
			groups += len(left)
			break
		}
		best := candidates[0]
		for _, candidate := range candidates {
			if candidate.Type != game.Bomb && candidate.Type != game.Rocket && len(candidate.Cards) > len(best.Cards) {
				best = candidate
			}
		}
		left = removeByRank(left, best.Cards)
		groups++
	}
	return -groups * 120
}

func removeByRank(hand, cards []game.Card) []game.Card {
	needed := game.RankCounts(cards)
	result := make([]game.Card, 0, len(hand)-len(cards))
	for _, card := range hand {
		if needed[card.Rank] > 0 {
			needed[card.Rank]--
			continue
		}
		result = append(result, card)
	}
	return result
}
