package games

import (
	"testing"

	"github.com/Kirill555dg/mini-games/internal/tictactoe"
)

func TestClassicCreationOffersOnlyThreeByThree(t *testing.T) {
	spec, ok := Find("tictactoe")
	if !ok || len(spec.Sizes) != 1 || spec.Sizes[0] != tictactoe.Size {
		t.Fatal("classic catalog must offer only the short 3x3 game")
	}
	for _, size := range []int{0, tictactoe.Size} {
		g, err := New(spec.ID, size, spec.MinPlayers)
		if err != nil || g.View(-1).Width != tictactoe.Size {
			t.Fatal("classic default unavailable")
		}
	}
	for _, size := range []int{-1, 1, 2, 4, 5, 6, 7, 99} {
		if _, err := New(spec.ID, size, spec.MinPlayers); err == nil {
			t.Fatalf("unsupported board %d can still be created", size)
		}
	}
}
