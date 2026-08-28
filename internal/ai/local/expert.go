package local

import (
	"context"
	"math/rand"
	"sort"

	"github.com/syjsion/Terminal_DDZ/internal/game"
	"github.com/syjsion/Terminal_DDZ/internal/player"
)

type expertCandidate struct {
	move      game.Move
	baseScore int
}

type rolloutState struct {
	hands    [3][]game.Card
	current  int
	landlord int
	lead     int
	target   *game.Move
	passes   int
	winner   int
}

func (a *Agent) chooseExpert(ctx context.Context, view player.PlayerView, legal []game.Move) (int, error) {
	unseen := unseenRankCounts(view)
	memo := make(map[string]int, 256)
	candidates := make([]expertCandidate, 0, len(legal))
	for _, move := range legal {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		candidates = append(candidates, expertCandidate{
			move:      move,
			baseScore: hardV2Score(ctx, view, move, unseen, memo),
		})
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].baseScore > candidates[j].baseScore
	})

	limit, samples, depth := expertBudget(view, len(candidates))
	if limit == 0 || samples == 0 {
		return candidates[0].move.ID, nil
	}
	if limit > len(candidates) {
		limit = len(candidates)
	}

	a.mu.Lock()
	seed := a.rng.Int63()
	a.mu.Unlock()
	rng := rand.New(rand.NewSource(seed))

	bestID := candidates[0].move.ID
	bestScore := candidates[0].baseScore
	for i := 0; i < limit; i++ {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		total, completed := 0, 0
		for sample := 0; sample < samples; sample++ {
			if err := ctx.Err(); err != nil {
				return 0, err
			}
			state, ok := sampleRolloutState(view, rng)
			if !ok {
				continue
			}
			applyRolloutMove(&state, candidates[i].move)
			total += runExpertRollout(ctx, state, view.Seat, depth, rng)
			completed++
		}
		if completed == 0 {
			continue
		}
		rolloutAverage := total / completed
		finalScore := candidates[i].baseScore + rolloutAverage*2
		if i == 0 || finalScore > bestScore {
			bestScore = finalScore
			bestID = candidates[i].move.ID
		}
	}
	return bestID, nil
}

func expertBudget(view player.PlayerView, candidateCount int) (limit, samples, depth int) {
	if candidateCount == 0 {
		return 0, 0, 0
	}
	enemyCards := enemyMinCards(view)
	switch {
	case len(view.OwnCards) <= 8 || enemyCards <= 3:
		return 7, 16, 36
	case len(view.OwnCards) <= 12 || enemyCards <= 6:
		return 6, 12, 30
	default:
		return 5, 8, 22
	}
}

func sampleRolloutState(view player.PlayerView, rng *rand.Rand) (rolloutState, bool) {
	state := rolloutState{current: view.Seat, landlord: view.LandlordSeat, lead: view.Seat, winner: -1}
	state.hands[view.Seat] = append([]game.Card(nil), view.OwnCards...)

	pool := unknownCardsForExpert(view)
	needed := [3]int{}
	for seat, count := range view.OtherCounts {
		if seat >= 0 && seat < len(needed) {
			needed[seat] = count
		}
	}

	if view.LandlordSeat >= 0 && view.LandlordSeat != view.Seat {
		reserveKnownBottomCards(view, &state, &pool, &needed)
	}

	totalNeeded := 0
	for seat := 0; seat < 3; seat++ {
		if seat != view.Seat {
			totalNeeded += needed[seat]
		}
	}
	if totalNeeded != len(pool) {
		return rolloutState{}, false
	}

	rng.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
	index := 0
	for seat := 0; seat < 3; seat++ {
		if seat == view.Seat {
			continue
		}
		count := needed[seat]
		state.hands[seat] = append(state.hands[seat], pool[index:index+count]...)
		index += count
	}

	if view.LastMove != nil {
		m := view.LastMove.Move
		m.Cards = append([]game.Card(nil), m.Cards...)
		state.target = &m
		state.lead = view.LastMove.Seat
		delta := (view.Seat - view.LastMove.Seat + 3) % 3
		if delta == 2 {
			state.passes = 1
		}
	}
	return state, true
}

func unknownCardsForExpert(view player.PlayerView) []game.Card {
	counts := unseenRankCounts(view)
	cards := make([]game.Card, 0, 37)
	for _, rank := range game.AllRanks {
		for i := 0; i < counts[rank]; i++ {
			cards = append(cards, game.Card{Rank: rank, Suit: game.Suit(i)})
		}
	}
	return cards
}

func reserveKnownBottomCards(view player.PlayerView, state *rolloutState, pool *[]game.Card, needed *[3]int) {
	landlord := view.LandlordSeat
	if landlord < 0 || landlord >= 3 || needed[landlord] <= 0 {
		return
	}
	known := game.RankCounts(view.BottomPublic)
	for _, action := range view.PlayedCards {
		if action.Kind != game.ActionPlay || action.Seat != landlord || action.Move.IsPass {
			continue
		}
		for rank, count := range game.RankCounts(action.Move.Cards) {
			known[rank] -= count
			if known[rank] < 0 {
				known[rank] = 0
			}
		}
	}
	for _, rank := range game.AllRanks {
		for known[rank] > 0 && needed[landlord] > 0 {
			card, ok := takeRankFromPool(pool, rank)
			if !ok {
				break
			}
			state.hands[landlord] = append(state.hands[landlord], card)
			needed[landlord]--
			known[rank]--
		}
	}
}

