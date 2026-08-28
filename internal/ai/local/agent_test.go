package local

import (
	"context"
	"testing"

	"github.com/syjsion/Terminal_DDZ/internal/game"
	"github.com/syjsion/Terminal_DDZ/internal/player"
)

func TestAgentsFinishDeterministicGames(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping local AI simulation in short mode")
	}
	difficulties := []Difficulty{Easy, Normal, Hard}
	for _, difficulty := range difficulties {
		t.Run(string(difficulty), func(t *testing.T) {
			for seed := int64(1); seed <= 34; seed++ {
				runDeterministicGame(t, difficulty, seed)
			}
		})
	}
}

func TestExpertFinishesDeterministicGames(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping expert AI simulation in short mode")
	}
	for seed := int64(1); seed <= 3; seed++ {
		runDeterministicGame(t, Expert, seed)
	}
}

func runDeterministicGame(t *testing.T, difficulty Difficulty, seed int64) {
	t.Helper()
	e := game.NewEngine(game.WithSeed(seed))
	if err := e.StartRound(); err != nil {
		t.Fatal(err)
	}
	agents := [3]*Agent{New(difficulty, seed*10+1), New(difficulty, seed*10+2), New(difficulty, seed*10+3)}
	for action := 0; action < 1000 && !e.State().Finished; action++ {
		state := e.State()
		if err := game.ValidateInvariants(state); err != nil {
			t.Fatalf("seed %d action %d: invariant: %v", seed, action, err)
		}
		seat := state.CurrentSeat
		view := player.BuildView(state, seat)
		switch state.Phase {
		case game.PhaseBidding:
			legal := e.LegalBids(seat)
			bid, err := agents[seat].ChooseBid(context.Background(), view, legal)
			if err != nil {
				t.Fatalf("seed %d: bid: %v", seed, err)
			}
			if err := e.ApplyBid(seat, bid); err != nil {
				t.Fatalf("seed %d: apply bid %d: %v", seed, bid, err)
			}
		case game.PhasePlaying:
			legal := e.LegalMoves(seat)
			moveID, err := agents[seat].ChooseMove(context.Background(), view, legal)
			if err != nil {
				t.Fatalf("seed %d: move: %v", seed, err)
			}
			if err := e.ApplyMove(seat, moveID); err != nil {
				t.Fatalf("seed %d: apply move %d: %v", seed, moveID, err)
			}
		}
	}
	if !e.State().Finished {
		t.Fatalf("seed %d did not finish", seed)
	}
	if err := game.ValidateInvariants(e.State()); err != nil {
		t.Fatalf("seed %d finished invariant: %v", seed, err)
	}
}

func TestParseExpertDifficulty(t *testing.T) {
	difficulty, err := ParseDifficulty("EXPERT")
	if err != nil || difficulty != Expert {
		t.Fatalf("ParseDifficulty(EXPERT) = %q, %v", difficulty, err)
	}
}

func TestAgentHonorsCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	agent := New(Normal, 1)
	if _, err := agent.ChooseMove(ctx, player.PlayerView{}, []game.Move{game.PassMove()}); err == nil {
		t.Fatal("expected cancellation error")
	}
}
