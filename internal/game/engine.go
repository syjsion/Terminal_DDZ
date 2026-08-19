package game

import (
	"errors"
	"fmt"
	"math/rand"
	"time"
)

var (
	ErrWrongPhase          = errors.New("当前阶段不允许该操作")
	ErrWrongTurn           = errors.New("还没有轮到该玩家")
	ErrInvalidBid          = errors.New("叫分不合法")
	ErrInvalidMove         = errors.New("出牌不合法")
	ErrGameFinished        = errors.New("游戏已经结束")
	ErrNoCardsSelected     = errors.New("请先选择要出的牌")
	ErrSelectedCardsAbsent = errors.New("所选牌不在当前手牌中")
	ErrSelectedHandInvalid = errors.New("所选牌不构成合法牌型")
	ErrSelectedCannotBeat  = errors.New("所选牌无法压过当前牌")
)

type Engine struct {
	state    GameState
	rng      *rand.Rand
	nextDeck []Card
	names    [3]string
}

type Option func(*Engine)

func WithSeed(seed int64) Option {
	return func(e *Engine) { e.rng = rand.New(rand.NewSource(seed)) }
}

func WithDeck(deck []Card) Option {
	return func(e *Engine) { e.nextDeck = append([]Card(nil), deck...) }
}

func WithNames(names [3]string) Option {
	return func(e *Engine) { e.names = names }
}

func NewEngine(opts ...Option) *Engine {
	e := &Engine{
		rng:   rand.New(rand.NewSource(time.Now().UnixNano())),
		names: [3]string{"你", "Worker-01", "Worker-02"},
	}
	for _, opt := range opts {
		opt(e)
	}
	e.state.Phase = PhaseBoot
	e.state.LandlordSeat = -1
	return e
}

func (e *Engine) State() GameState { return e.state.Clone() }

func (e *Engine) StartRound() error {
	if e.state.Phase == PhaseBidding || e.state.Phase == PhasePlaying {
		return ErrWrongPhase
	}
	e.state.Round++
	return e.deal()
}

func (e *Engine) deal() error {
	deck := e.nextDeck
	if len(deck) > 0 {
		e.nextDeck = nil
		if err := validateDeck(deck); err != nil {
			return err
		}
		deck = append([]Card(nil), deck...)
	} else {
		deck = NewDeck()
		e.rng.Shuffle(len(deck), func(i, j int) { deck[i], deck[j] = deck[j], deck[i] })
	}

	e.state.GameID++
	e.state.TurnID++
	e.state.Phase = PhaseBidding
	e.state.LandlordSeat = -1
	e.state.BottomCards = append([]Card(nil), deck[51:]...)
	SortCards(e.state.BottomCards)
	e.state.History = nil
	e.state.BidScore = 0
	e.state.Multiplier = 1
	e.state.Bombs = 0
	e.state.SuccessfulPlays = [3]int{}
	e.state.WinnerTeam = TeamNone
	e.state.Finished = false
	e.state.Result = RoundResult{}
	e.state.CurrentTrick = TrickState{LeadSeat: -1}
	first := e.rng.Intn(3)
	e.state.BidState = BidState{FirstSeat: first, HighestSeat: -1}
	e.state.CurrentSeat = first
	for seat := 0; seat < 3; seat++ {
		hand := make([]Card, 17)
		copy(hand, deck[seat*17:(seat+1)*17])
		SortCards(hand)
		e.state.Players[seat] = PlayerState{Seat: seat, Name: e.names[seat], Role: RoleUnknown, Hand: hand}
	}
	return nil
}

func validateDeck(deck []Card) error {
	if len(deck) != 54 {
		return fmt.Errorf("牌堆必须包含 54 张牌，当前为 %d", len(deck))
	}
	want := NewDeck()
	key := func(c Card) int { return int(c.Rank)*10 + int(c.Suit) }
	counts := make(map[int]int, 54)
	for _, c := range want {
		counts[key(c)]++
	}
	for _, c := range deck {
		counts[key(c)]--
	}
	for _, n := range counts {
		if n != 0 {
			return errors.New("牌堆包含重复或未知实体牌")
		}
	}
	return nil
}

