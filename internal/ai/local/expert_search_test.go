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
