package ai

import (
	"context"

	"github.com/syjsion/Terminal_DDZ/internal/game"
	"github.com/syjsion/Terminal_DDZ/internal/player"
)

type Agent interface {
	ChooseBid(ctx context.Context, view player.PlayerView, legalBids []int) (int, error)
	ChooseMove(ctx context.Context, view player.PlayerView, legalMoves []game.Move) (int, error)
}