func (e *Engine) LegalBids(seat int) []int {
	if e.state.Phase != PhaseBidding || seat != e.state.CurrentSeat {
		return nil
	}
	bids := []int{0}
	for bid := e.state.BidState.HighestBid + 1; bid <= 3; bid++ {
		bids = append(bids, bid)
	}
	return bids
}

func (e *Engine) ApplyBid(seat, bid int) error {
	if e.state.Phase != PhaseBidding {
		return ErrWrongPhase
	}
	if seat != e.state.CurrentSeat {
		return ErrWrongTurn
	}
	if !containsInt(e.LegalBids(seat), bid) {
		return ErrInvalidBid
	}
	e.state.History = append(e.state.History, ActionRecord{Number: len(e.state.History) + 1, Kind: ActionBid, Seat: seat, Bid: bid})
	e.state.BidState.Actions++
	if bid > e.state.BidState.HighestBid {
		e.state.BidState.HighestBid = bid
		e.state.BidState.HighestSeat = seat
	}
	e.state.TurnID++
	if bid == 3 || e.state.BidState.Actions == 3 {
		if e.state.BidState.HighestBid == 0 {
			return e.deal()
		}
		e.confirmLandlord()
		return nil
	}
	e.state.CurrentSeat = (seat + 1) % 3
	return nil
}

func (e *Engine) confirmLandlord() {
	landlord := e.state.BidState.HighestSeat
	e.state.LandlordSeat = landlord
	e.state.BidScore = e.state.BidState.HighestBid
	for seat := 0; seat < 3; seat++ {
		e.state.Players[seat].Role = RoleFarmer
	}
	e.state.Players[landlord].Role = RoleLandlord
	e.state.Players[landlord].Hand = append(e.state.Players[landlord].Hand, e.state.BottomCards...)
	SortCards(e.state.Players[landlord].Hand)
	e.state.CurrentSeat = landlord
	e.state.CurrentTrick = TrickState{LeadSeat: landlord}
	e.state.Phase = PhasePlaying
}

func (e *Engine) LegalMoves(seat int) []Move {
	if e.state.Phase != PhasePlaying || seat != e.state.CurrentSeat || e.state.Finished {
		return nil
	}
	target := e.state.CurrentTrick.LastMove
	return GenerateLegalMoves(e.state.Players[seat].Hand, target, target != nil)
}

func (e *Engine) ApplyMove(seat, moveID int) error {
	if e.state.Finished {
		return ErrGameFinished
	}
	if e.state.Phase != PhasePlaying {
		return ErrWrongPhase
	}
	if seat != e.state.CurrentSeat {
		return ErrWrongTurn
	}
	legal := e.LegalMoves(seat)
	var selected *Move
	for i := range legal {
		if legal[i].ID == moveID {
			selected = &legal[i]
			break
		}
	}
	if selected == nil {
		return ErrInvalidMove
	}
	move := *selected
	if err := e.validateSelectedMove(seat, move); err != nil {
		return err
	}
	e.state.TurnID++
	if move.IsPass {
		e.state.History = append(e.state.History, ActionRecord{Number: len(e.state.History) + 1, Kind: ActionPlay, Seat: seat, Move: move})
		e.state.CurrentTrick.ConsecutivePasses++
		if e.state.CurrentTrick.ConsecutivePasses == 2 {
			lead := e.state.CurrentTrick.LeadSeat
			e.state.CurrentSeat = lead
			e.state.CurrentTrick = TrickState{LeadSeat: lead}
			return nil
		}
		e.state.CurrentSeat = (seat + 1) % 3
		return nil
	}

	move.Cards = e.removeCards(seat, move.Cards)
	e.state.History = append(e.state.History, ActionRecord{Number: len(e.state.History) + 1, Kind: ActionPlay, Seat: seat, Move: move})
	e.state.SuccessfulPlays[seat]++
	if move.Type == Bomb || move.Type == Rocket {
		e.state.Multiplier *= 2
		e.state.Bombs++
	}
	e.state.CurrentTrick = TrickState{LeadSeat: seat, LastMove: cloneMovePtr(move)}
	if len(e.state.Players[seat].Hand) == 0 {
		e.finish(seat)
		return nil
	}
	e.state.CurrentSeat = (seat + 1) % 3
	return nil
}

