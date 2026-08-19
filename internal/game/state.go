package game

type Phase uint8

const (
	PhaseBoot Phase = iota
	PhaseBidding
	PhasePlaying
	PhaseFinished
)

type Role uint8

const (
	RoleUnknown Role = iota
	RoleFarmer
	RoleLandlord
)

func (r Role) String() string {
	if r == RoleLandlord {
		return "地主"
	}
	if r == RoleFarmer {
		return "农民"
	}
	return "未定"
}

type Team uint8

const (
	TeamNone Team = iota
	TeamLandlord
	TeamFarmers
)

type PlayerState struct {
	Seat int
	Name string
	Role Role
	Hand []Card
}

type BidState struct {
	FirstSeat   int
	Actions     int
	HighestBid  int
	HighestSeat int
}

type TrickState struct {
	LeadSeat          int
	LastMove          *Move
	ConsecutivePasses int
}

type ActionKind uint8

const (
	ActionBid ActionKind = iota + 1
	ActionPlay
)

type ActionRecord struct {
	Number int
	Kind   ActionKind
	Seat   int
	Bid    int
	Move   Move
}

type RoundResult struct {
	WinnerTeam Team
	BidScore   int
	Bombs      int
	Spring     bool
	Multiplier int
	BaseScore  int
	Scores     [3]int
}

type GameState struct {
	Phase           Phase
	Round           int
	GameID          uint64
	TurnID          uint64
	CurrentSeat     int
	LandlordSeat    int
	Players         [3]PlayerState
	BottomCards     []Card
	CurrentTrick    TrickState
	History         []ActionRecord
	BidState        BidState
	BidScore        int
	Multiplier      int
	Bombs           int
	SuccessfulPlays [3]int
	WinnerTeam      Team
	Finished        bool
	Result          RoundResult
}

func (s GameState) Clone() GameState {
	copyState := s
	copyState.BottomCards = append([]Card(nil), s.BottomCards...)
	copyState.History = append([]ActionRecord(nil), s.History...)
	for i := range s.Players {
		copyState.Players[i].Hand = append([]Card(nil), s.Players[i].Hand...)
	}
	if s.CurrentTrick.LastMove != nil {
		m := *s.CurrentTrick.LastMove
		m.Cards = append([]Card(nil), m.Cards...)
		copyState.CurrentTrick.LastMove = &m
	}
	return copyState
}
