package engine

import "testing"

func TestWinningDirectionsAndRowBoundaries(t *testing.T) {
	for _, geometry := range []struct{ width, height, target int }{{3, 3, 3}, {7, 6, 4}} {
		for _, direction := range [][2]int{{0, 1}, {1, 0}, {1, 1}, {1, -1}} {
			board := make([]int, geometry.width*geometry.height)
			column := 0
			if direction[1] < 0 {
				column = geometry.width - 1
			}
			for n := 0; n < geometry.target; n++ {
				board[n*direction[0]*geometry.width+column+n*direction[1]] = 1
			}
			if len(WinningLine(board, geometry.width, geometry.height, geometry.target, 1)) != geometry.target {
				t.Fatal("missed a winning direction")
			}
		}
		board := make([]int, geometry.width*geometry.height)
		for n := 0; n < geometry.target; n++ {
			board[geometry.width-1+n] = 1
		}
		if len(WinningLine(board, geometry.width, geometry.height, geometry.target, 1)) != 0 {
			t.Fatal("line wrapped across a row boundary")
		}
	}
}
