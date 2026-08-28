package local

import (
	"context"
	"math/rand"

	"github.com/syjsion/Terminal_DDZ/internal/game"
	"github.com/syjsion/Terminal_DDZ/internal/player"
)

type rolloutState struct {
	hands    [3][]game.Card
	current  int
	landlord int
	lead     int
	target   *game.Move
	passes   int
	winner   int
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
	bestIndex, secondIndex := 0, -1
	bestScore, secondScore := -1<<30, -1<<30
	for i, move := range legal {
		score := rolloutMoveScore(state, move)
		if score > bestScore {
			secondIndex, secondScore = bestIndex, bestScore
			bestIndex, bestScore = i, score
		} else if score > secondScore {
			secondIndex, secondScore = i, score
		}
	}
	// A small amount of near-best policy diversity reduces rollout bias while
	// never selecting an obviously inferior tactical move.
	if secondIndex >= 0 && bestScore-secondScore <= 550 && rng.Intn(100) < 12 {
		return legal[secondIndex]
	}
	return legal[bestIndex]
}

func rolloutMoveScore(state rolloutState, move game.Move) int {
	seat := state.current
	hand := state.hands[seat]
	enemyCards := rolloutEnemyMinCards(state, seat)
	if !move.IsPass && len(move.Cards) == len(hand) {
		return 100000
	}

	next := state
	applyRolloutMove(&next, move)
	if next.winner == seat {
		return 100000
	}

	if move.IsPass {
		score := -100
		if state.target != nil && sameRolloutTeam(state.landlord, seat, state.lead) {
			score += 1700
			if len(state.hands[state.lead]) <= 2 {
				score += 1400
			}
		}
		if enemyCards <= 2 {
			score -= 1000
		}
		score += immediateNextFinishScore(next, seat)
		if next.target == nil && !sameRolloutTeam(state.landlord, seat, next.current) {
			cards := len(next.hands[next.current])
			if cards <= 2 {
				score -= 1600
			} else if cards <= 4 {
				score -= 500
			}
		}
		return score
	}

	remainingHand := next.hands[seat]
	remaining := len(remainingHand)
	score := len(move.Cards)*145 - int(move.MainRank)*5 - remaining*8
	switch move.Type {
	case game.Straight, game.PairStraight, game.Plane, game.PlaneWithSingles, game.PlaneWithPairs:
		score += len(move.Cards) * 38
	case game.TripleWithSingle, game.TripleWithPair:
		score += 220
	case game.Bomb:
		score -= 700
	case game.Rocket:
		score -= 900
	}

	// Endgame rollouts can afford a stronger hand-shape estimate. This keeps
	// the simulated players from breaking a near-finished combination merely
	// to shed one extra high card.
	if remaining <= 8 {
		turns := quickTurnEstimate(remainingHand)
		score -= turns * 240
		switch turns {
		case 1:
			score += 1600
		case 2:
			score += 700
		}
	} else {
		score -= rolloutFragmentation(remainingHand) * 35
	}

	score += immediateNextFinishScore(next, seat)

	// In sampled endgames we know the determinized hands, so we can detect
	// whether this move is likely to retain control without peeking at the real
	// hidden deal.
	if remaining <= 6 || enemyCards <= 3 || (remaining <= 10 && len(move.Cards) >= 5) {
		if !rolloutEnemyCanBeat(next, seat, move) {
			score += 900
			if remaining <= 3 {
				score += 800
			}
		}
	}

	if enemyCards <= 2 {
		score += 450
		if move.Type == game.Bomb || move.Type == game.Rocket {
			score += 950
		}
	}
	if enemyCards == 1 && move.Type == game.Single {
		score -= 850
	}

	if state.target != nil && sameRolloutTeam(state.landlord, seat, state.lead) {
		score -= 1500
		if len(state.hands[state.lead]) <= 2 {
			score -= 900
		}
		if move.Type == game.Bomb || move.Type == game.Rocket {
			score -= 1100
		}
	}

	return score
}

func immediateNextFinishScore(state rolloutState, mover int) int {
	if state.winner >= 0 || !canFinishOnTurn(state, state.current) {
		return 0
	}
	if sameRolloutTeam(state.landlord, mover, state.current) {
		return 6500
	}
	return -14000
}

func canFinishOnTurn(state rolloutState, seat int) bool {
	if seat < 0 || seat >= 3 || len(state.hands[seat]) == 0 {
		return false
	}
	move, err := game.AnalyzeMove(state.hands[seat])
	if err != nil {
		return false
	}
	if state.target == nil {
		return true
	}
	return game.Beats(move, *state.target)
}

func rolloutEnemyCanBeat(state rolloutState, mover int, target game.Move) bool {
	for seat := 0; seat < 3; seat++ {
		if seat == mover || sameRolloutTeam(state.landlord, mover, seat) {
			continue
		}
		if len(game.GenerateLegalMoves(state.hands[seat], &target, false)) > 0 {
			return true
		}
	}
	return false
}

func rolloutFragmentation(hand []game.Card) int {
	counts := game.RankCounts(hand)
	penalty := 0
	for _, rank := range game.AllRanks {
		switch counts[rank] {
		case 1:
			penalty += 3
		case 2:
			penalty += 2
		case 3:
			penalty += 1
		}
	}
	return penalty
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
	landlordMetric := quickTurnEstimate(state.hands[landlord])*600 + len(state.hands[landlord])*22
	farmerMetrics := make([]int, 0, 2)
	for seat := 0; seat < 3; seat++ {
		if seat == landlord {
			continue
		}
		farmerMetrics = append(farmerMetrics, quickTurnEstimate(state.hands[seat])*600+len(state.hands[seat])*22)
	}
	bestFarmer, supportFarmer := farmerMetrics[0], farmerMetrics[1]
	if supportFarmer < bestFarmer {
		bestFarmer, supportFarmer = supportFarmer, bestFarmer
	}
	farmersMetric := bestFarmer + supportFarmer/7
	score := farmersMetric - landlordMetric

	controller := state.lead
	if state.target == nil {
		controller = state.current
	}
	if controller == landlord {
		score += 260
	} else if controller >= 0 {
		score -= 260
	}
	if len(state.hands[landlord]) <= 2 {
		score += 350
	}
	for seat := 0; seat < 3; seat++ {
		if seat != landlord && len(state.hands[seat]) <= 2 {
			score -= 230
		}
	}

	if rootSeat != landlord {
		score = -score
	}
	if score > 4200 {
		return 4200
	}
	if score < -4200 {
		return -4200
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
