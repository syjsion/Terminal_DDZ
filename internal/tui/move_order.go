package tui

import (
	"sort"

	"github.com/syjsion/Terminal_DDZ/internal/game"
)

// rankHumanMoves returns a display-only ordering. Move IDs remain untouched so
// AI decisions and the engine's canonical ordering stay stable.
func rankHumanMoves(hand []game.Card, moves []game.Move, target *game.Move) []game.Move {
	ordered := append([]game.Move(nil), moves...)
	handCounts := game.RankCounts(hand)
	sort.SliceStable(ordered, func(i, j int) bool {
		a, b := ordered[i], ordered[j]
		if len(a.Cards) == len(hand) && !a.IsPass {
			return len(b.Cards) != len(hand) || b.IsPass
		}
		if len(b.Cards) == len(hand) && !b.IsPass {
			return false
		}
		ac, bc := displayClass(a, target != nil), displayClass(b, target != nil)
		if splitBomb(handCounts, a) {
			ac += 1000
		}
		if splitBomb(handCounts, b) {
			bc += 1000
		}
		if ac != bc {
			return ac < bc
		}
		if len(a.Cards) != len(b.Cards) {
			return len(a.Cards) > len(b.Cards)
		}
		if a.MainRank != b.MainRank {
			return a.MainRank < b.MainRank
		}
		return ranksLess(a.Cards, b.Cards)
	})
	return ordered
}

func displayClass(move game.Move, following bool) int {
	if following {
		switch {
		case move.IsPass:
			return 100
		case move.Type == game.Bomb:
			return 200
		case move.Type == game.Rocket:
			return 300
		default:
			return 0
		}
	}
	switch move.Type {
	case game.PlaneWithPairs, game.PlaneWithSingles, game.Plane:
		return 0
	case game.Straight, game.PairStraight:
		return 100
	case game.TripleWithPair:
		return 200
	case game.TripleWithSingle:
		return 300
	case game.Triple:
		return 400
	case game.Pair:
		return 500
	case game.Single:
		return 600
	case game.FourWithTwoSingles, game.FourWithTwoPairs:
		return 700
	case game.Bomb:
		return 800
	case game.Rocket:
		return 900
	default:
		return 1000
	}
}

func splitBomb(hand map[game.Rank]int, move game.Move) bool {
	if move.IsPass || move.Type == game.Bomb || move.Type == game.Rocket || move.Type == game.FourWithTwoSingles || move.Type == game.FourWithTwoPairs {
		return false
	}
	for rank, used := range game.RankCounts(move.Cards) {
		if hand[rank] == 4 && used > 0 && used < 4 {
			return true
		}
	}
	return false
}

func ranksLess(a, b []game.Card) bool {
	for index := 0; index < len(a) && index < len(b); index++ {
		if a[index].Rank != b[index].Rank {
			return a[index].Rank < b[index].Rank
		}
	}
	return len(a) < len(b)
}

func handTypeLabel(handType game.HandType) string {
	labels := map[game.HandType]string{
		game.Single: "单张", game.Pair: "对子", game.Triple: "三张",
		game.TripleWithSingle: "三带一", game.TripleWithPair: "三带二",
		game.Straight: "顺子", game.PairStraight: "连对", game.Plane: "飞机",
		game.PlaneWithSingles: "飞机带单", game.PlaneWithPairs: "飞机带对",
		game.FourWithTwoSingles: "四带二单", game.FourWithTwoPairs: "四带二对",
		game.Bomb: "炸弹", game.Rocket: "王炸",
	}
	if label, ok := labels[handType]; ok {
		return label
	}
	return "未知"
}
