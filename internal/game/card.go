package game

import (
	"fmt"
	"sort"
	"strings"
)

type Rank uint8

const (
	Rank3 Rank = iota + 3
	Rank4
	Rank5
	Rank6
	Rank7
	Rank8
	Rank9
	Rank10
	RankJ
	RankQ
	RankK
	RankA
	Rank2
	RankSJ
	RankBJ
)

var AllRanks = []Rank{Rank3, Rank4, Rank5, Rank6, Rank7, Rank8, Rank9, Rank10, RankJ, RankQ, RankK, RankA, Rank2, RankSJ, RankBJ}

func (r Rank) String() string {
	switch r {
	case Rank3, Rank4, Rank5, Rank6, Rank7, Rank8, Rank9:
		return fmt.Sprintf("%d", r)
	case Rank10:
		return "10"
	case RankJ:
		return "J"
	case RankQ:
		return "Q"
	case RankK:
		return "K"
	case RankA:
		return "A"
	case Rank2:
		return "2"
	case RankSJ:
		return "SJ"
	case RankBJ:
		return "BJ"
	default:
		return "?"
	}
}

type Suit uint8

const (
	Club Suit = iota
	Diamond
	Heart
	Spade
	Joker
)

type Card struct {
	Rank Rank
	Suit Suit
}

func (c Card) String() string { return c.Rank.String() }

func NewDeck() []Card {
	deck := make([]Card, 0, 54)
	for _, r := range AllRanks[:13] {
		for s := Club; s <= Spade; s++ {
			deck = append(deck, Card{Rank: r, Suit: s})
		}
	}
	return append(deck, Card{Rank: RankSJ, Suit: Joker}, Card{Rank: RankBJ, Suit: Joker})
}

func SortCards(cards []Card) {
	sort.Slice(cards, func(i, j int) bool {
		if cards[i].Rank == cards[j].Rank {
			return cards[i].Suit < cards[j].Suit
		}
		return cards[i].Rank < cards[j].Rank
	})
}

func CardsString(cards []Card) string {
	if len(cards) == 0 {
		return ""
	}
	copyCards := append([]Card(nil), cards...)
	SortCards(copyCards)
	parts := make([]string, len(copyCards))
	for i, c := range copyCards {
		parts[i] = c.String()
	}
	return strings.Join(parts, " ")
}

func RankCounts(cards []Card) map[Rank]int {
	counts := make(map[Rank]int, len(cards))
	for _, c := range cards {
		counts[c.Rank]++
	}
	return counts
}

func cardsFromCounts(counts map[Rank]int) []Card {
	var cards []Card
	for _, r := range AllRanks {
		for i := 0; i < counts[r]; i++ {
			cards = append(cards, Card{Rank: r, Suit: Suit(i)})
		}
	}
	return cards
}
