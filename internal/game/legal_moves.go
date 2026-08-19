package game

func GenerateLegalMoves(hand []Card, target *Move, canPass bool) []Move {
	counts := RankCounts(hand)
	all := enumerateMoves(counts)
	seen := make(map[string]struct{}, len(all))
	moves := make([]Move, 0, len(all)+1)
	if canPass && target != nil {
		moves = append(moves, PassMove())
	}
	for _, move := range all {
		if target != nil && !Beats(move, *target) {
			continue
		}
		key := move.key()
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		moves = append(moves, move)
	}
	stableSortMoves(moves)
	return moves
}

func enumerateMoves(hand map[Rank]int) []Move {
	var moves []Move
	for _, r := range AllRanks {
		c := hand[r]
		if c >= 1 {
			moves = append(moves, moveFromCounts(Single, r, 1, map[Rank]int{r: 1}))
		}
		if c >= 2 && r < RankSJ {
			moves = append(moves, moveFromCounts(Pair, r, 1, map[Rank]int{r: 2}))
		}
		if c >= 3 && r < RankSJ {
			moves = append(moves, moveFromCounts(Triple, r, 1, map[Rank]int{r: 3}))
			for _, w := range AllRanks {
				if w != r && hand[w] >= 1 {
					moves = append(moves, moveFromCounts(TripleWithSingle, r, 1, map[Rank]int{r: 3, w: 1}))
				}
				if w != r && w < RankSJ && hand[w] >= 2 {
					moves = append(moves, moveFromCounts(TripleWithPair, r, 1, map[Rank]int{r: 3, w: 2}))
				}
			}
		}
		if c >= 4 && r < RankSJ {
			moves = append(moves, moveFromCounts(Bomb, r, 1, map[Rank]int{r: 4}))
			for _, wings := range chooseCards(hand, map[Rank]bool{r: true}, 2, 2) {
				combined := cloneCounts(wings)
				combined[r] = 4
				moves = append(moves, moveFromCounts(FourWithTwoSingles, r, 1, combined))
			}
			for _, pairs := range choosePairs(hand, map[Rank]bool{r: true}, 2) {
				combined := cloneCounts(pairs)
				combined[r] = 4
				moves = append(moves, moveFromCounts(FourWithTwoPairs, r, 1, combined))
			}
		}
	}
	if hand[RankSJ] >= 1 && hand[RankBJ] >= 1 {
		moves = append(moves, moveFromCounts(Rocket, RankBJ, 1, map[Rank]int{RankSJ: 1, RankBJ: 1}))
	}

	moves = append(moves, sequenceMoves(hand, 1, 5, Straight)...)
	moves = append(moves, sequenceMoves(hand, 2, 3, PairStraight)...)

	for start := Rank3; start <= RankA; start++ {
		for end := start; end <= RankA && hand[end] >= 3; end++ {
			groups := int(end-start) + 1
			if groups < 2 {
				continue
			}
			body := make(map[Rank]int, groups)
			excluded := make(map[Rank]bool, groups)
			for r := start; r <= end; r++ {
				body[r] = 3
				excluded[r] = true
			}
			moves = append(moves, moveFromCounts(Plane, end, groups, body))
			for _, wings := range chooseCards(hand, excluded, groups, 2) {
				combined := mergeCounts(body, wings)
				moves = append(moves, moveFromCounts(PlaneWithSingles, end, groups, combined))
			}
			for _, wings := range choosePairs(hand, excluded, groups) {
				combined := mergeCounts(body, wings)
				moves = append(moves, moveFromCounts(PlaneWithPairs, end, groups, combined))
			}
		}
	}
	return moves
}

func sequenceMoves(hand map[Rank]int, need, minGroups int, kind HandType) []Move {
	var result []Move
	for start := Rank3; start <= RankA; start++ {
		for end := start; end <= RankA && hand[end] >= need; end++ {
			groups := int(end-start) + 1
			if groups < minGroups {
				continue
			}
			counts := make(map[Rank]int, groups)
			for r := start; r <= end; r++ {
				counts[r] = need
			}
			result = append(result, moveFromCounts(kind, end, groups, counts))
		}
	}
	return result
}

func chooseCards(hand map[Rank]int, excluded map[Rank]bool, total, perRankCap int) []map[Rank]int {
	var result []map[Rank]int
	current := make(map[Rank]int)
	var walk func(index, remaining int)
	walk = func(index, remaining int) {
		if remaining == 0 {
			result = append(result, cloneCounts(current))
			return
		}
		if index >= len(AllRanks) {
			return
		}
		r := AllRanks[index]
		if excluded[r] {
			walk(index+1, remaining)
			return
		}
		max := hand[r]
		if max > perRankCap {
			max = perRankCap
		}
		if max > remaining {
			max = remaining
		}
		for n := 0; n <= max; n++ {
			if n > 0 {
				current[r] = n
			} else {
				delete(current, r)
			}
			walk(index+1, remaining-n)
		}
		delete(current, r)
	}
	walk(0, total)
	return result
}

func choosePairs(hand map[Rank]int, excluded map[Rank]bool, total int) []map[Rank]int {
	var eligible []Rank
	for _, r := range AllRanks {
		if !excluded[r] && r < RankSJ && hand[r] >= 2 {
			eligible = append(eligible, r)
		}
	}
	var result []map[Rank]int
	var walk func(start int, picked []Rank)
	walk = func(start int, picked []Rank) {
		if len(picked) == total {
			counts := make(map[Rank]int, total)
			for _, r := range picked {
				counts[r] = 2
			}
			result = append(result, counts)
			return
		}
		for i := start; i < len(eligible); i++ {
			walk(i+1, append(picked, eligible[i]))
		}
	}
	walk(0, nil)
	return result
}

func cloneCounts(src map[Rank]int) map[Rank]int {
	dst := make(map[Rank]int, len(src))
	for r, c := range src {
		dst[r] = c
	}
	return dst
}

func mergeCounts(a, b map[Rank]int) map[Rank]int {
	result := cloneCounts(a)
	for r, c := range b {
		result[r] += c
	}
	return result
}
