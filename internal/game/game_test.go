package game

import (
	"fmt"
	"strings"
	"testing"
)

func testCards(spec string) []Card {
	lookup := map[string]Rank{
		"3": Rank3, "4": Rank4, "5": Rank5, "6": Rank6, "7": Rank7, "8": Rank8, "9": Rank9,
		"10": Rank10, "J": RankJ, "Q": RankQ, "K": RankK, "A": RankA, "2": Rank2, "SJ": RankSJ, "BJ": RankBJ,
	}
	seen := make(map[Rank]int)
	var cards []Card
	for _, token := range strings.Fields(spec) {
		r, ok := lookup[token]
		if !ok {
			panic("unknown test rank: " + token)
		}
		cards = append(cards, Card{Rank: r, Suit: Suit(seen[r])})
		seen[r]++
	}
	return cards
}

func TestRankStringAndOrder(t *testing.T) {
	if Rank10.String() != "10" {
		t.Fatalf("Rank10 = %q", Rank10.String())
	}
	if !(Rank3 < Rank10 && Rank10 < RankA && Rank2 < RankSJ && RankSJ < RankBJ) {
		t.Fatal("rank order is invalid")
	}
	if got := CardsString(testCards("BJ 10 3 SJ")); got != "3 10 SJ BJ" {
		t.Fatalf("CardsString = %q", got)
	}
}

func TestAnalyzeMove(t *testing.T) {
	tests := []struct {
		name  string
		cards string
		want  HandType
		valid bool
	}{
		{"single", "A", Single, true}, {"pair", "10 10", Pair, true}, {"joker pair", "SJ SJ", 0, false},
		{"triple", "2 2 2", Triple, true}, {"triple single", "3 3 3 4", TripleWithSingle, true},
		{"triple pair", "A A A 10 10", TripleWithPair, true}, {"straight", "10 J Q K A", Straight, true},
		{"straight with 2", "J Q K A 2", 0, false}, {"straight starting 2", "2 3 4 5 6", 0, false},
		{"pair straight", "Q Q K K A A", PairStraight, true}, {"plane", "3 3 3 4 4 4", Plane, true},
		{"plane single", "3 3 3 4 4 4 5 5", PlaneWithSingles, true},
		{"plane single body wing", "3 3 3 3 4 4 4 5", 0, false},
		{"plane pair", "7 7 7 8 8 8 3 3 J J", PlaneWithPairs, true},
		{"plane has 2", "A A A 2 2 2 3 4", 0, false},
		{"four singles", "6 6 6 6 J J", FourWithTwoSingles, true},
		{"four pairs", "A A A A J J Q Q", FourWithTwoPairs, true},
		{"four same pair wings", "A A A A J J J J", 0, false},
		{"bomb", "2 2 2 2", Bomb, true}, {"rocket", "SJ BJ", Rocket, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			move, err := AnalyzeMove(testCards(tc.cards))
			if tc.valid && (err != nil || move.Type != tc.want) {
				t.Fatalf("AnalyzeMove(%q) = %v, %v", tc.cards, move.Type, err)
			}
			if !tc.valid && err == nil {
				t.Fatalf("AnalyzeMove(%q) unexpectedly valid as %v", tc.cards, move.Type)
			}
		})
	}
}

func mustMove(t *testing.T, cards string) Move {
	t.Helper()
	m, err := AnalyzeMove(testCards(cards))
	if err != nil {
		t.Fatalf("AnalyzeMove(%q): %v", cards, err)
	}
	return m
}