// ApplyCards lets a human-facing client submit selected cards without knowing
// the temporary legal move ID. The selection is still matched against the
// engine-generated legal move set before it is applied.
func (e *Engine) ApplyCards(seat int, cards []Card) error {
	if e.state.Finished {
		return ErrGameFinished
	}
	if e.state.Phase != PhasePlaying {
		return ErrWrongPhase
	}
	if seat != e.state.CurrentSeat {
		return ErrWrongTurn
	}
	if len(cards) == 0 {
		return ErrNoCardsSelected
	}
	have := RankCounts(e.state.Players[seat].Hand)
	for rank, count := range RankCounts(cards) {
		if have[rank] < count {
			return ErrSelectedCardsAbsent
		}
	}
	analyzed, err := AnalyzeMove(cards)
	if err != nil {
		return ErrSelectedHandInvalid
	}
	if target := e.state.CurrentTrick.LastMove; target != nil && !Beats(analyzed, *target) {
		return ErrSelectedCannotBeat
	}
	key := analyzed.key()
	for _, legal := range e.LegalMoves(seat) {
		if !legal.IsPass && legal.key() == key {
			return e.ApplyMove(seat, legal.ID)
		}
	}
	return ErrInvalidMove
}

func (e *Engine) validateSelectedMove(seat int, move Move) error {
	if move.IsPass {
		if e.state.CurrentTrick.LastMove == nil {
			return ErrInvalidMove
		}
		return nil
	}
	actual, err := AnalyzeMove(move.Cards)
	if err != nil || actual.Type != move.Type || actual.MainRank != move.MainRank || actual.Length != move.Length {
		return ErrInvalidMove
	}
	have := RankCounts(e.state.Players[seat].Hand)
	for r, n := range RankCounts(move.Cards) {
		if have[r] < n {
			return ErrInvalidMove
		}
	}
	if target := e.state.CurrentTrick.LastMove; target != nil && !Beats(move, *target) {
		return ErrInvalidMove
	}
	return nil
}

func (e *Engine) removeCards(seat int, cards []Card) []Card {
	needed := RankCounts(cards)
	hand := e.state.Players[seat].Hand
	kept := hand[:0]
	removed := make([]Card, 0, len(cards))
	for _, card := range hand {
		if needed[card.Rank] > 0 {
			needed[card.Rank]--
			removed = append(removed, card)
			continue
		}
		kept = append(kept, card)
	}
	e.state.Players[seat].Hand = kept
	return removed
}

func (e *Engine) finish(winnerSeat int) {
	e.state.Finished = true
	e.state.Phase = PhaseFinished
	if winnerSeat == e.state.LandlordSeat {
		e.state.WinnerTeam = TeamLandlord
	} else {
		e.state.WinnerTeam = TeamFarmers
	}
	spring := false
	if e.state.WinnerTeam == TeamLandlord {
		spring = true
		for seat := 0; seat < 3; seat++ {
			if seat != e.state.LandlordSeat && e.state.SuccessfulPlays[seat] > 0 {
				spring = false
			}
		}
	} else if e.state.SuccessfulPlays[e.state.LandlordSeat] == 1 {
		spring = true
	}
	if spring {
		e.state.Multiplier *= 2
	}
	base := e.state.BidScore * e.state.Multiplier
	var scores [3]int
	for seat := 0; seat < 3; seat++ {
		if seat == e.state.LandlordSeat {
			if e.state.WinnerTeam == TeamLandlord {
				scores[seat] = base * 2
			} else {
				scores[seat] = -base * 2
			}
		} else if e.state.WinnerTeam == TeamFarmers {
			scores[seat] = base
		} else {
			scores[seat] = -base
		}
	}
	e.state.Result = RoundResult{WinnerTeam: e.state.WinnerTeam, BidScore: e.state.BidScore, Bombs: e.state.Bombs, Spring: spring, Multiplier: e.state.Multiplier, BaseScore: base, Scores: scores}
}

func cloneMovePtr(move Move) *Move {
	move.Cards = append([]Card(nil), move.Cards...)
	return &move
}

func containsInt(values []int, value int) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}
