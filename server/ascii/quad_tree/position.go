package quad_tree

func TranslateTo1D(row, col, cols int) int {
	return row*cols + col
}
