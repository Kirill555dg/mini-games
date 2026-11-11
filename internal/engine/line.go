package engine

// WinningLine uses coordinates so horizontal and diagonal lines cannot wrap.
func WinningLine(board []int, width, height, length, mark int) []int {
	for row := 0; row < height; row++ {
		for col := 0; col < width; col++ {
			if board[row*width+col] != mark {
				continue
			}
			for _, d := range [][2]int{{0, 1}, {1, 0}, {1, 1}, {1, -1}} {
				er, ec := row+(length-1)*d[0], col+(length-1)*d[1]
				if er < 0 || er >= height || ec < 0 || ec >= width {
					continue
				}
				line := make([]int, 0, length)
				for n := 0; n < length; n++ {
					i := (row+n*d[0])*width + col + n*d[1]
					if board[i] != mark {
						break
					}
					line = append(line, i)
				}
				if len(line) == length {
					return line
				}
			}
		}
	}
	return nil
}
