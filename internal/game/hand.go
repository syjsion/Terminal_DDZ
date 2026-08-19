package game

import "errors"

var ErrInvalidHand = errors.New("非法牌型")

func AnalyzeMove(cards []Card) (Move, error) {
	if len(cards) == 0 {
		return Move{}, ErrInvalidHand
	}
	counts := RankCounts(cards)
	n := len(cards)

	if n == 1 {
		return canonicalMove(Single, cards[0].Rank, 1, cards), nil
	}
	if n == 2 {
		if counts[RankSJ] == 1 && counts[RankBJ] == 1 {
			return canonicalMove(Rocket, RankBJ, 1, cards), nil
		}
		if r, ok := onlyCount(counts, 2); ok && r < RankSJ {
			return canonicalMove(Pair, r, 1, cards), nil
		}
		return Move{}, ErrInvalidHand
	}
	if n == 3 {
		if r, ok := onlyCount(counts, 3); ok && r < RankSJ {
			return canonicalMove(Triple, r, 1, cards), nil
		}
		return Move{}, ErrInvalidHand
	}
	if n == 4 {
		if r, ok := onlyCount(counts, 4); ok && r < RankSJ {
			return canonicalMove(Bomb, r, 1, cards), nil
		}
		if r, ok := findExactCount(counts, 3); ok {
			return canonicalMove(TripleWithSingle, r, 1, cards), nil
		}
	}
	if n == 5 {
		if r, ok := findExactCount(counts, 3); ok {
			for wing, c := range counts {
				if wing != r && c == 2 && wing < RankSJ {
					return canonicalMove(TripleWithPair, r, 1, cards), nil
				}
			}
		}
	}

	if n >= 5 && allCounts(counts, 1) {
		if low, high, ok := consecutive(counts, 1); ok && high <= RankA && int(high-low)+1 == n {
			return canonicalMove(Straight, high, n, cards), nil
		}
	}
	if n >= 6 && n%2 == 0 && allCounts(counts, 2) {
		if low, high, ok := consecutive(counts, 2); ok && high <= RankA && int(high-low)+1 == n/2 {
			return canonicalMove(PairStraight, high, n/2, cards), nil
		}
	}
	if n >= 6 && n%3 == 0 {
		if high, ok := planeMain(counts, n/3, wingNone); ok {
			return canonicalMove(Plane, high, n/3, cards), nil
		}
	}
	if n >= 8 && n%4 == 0 {
		if high, ok := planeMain(counts, n/4, wingSingles); ok {
			return canonicalMove(PlaneWithSingles, high, n/4, cards), nil
		}
	}
	if n >= 10 && n%5 == 0 {
		if high, ok := planeMain(counts, n/5, wingPairs); ok {
			return canonicalMove(PlaneWithPairs, high, n/5, cards), nil
		}
	}
	if n == 6 {
		if main, ok := findExactCount(counts, 4); ok {
			valid := true
			for r := range counts {
				if r == main {
					continue
				}
				if r == RankSJ || r == RankBJ {
					continue
				}
			}
			if valid {
				return canonicalMove(FourWithTwoSingles, main, 1, cards), nil
			}
		}
	}
	if n == 8 {
		if main, ok := findExactCount(counts, 4); ok {
			pairs := 0
			valid := true
			for r, c := range counts {
				if r == main {
					continue
				}
				if c != 2 || r >= RankSJ {
					valid = false
				}
				pairs++
			}
			if valid && pairs == 2 {
				return canonicalMove(FourWithTwoPairs, main, 1, cards), nil
			}
		}
	}
	return Move{}, ErrInvalidHand
}

func canonicalMove(t HandType, main Rank, length int, cards []Card) Move {
	copyCards := append([]Card(nil), cards...)
	SortCards(copyCards)
	return Move{Type: t, MainRank: main, Length: length, Cards: copyCards}
}

func onlyCount(counts map[Rank]int, want int) (Rank, bool) {
	if len(counts) != 1 {
		return 0, false
	}
	for r, c := range counts {
		return r, c == want
	}
	return 0, false
}

func findExactCount(counts map[Rank]int, want int) (Rank, bool) {
	found := Rank(0)
	for r, c := range counts {
		if c == want {
			if found != 0 {
				return 0, false
			}
			found = r
		}
	}
	return found, found != 0
}

func allCounts(counts map[Rank]int, want int) bool {
	for _, c := range counts {
		if c != want {
			return false
		}
	}
	return true
}

func consecutive(counts map[Rank]int, exact int) (Rank, Rank, bool) {
	low, high := RankBJ, Rank(0)
	for r, c := range counts {
		if c != exact {
			return 0, 0, false
		}
		if r < low {
			low = r
		}
		if r > high {
			high = r
		}
	}
	if high == 0 || int(high-low)+1 != len(counts) {
		return 0, 0, false
	}
	return low, high, true
}

type wingMode uint8

const (
	wingNone wingMode = iota
	wingSingles
	wingPairs
)

func planeMain(counts map[Rank]int, groups int, mode wingMode) (Rank, bool) {
	if groups < 2 {
		return 0, false
	}
	for start := Rank3; int(start)+groups-1 <= int(RankA); start++ {
		high := Rank(int(start) + groups - 1)
		ok := true
		for r := start; r <= high; r++ {
			if counts[r] != 3 {
				ok = false
				break
			}
		}
		if !ok {
			continue
		}
		wingCards, wingRanks := 0, 0
		for r, c := range counts {
			if r >= start && r <= high {
				continue
			}
			switch mode {
			case wingNone:
				if c != 0 {
					ok = false
				}
			case wingSingles:
				if c < 1 || c > 2 {
					ok = false
				}
				wingCards += c
			case wingPairs:
				if c != 2 || r >= RankSJ {
					ok = false
				}
				wingCards += c
				wingRanks++
			}
		}
		if !ok {
			continue
		}
		switch mode {
		case wingNone:
			if wingCards == 0 {
				return high, true
			}
		case wingSingles:
			if wingCards == groups {
				return high, true
			}
		case wingPairs:
			if wingCards == groups*2 && wingRanks == groups {
				return high, true
			}
		}
	}
	return 0, false
}

func Beats(move, target Move) bool {
	if move.IsPass || target.IsPass {
		return false
	}
	if move.Type == Rocket {
		return target.Type != Rocket
	}
	if target.Type == Rocket {
		return false
	}
	if move.Type == Bomb {
		return target.Type != Bomb || move.MainRank > target.MainRank
	}
	if target.Type == Bomb || move.Type != target.Type || move.Length != target.Length || len(move.Cards) != len(target.Cards) {
		return false
	}
	return move.MainRank > target.MainRank
}
