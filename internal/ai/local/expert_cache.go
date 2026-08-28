package local

import (
	"math/rand"

	"github.com/syjsion/Terminal_DDZ/internal/game"
)

type expertTacticalCache struct {
	rootSeat int
	salt     int64
	values   map[string]int
	hits     int
}

func newExpertTacticalCache(rootSeat int, salt int64) *expertTacticalCache {
	return &expertTacticalCache{
		rootSeat: rootSeat,
		salt:     salt,
		values:   make(map[string]int, 128),
	}
}

func (c *expertTacticalCache) lookup(state rolloutState, plies, maxDepth int) (int, bool) {
	value, ok := c.values[expertTacticalStateKey(state, c.rootSeat, plies, maxDepth)]
	if ok {
		c.hits++
	}
	return value, ok
}

func (c *expertTacticalCache) store(state rolloutState, plies, maxDepth, value int) {
	c.values[expertTacticalStateKey(state, c.rootSeat, plies, maxDepth)] = value
}

func (c *expertTacticalCache) rolloutRNG(state rolloutState, plies, maxDepth int) *rand.Rand {
	key := expertTacticalStateKey(state, c.rootSeat, plies, maxDepth)
	return rand.New(rand.NewSource(expertTacticalSeed(c.salt, key)))
}

// expertTacticalStateKey intentionally encodes ranks rather than suits because
// Dou Dizhu move legality and comparison are rank-based. Search depth belongs in
// the key because the same card position can have a different value at a
// different tactical horizon.
func expertTacticalStateKey(state rolloutState, rootSeat, plies, maxDepth int) string {
	key := make([]byte, 0, len(game.AllRanks)*3+16)
	for seat := 0; seat < 3; seat++ {
		counts := game.RankCounts(state.hands[seat])
		for _, rank := range game.AllRanks {
			key = append(key, byte(counts[rank]))
		}
	}
	key = append(key,
		encodeExpertSeat(state.current),
		encodeExpertSeat(state.landlord),
		encodeExpertSeat(state.lead),
		byte(state.passes),
		encodeExpertSeat(state.winner),
		encodeExpertSeat(rootSeat),
		byte(plies),
		byte(maxDepth),
	)
	if state.target == nil {
		return string(append(key, 0))
	}
	pass := byte(0)
	if state.target.IsPass {
		pass = 1
	}
	key = append(key,
		1,
		byte(state.target.Type),
		byte(state.target.MainRank),
		byte(state.target.Length),
		byte(len(state.target.Cards)),
		pass,
	)
	return string(key)
}

func encodeExpertSeat(seat int) byte {
	return byte(seat + 1)
}

func expertTacticalSeed(salt int64, key string) int64 {
	const (
		offset64 = uint64(1469598103934665603)
		prime64  = uint64(1099511628211)
	)
	hash := offset64 ^ uint64(salt)
	for i := 0; i < len(key); i++ {
		hash ^= uint64(key[i])
		hash *= prime64
	}
	return int64(hash)
}