func takeRankFromPool(pool *[]game.Card, rank game.Rank) (game.Card, bool) {
	cards := *pool
	for i, card := range cards {
		if card.Rank != rank {
			continue
		}
		cards[i] = cards[len(cards)-1]
		*pool = cards[:len(cards)-1]
		return card, true
	}
	return game.Card{}, false
}

func runExpertRollout(ctx context.Context, state rolloutState, rootSeat, maxDepth int, rng *rand.Rand) int {
	for depth := 0; depth < maxDepth && state.winner < 0; depth++ {
		if ctx.Err() != nil {
			return evaluateRolloutPosition(state, rootSeat)
		}
		legal := game.GenerateLegalMoves(state.hands[state.current], state.target, state.target != nil)
		if len(legal) == 0 {
			return evaluateRolloutPosition(state, rootSeat)
		}
		move := chooseRolloutMove(state, legal, rng)
		applyRolloutMove(&state, move)
	}
	if state.winner >= 0 {
		if sameRolloutTeam(state.landlord, rootSeat, state.winner) {
			return 10000
		}
		return -10000
	}
	return evaluateRolloutPosition(state, rootSeat)
}

func chooseRolloutMove(state rolloutState, legal []game.Move, rng *rand.Rand) game.Move {
	best := legal[0]
	bestScore := -1 << 30
	for _, move := range legal {
		score := rolloutMoveScore(state, move)
		if score > bestScore || (score == bestScore && rng.Intn(2) == 0) {
			best, bestScore = move, score
		}
	}
	return best
}

func rolloutMoveScore(state rolloutState, move game.Move) int {
	seat := state.current
	hand := state.hands[seat]
	enemyCards := rolloutEnemyMinCards(state, seat)
	if move.IsPass {
		score := -80
		if state.target != nil && sameRolloutTeam(state.landlord, seat, state.lead) {
			score += 1200
			if len(state.hands[state.lead]) <= 2 {
				score += 800
			}
		}
		if enemyCards <= 2 {
			score -= 900
		}
		return score
	}
	if len(move.Cards) == len(hand) {
		return 100000
	}

	remaining := len(hand) - len(move.Cards)
	score := len(move.Cards)*140 - int(move.MainRank)*5 - remaining*10
	switch move.Type {
	case game.Straight, game.PairStraight, game.Plane, game.PlaneWithSingles, game.PlaneWithPairs:
		score += len(move.Cards) * 35
	case game.TripleWithSingle, game.TripleWithPair:
		score += 180
	case game.Bomb:
		score -= 650
	case game.Rocket:
		score -= 850
	}
	if remaining <= 2 {
		score += 900
	}
	if enemyCards <= 2 {
		score += 500
		if move.Type == game.Bomb || move.Type == game.Rocket {
			score += 900
		}
	}
	if enemyCards == 1 && move.Type == game.Single {
		score -= 1000
	}
	if state.target != nil && sameRolloutTeam(state.landlord, seat, state.lead) {
		score -= 1200
		if move.Type == game.Bomb || move.Type == game.Rocket {
			score -= 1000
		}
	}
	return score
}

func applyRolloutMove(state *rolloutState, move game.Move) {
	seat := state.current
	if move.IsPass {
		state.passes++
		if state.passes >= 2 {
			state.current = state.lead
			state.target = nil
			state.passes = 0
			return
		}
		state.current = (seat + 1) % 3
		return
	}

	state.hands[seat] = removeByRank(state.hands[seat], move.Cards)
	if len(state.hands[seat]) == 0 {
		state.winner = seat
		return
	}
	m := move
	m.Cards = append([]game.Card(nil), move.Cards...)
	state.target = &m
	state.lead = seat
	state.passes = 0
	state.current = (seat + 1) % 3
}

func evaluateRolloutPosition(state rolloutState, rootSeat int) int {
	landlord := state.landlord
	if landlord < 0 || landlord >= 3 {
		return 0
	}
	landlordTurns := quickTurnEstimate(state.hands[landlord])
	farmerTurns := 99
	for seat := 0; seat < 3; seat++ {
		if seat == landlord {
			continue
		}
		if turns := quickTurnEstimate(state.hands[seat]); turns < farmerTurns {
			farmerTurns = turns
		}
	}
	score := (farmerTurns - landlordTurns) * 450
	if rootSeat != landlord {
		score = -score
	}
	if score > 3500 {
		return 3500
	}
	if score < -3500 {
		return -3500
	}
	return score
}

func quickTurnEstimate(hand []game.Card) int {
	if len(hand) == 0 {
		return 0
	}
	if len(hand) <= 8 {
		return greedyGroupEstimateV2(hand)
	}
	counts := game.RankCounts(hand)
	groups := 0
	for _, rank := range game.AllRanks {
		if counts[rank] > 0 {
			groups++
		}
	}
	if groups < 1 {
		groups = 1
	}
	return groups
}

func rolloutEnemyMinCards(state rolloutState, seat int) int {
	best := 1 << 30
	for other := 0; other < 3; other++ {
		if other == seat || sameRolloutTeam(state.landlord, seat, other) {
			continue
		}
		if len(state.hands[other]) < best {
			best = len(state.hands[other])
		}
	}
	return best
}

func sameRolloutTeam(landlord, a, b int) bool {
	if landlord < 0 {
		return false
	}
	return (a == landlord) == (b == landlord)
}
