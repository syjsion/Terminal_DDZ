package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/syjsion/Terminal_DDZ/internal/ai/local"
	"github.com/syjsion/Terminal_DDZ/internal/game"
	"github.com/syjsion/Terminal_DDZ/internal/player"
)

type matchupResult struct {
	games  int
	wins   int
	points int
}

func main() {
	games := flag.Int("games", 30, "games per matchup")
	seed := flag.Int64("seed", 1, "first deterministic game seed")
	flag.Parse()
	if *games < 1 {
		fmt.Fprintln(os.Stderr, "--games must be >= 1")
		os.Exit(2)
	}

	ctx := context.Background()
	started := time.Now()
	landlord, err := runMatchup(ctx, *games, *seed, true)
	if err != nil {
		fmt.Fprintln(os.Stderr, "expert landlord benchmark:", err)
		os.Exit(1)
	}
	farmers, err := runMatchup(ctx, *games, *seed+100000, false)
	if err != nil {
		fmt.Fprintln(os.Stderr, "expert farmers benchmark:", err)
		os.Exit(1)
	}

	fmt.Printf("Expert vs Hard benchmark (%d games per role)\n", *games)
	fmt.Printf("Expert landlord: %d/%d wins (%.1f%%), team points %+d (avg %+.2f)\n", landlord.wins, landlord.games, percent(landlord), landlord.points, averagePoints(landlord))
	fmt.Printf("Expert farmers:  %d/%d wins (%.1f%%), team points %+d (avg %+.2f)\n", farmers.wins, farmers.games, percent(farmers), farmers.points, averagePoints(farmers))
	combined := matchupResult{
		games:  landlord.games + farmers.games,
		wins:   landlord.wins + farmers.wins,
		points: landlord.points + farmers.points,
	}
	fmt.Printf("Combined:        %d/%d wins (%.1f%%), team points %+d (avg %+.2f)\n", combined.wins, combined.games, percent(combined), combined.points, averagePoints(combined))
	fmt.Printf("Elapsed: %s\n", time.Since(started).Round(time.Millisecond))
}

func runMatchup(ctx context.Context, games int, firstSeed int64, expertLandlord bool) (matchupResult, error) {
	result := matchupResult{games: games}
	for i := 0; i < games; i++ {
		seed := firstSeed + int64(i)
		landlordSeat := i % 3
		round, err := runForcedRoleGame(ctx, seed, landlordSeat, expertLandlord)
		if err != nil {
			return result, fmt.Errorf("seed %d: %w", seed, err)
		}
		if expertLandlord && round.WinnerTeam == game.TeamLandlord {
			result.wins++
		}
		if !expertLandlord && round.WinnerTeam == game.TeamFarmers {
			result.wins++
		}
		result.points += expertTeamPoints(round, landlordSeat, expertLandlord)
	}
	return result, nil
}

func runForcedRoleGame(ctx context.Context, seed int64, landlordSeat int, expertLandlord bool) (game.RoundResult, error) {
	e := game.NewEngine(game.WithSeed(seed))
	if err := e.StartRound(); err != nil {
		return game.RoundResult{}, err
	}
	for e.State().Phase == game.PhaseBidding {
		state := e.State()
		bid := 0
		if state.CurrentSeat == landlordSeat {
			bid = 3
		}
		if err := e.ApplyBid(state.CurrentSeat, bid); err != nil {
			return game.RoundResult{}, err
		}
	}

	var agents [3]*local.Agent
	for seat := 0; seat < 3; seat++ {
		difficulty := local.Hard
		if (seat == landlordSeat) == expertLandlord {
			difficulty = local.Expert
		}
		agents[seat] = local.New(difficulty, seed*10+int64(seat)+1)
	}

	for action := 0; action < 1000 && !e.State().Finished; action++ {
		state := e.State()
		if err := game.ValidateInvariants(state); err != nil {
			return game.RoundResult{}, err
		}
		seat := state.CurrentSeat
		view := player.BuildView(state, seat)
		moveID, err := agents[seat].ChooseMove(ctx, view, e.LegalMoves(seat))
		if err != nil {
			return game.RoundResult{}, err
		}
		if err := e.ApplyMove(seat, moveID); err != nil {
			return game.RoundResult{}, err
		}
	}
	state := e.State()
	if !state.Finished {
		return game.RoundResult{}, fmt.Errorf("game did not finish")
	}
	return state.Result, nil
}

func expertTeamPoints(round game.RoundResult, landlordSeat int, expertLandlord bool) int {
	if expertLandlord {
		return round.Scores[landlordSeat]
	}
	points := 0
	for seat := 0; seat < 3; seat++ {
		if seat != landlordSeat {
			points += round.Scores[seat]
		}
	}
	return points
}

func percent(result matchupResult) float64 {
	if result.games == 0 {
		return 0
	}
	return float64(result.wins) * 100 / float64(result.games)
}

func averagePoints(result matchupResult) float64 {
	if result.games == 0 {
		return 0
	}
	return float64(result.points) / float64(result.games)
}
