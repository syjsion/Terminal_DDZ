package game

import (
	"fmt"
	"sort"
	"strings"
)

type HandType uint8

const (
	HandInvalid HandType = iota
	Single
	Pair
	Triple
	TripleWithSingle
	TripleWithPair
	Straight
	PairStraight
	Plane
	PlaneWithSingles
	PlaneWithPairs
	FourWithTwoSingles
	FourWithTwoPairs
	Bomb
	Rocket
)

func (t HandType) String() string {
	names := [...]string{"INVALID", "SINGLE", "PAIR", "TRIPLE", "TRIPLE+SINGLE", "TRIPLE+PAIR", "STRAIGHT", "PAIR-STRAIGHT", "PLANE", "PLANE+SINGLES", "PLANE+PAIRS", "FOUR+SINGLES", "FOUR+PAIRS", "BOMB", "ROCKET"}
	if int(t) >= len(names) {
		return names[0]
	}
	return names[t]
}

type Move struct {
	ID       int
	Type     HandType
	Cards    []Card
	MainRank Rank
	Length   int
	IsPass   bool
}

func PassMove() Move { return Move{Type: HandInvalid, IsPass: true} }

func (m Move) String() string {
	if m.IsPass {
		return "PASS"
	}
	return CardsString(m.Cards)
}

func (m Move) Label() string {
	if m.IsPass {
		return "PASS"
	}
	label := m.String()
	if m.Type == Bomb {
		return label + " [BOMB]"
	}
	if m.Type == Rocket {
		return label + " [ROCKET]"
	}
	return label
}

func (m Move) key() string {
	if m.IsPass {
		return "pass"
	}
	counts := RankCounts(m.Cards)
	var b strings.Builder
	for _, r := range AllRanks {
		if counts[r] > 0 {
			fmt.Fprintf(&b, "%d:%d,", r, counts[r])
		}
	}
	return b.String()
}

func moveFromCounts(t HandType, main Rank, length int, counts map[Rank]int) Move {
	return Move{Type: t, MainRank: main, Length: length, Cards: cardsFromCounts(counts)}
}

func stableSortMoves(moves []Move) {
	priority := func(m Move) int {
		if m.IsPass {
			return 0
		}
		if m.Type == Bomb {
			return 2
		}
		if m.Type == Rocket {
			return 3
		}
		return 1
	}
	sort.SliceStable(moves, func(i, j int) bool {
		a, b := moves[i], moves[j]
		if priority(a) != priority(b) {
			return priority(a) < priority(b)
		}
		if a.Type != b.Type {
			return a.Type < b.Type
		}
		if len(a.Cards) != len(b.Cards) {
			return len(a.Cards) > len(b.Cards)
		}
		if a.MainRank != b.MainRank {
			return a.MainRank < b.MainRank
		}
		return a.key() < b.key()
	})
	for i := range moves {
		moves[i].ID = i
	}
}
