package goverture

import (
	"testing"
)

func TestEstimateDLX(t *testing.T) {
	// skip it
	t.Skip("Skipping test")

	// Example usage
	matrix := [][]int{
		{1, 0, 0, 1, 0, 0, 0},
		{1, 0, 0, 1, 0, 0, 0},
		{0, 0, 0, 1, 1, 0, 1},
		{0, 0, 1, 0, 1, 1, 0},
		{0, 1, 1, 0, 0, 1, 1},
		{0, 1, 0, 0, 0, 0, 1},
	}

	res := EstimateDLX(matrix, 1)
	if res != 2 {
		t.Errorf("Expected estimate to be 2, got %f", res)
	}
}
