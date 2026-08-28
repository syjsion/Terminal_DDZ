package local

import (
	"testing"

	"github.com/syjsion/Terminal_DDZ/internal/game"
	"github.com/syjsion/Terminal_DDZ/internal/player"
)

func TestSelectRootISMCTSActionExploresUnderVisitedMove(t *testing.T) {
	stats := []rootISMCTSStat{
		{visits: 20, total: 20000, baseScore: 1000},
		{visits: 2, total: 1800, baseScore: 900},
	}
	if got := selectRootISMCTSAction(stats, 22); got != 1 {
		t.Fatalf("selectRootISMCTSAction = %d, want under-visited action 1", got)
	}
}

func TestExpertV2BudgetIncreasesInEndgame(t *testing.T) {
	mid := player.PlayerView{
		Seat:         0,
		LandlordSeat: 0,
		OwnCards:     make([]game.Card, 16),
		OtherCounts:  map[int]int{1: 12, 2: 11},
	}
	end := player.PlayerView{
		Seat:         0,
		LandlordSeat: 0,
		OwnCards:     make([]game.Card, 6),
		OtherCounts:  map[int]int{1: 2, 2: 4},
	}
	_, midSims, midDepth := expertV2Budget(mid, 8)
	_, endSims, endDepth := expertV2Budget(end, 8)
	if endSims <= midSims || endDepth <= midDepth {
		t.Fatalf("endgame budget (%d,%d) should exceed midgame (%d,%d)", endSims, endDepth, midSims, midDepth)
	}
}

func TestExpertV2FinalValueUsesRolloutEvidence(t *testing.T) {
	strongBasePoorRollout := rootISMCTSStat{baseScore: 5000, visits: 10, total: -20000}
	weakerBaseGoodRollout := rootISMCTSStat{baseScore: 1000, visits: 10, total: 20000}
	if expertV2FinalValue(weakerBaseGoodRollout) <= expertV2FinalValue(strongBasePoorRollout) {
		t.Fatal("rollout evidence should be able to overturn the heuristic prior")
	}
}

func TestSelectExpertRootCandidatesKeepsTacticalMoves(t *testing.T) {
	stats := []rootISMCTSStat{
		{move: game.Move{ID: 1, Type: game.Single}, baseScore: 1000},
		{move: game.Move{ID: 2, Type: game.Pair}, baseScore: 900},
		{move: game.Move{ID: 3, IsPass: true}, baseScore: 100},
		{move: game.Move{ID: 4, Type: game.Bomb}, baseScore: 50},
	}
	view := player.PlayerView{
		Seat:         0,
		LandlordSeat: 2,
		OtherCounts:  map[int]int{1: 5, 2: 2},
		LastMove:     &player.PublicMove{Seat: 1},
	}
	selected := selectExpertRootCandidates(stats, 2, view)
	if !rootCandidatesContainID(selected, 3) {
		t.Fatal("teammate-lead pass candidate should remain in root search")
	}
	if !rootCandidatesContainID(selected, 4) {
		t.Fatal("bomb candidate should remain when enemy has two cards")
	}
}

func TestRolloutAvoidsLettingNextEnemyFinish(t *testing.T) {
	state := rolloutState{current: 0, landlord: 0, lead: 0, winner: -1}
	state.hands[0] = []game.Card{{Rank: game.RankK}, {Rank: game.Rank3}, {Rank: game.Rank3}}
	state.hands[1] = []game.Card{{Rank: game.RankA}}
	state.hands[2] = []game.Card{{Rank: game.Rank4}, {Rank: game.Rank5}, {Rank: game.Rank6}}

	singleK, err := game.AnalyzeMove([]game.Card{{Rank: game.RankK}})
	if err != nil {
		t.Fatal(err)
	}
	pair3, err := game.AnalyzeMove([]game.Card{{Rank: game.Rank3}, {Rank: game.Rank3}})
	if err != nil {
		t.Fatal(err)
	}
	if rolloutMoveScore(state, singleK) >= rolloutMoveScore(state, pair3) {
		t.Fatal("rollout should strongly avoid a move that lets the next enemy go out immediately")
	}
}

func TestRolloutRewardsFeedingTeammateFinish(t *testing.T) {
	state := rolloutState{current: 0, landlord: 2, lead: 0, winner: -1}
	state.hands[0] = []game.Card{{Rank: game.RankK}, {Rank: game.Rank3}, {Rank: game.Rank3}}
	state.hands[1] = []game.Card{{Rank: game.RankA}}
	state.hands[2] = []game.Card{{Rank: game.Rank4}, {Rank: game.Rank5}, {Rank: game.Rank6}}

	singleK, err := game.AnalyzeMove([]game.Card{{Rank: game.RankK}})
	if err != nil {
		t.Fatal(err)
	}
	pair3, err := game.AnalyzeMove([]game.Card{{Rank: game.Rank3}, {Rank: game.Rank3}})
	if err != nil {
		t.Fatal(err)
	}
	if rolloutMoveScore(state, singleK) <= rolloutMoveScore(state, pair3) {
		t.Fatal("rollout should reward a lead that lets the next teammate empty their hand")
	}
}

func rootCandidatesContainID(stats []rootISMCTSStat, id int) bool {
	for _, stat := range stats {
		if stat.move.ID == id {
			return true
		}
	}
	return false
}