func TestBeats(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"4", "3", true}, {"3 4 5 6 7", "4 5 6 7 8", false},
		{"4 5 6 7 8", "3 4 5 6 7", true}, {"4 5 6 7 8 9", "3 4 5 6 7", false},
		{"3 3 3 3", "A A A", true}, {"4 4 4 4", "3 3 3 3", true},
		{"A A A", "3 3 3 3", false}, {"SJ BJ", "2 2 2 2", true},
	}
	for _, tc := range tests {
		if got := Beats(mustMove(t, tc.a), mustMove(t, tc.b)); got != tc.want {
			t.Errorf("Beats(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestLegalMovesAreValidUniqueAndStable(t *testing.T) {
	hand := testCards("3 3 3 3 4 4 4 5 5 6 6 7 8 9 10 J Q K A 2 SJ BJ")
	first := GenerateLegalMoves(hand, nil, false)
	second := GenerateLegalMoves(hand, nil, false)
	if len(first) == 0 || len(first) != len(second) {
		t.Fatalf("unstable move count: %d, %d", len(first), len(second))
	}
	seen := map[string]bool{}
	for i, move := range first {
		if move.IsPass {
			t.Fatal("lead moves contain PASS")
		}
		if move.ID != i || move.key() != second[i].key() {
			t.Fatalf("unstable move at %d", i)
		}
		if _, err := AnalyzeMove(move.Cards); err != nil {
			t.Fatalf("generated invalid move %q: %v", move.String(), err)
		}
		if seen[move.key()] {
			t.Fatalf("duplicate move %q", move.String())
		}
		seen[move.key()] = true
		for r, n := range RankCounts(move.Cards) {
			if RankCounts(hand)[r] < n {
				t.Fatalf("move uses unavailable cards: %q", move.String())
			}
		}
	}
	if !seen[mustMove(t, "SJ BJ").key()] || !seen[mustMove(t, "3 3 3 3").key()] {
		t.Fatal("rocket or bomb is missing")
	}
}

func TestLegalMovesFollowing(t *testing.T) {
	target := mustMove(t, "9 9")
	moves := GenerateLegalMoves(testCards("3 3 10 10 J J 2 2 5 5 5 5 SJ BJ"), &target, true)
	if len(moves) == 0 || !moves[0].IsPass {
		t.Fatal("following moves must start with PASS")
	}
	for _, move := range moves[1:] {
		if !Beats(move, target) {
			t.Fatalf("move %q does not beat target", move.String())
		}
	}
}

func TestEngineDealAndBidding(t *testing.T) {
	e := NewEngine(WithSeed(7))
	if err := e.StartRound(); err != nil {
		t.Fatal(err)
	}
	s := e.State()
	if len(s.BottomCards) != 3 {
		t.Fatalf("bottom cards = %d", len(s.BottomCards))
	}
	for i := range s.Players {
		if len(s.Players[i].Hand) != 17 {
			t.Fatalf("seat %d hand = %d", i, len(s.Players[i].Hand))
		}
	}
	firstGame := s.GameID
	for i := 0; i < 3; i++ {
		if err := e.ApplyBid(e.State().CurrentSeat, 0); err != nil {
			t.Fatal(err)
		}
	}
	if s = e.State(); s.Phase != PhaseBidding || s.GameID == firstGame || s.BidState.Actions != 0 {
		t.Fatalf("all-pass did not redeal: %+v", s.BidState)
	}
	landlord := s.CurrentSeat
	if err := e.ApplyBid(landlord, 3); err != nil {
		t.Fatal(err)
	}
	s = e.State()
	if s.Phase != PhasePlaying || s.LandlordSeat != landlord || len(s.Players[landlord].Hand) != 20 {
		t.Fatalf("landlord confirmation failed: seat=%d state=%+v", landlord, s)
	}
}

func findMoveID(t *testing.T, moves []Move, predicate func(Move) bool) int {
	t.Helper()
	for _, m := range moves {
		if predicate(m) {
			return m.ID
		}
	}
	t.Fatal("move not found")
	return -1
}

func TestEngineTwoPassesResetTrick(t *testing.T) {
	e := NewEngine(WithSeed(11))
	_ = e.StartRound()
	landlord := e.State().CurrentSeat
	_ = e.ApplyBid(landlord, 3)
	id := findMoveID(t, e.LegalMoves(landlord), func(m Move) bool { return m.Type == Single })
	if err := e.ApplyMove(landlord, id); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		seat := e.State().CurrentSeat
		pass := findMoveID(t, e.LegalMoves(seat), func(m Move) bool { return m.IsPass })
		if err := e.ApplyMove(seat, pass); err != nil {
			t.Fatal(err)
		}
	}
	s := e.State()
	if s.CurrentSeat != landlord || s.CurrentTrick.LastMove != nil || s.CurrentTrick.ConsecutivePasses != 0 {
		t.Fatalf("trick not reset: %+v", s.CurrentTrick)
	}
}

func TestEngineScoringSpringAndBomb(t *testing.T) {
	e := NewEngine(WithSeed(1))
	e.state = GameState{
		Phase: PhasePlaying, CurrentSeat: 0, LandlordSeat: 0, BidScore: 2, Multiplier: 1,
		Players:      [3]PlayerState{{Seat: 0, Role: RoleLandlord, Hand: testCards("3 3 3 3")}, {Seat: 1, Role: RoleFarmer, Hand: testCards("4")}, {Seat: 2, Role: RoleFarmer, Hand: testCards("5")}},
		CurrentTrick: TrickState{LeadSeat: 0},
	}
	id := findMoveID(t, e.LegalMoves(0), func(m Move) bool { return m.Type == Bomb })
	if err := e.ApplyMove(0, id); err != nil {
		t.Fatal(err)
	}
	r := e.State().Result
	if !r.Spring || r.Multiplier != 4 || r.BaseScore != 8 || r.Scores != [3]int{16, -8, -8} {
		t.Fatalf("unexpected result: %+v", r)
	}
}

func TestEngineAntiSpring(t *testing.T) {
	e := NewEngine(WithSeed(1))
	e.state = GameState{
		Phase: PhasePlaying, CurrentSeat: 1, LandlordSeat: 0, BidScore: 1, Multiplier: 1,
		Players:         [3]PlayerState{{Seat: 0, Role: RoleLandlord, Hand: testCards("A")}, {Seat: 1, Role: RoleFarmer, Hand: testCards("3")}, {Seat: 2, Role: RoleFarmer, Hand: testCards("4")}},
		CurrentTrick:    TrickState{LeadSeat: 1},
		SuccessfulPlays: [3]int{1, 0, 0},
	}
	id := findMoveID(t, e.LegalMoves(1), func(m Move) bool { return m.Type == Single })
	if err := e.ApplyMove(1, id); err != nil {
		t.Fatal(err)
	}
	r := e.State().Result
	if !r.Spring || r.Multiplier != 2 || r.Scores != [3]int{-4, 2, 2} {
		t.Fatalf("unexpected anti-spring result: %+v", r)
	}
}

func TestCounterUsesOnlyVisibleInformation(t *testing.T) {
	e := NewEngine(WithSeed(3))
	_ = e.StartRound()
	s := e.State()
	counter := BuildCounter(s, 0)
	total := 0
	for _, n := range counter.UnknownByRank {
		total += n
	}
	if total != 37 {
		t.Fatalf("unknown cards = %d, want 37", total)
	}
	if got := fmt.Sprint(counter.OtherCounts); got == "" {
		t.Fatal("other counts missing")
	}
}
