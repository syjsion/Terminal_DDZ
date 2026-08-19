package player

import (
	"reflect"
	"testing"

	"github.com/syjsion/Terminal_DDZ/internal/game"
)

func TestBuildViewDoesNotExposeOpponentHands(t *testing.T) {
	e := game.NewEngine(game.WithSeed(9))
	if err := e.StartRound(); err != nil {
		t.Fatal(err)
	}
	state := e.State()
	view := BuildView(state, 1)
	if len(view.OwnCards) != 17 || len(view.OtherCounts) != 2 {
		t.Fatalf("unexpected view: %+v", view)
	}
	typeOfView := reflect.TypeOf(view)
	for i := 0; i < typeOfView.NumField(); i++ {
		name := typeOfView.Field(i).Name
		if name == "Players" || name == "GameState" || name == "OpponentHands" {
			t.Fatalf("unsafe field exposed: %s", name)
		}
	}
	if len(view.BottomPublic) != 0 {
		t.Fatal("bottom cards exposed during bidding")
	}
	landlord := state.CurrentSeat
	if err := e.ApplyBid(landlord, 3); err != nil {
		t.Fatal(err)
	}
	view = BuildView(e.State(), 1)
	if len(view.BottomPublic) != 3 {
		t.Fatal("public bottom cards missing after bidding")
	}
}
