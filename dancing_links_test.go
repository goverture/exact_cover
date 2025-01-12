package goverture

import (
	"context"
	"testing"
	"time"
)

func TestBuildDLXAsNeeded(t *testing.T) {
	// Example usage
	matrix := [][]int{
		{1, 0, 0, 1, 0, 0, 0},
		{1, 0, 0, 1, 0, 0, 0},
		{0, 0, 0, 1, 1, 0, 1},
		{0, 0, 1, 0, 1, 1, 0},
		{0, 1, 1, 0, 0, 1, 1},
		{0, 1, 0, 0, 0, 0, 1},
	}

	sparseMatrix := SparseMatrixFromArray(matrix)

	for i, row := range matrix {
		sparseMatrix[i] = make(SparseRow, 0)
		for j, val := range row {
			if val == 1 {
				sparseMatrix[i][j] = 1
			}
		}
	}

	secondaryColumns := make(map[int]bool)
	for i := 0; i < len(matrix[0]); i++ {
		secondaryColumns[i] = false
	}

	matrixChan := make(chan SparseRow)
	go func() {
		for _, row := range sparseMatrix {
			matrixChan <- row
		}
		close(matrixChan)
	}()

	isSecondaryColumn := func(colIndex int) bool {
		ok, exists := secondaryColumns[colIndex]
		return exists && ok
	}

	res := BuildDLXAsNeeded(matrixChan, isSecondaryColumn)
	_ = res
	// TODO: We need an actual assert here
	println("ok")
}

func TestSolveDLX(t *testing.T) {
	matrix := [][]int{
		{1, 0, 0, 1, 0, 0, 0}, // Row 0
		{0, 0, 0, 1, 1, 0, 1}, // Row 1
		{0, 0, 1, 0, 1, 1, 0}, // Row 2
		{0, 1, 1, 0, 0, 1, 1}, // Row 3
		{0, 1, 0, 0, 0, 0, 1}, // Row 4
		{1, 1, 1, 0, 0, 1, 0}, // Row 5
		{1, 1, 1, 1, 1, 1, 1}, // Row 6
	}

	sparseMatrix := SparseMatrixFromArray(matrix)

	solutionsChan := SolveDLX(context.Background(), sparseMatrix, -1*time.Second)

	// Collect all final solutions into a slice
	var solutions [][]int
	for sol := range solutionsChan {
		if sol.IsFinal {
			var solutionIndices []int
			for _, row := range sol.Matrix {
				index, found := FindRowIndex(sparseMatrix, row)
				if !found {
					t.Errorf("Row %v not found in the matrix", row)
					continue
				}
				solutionIndices = append(solutionIndices, index)
			}
			solutions = append(solutions, solutionIndices)
		}
	}

	// Define expected solutions as slices of row indices
	expectedSolutions := [][]int{
		{0, 2, 4}, // Solution 1
		{1, 5},    // Solution 2
		{6},       // Solution 3
	}

	// Check if the number of solutions matches
	if len(solutions) != len(expectedSolutions) {
		t.Errorf("Expected %d solutions, got %d", len(expectedSolutions), len(solutions))
	}

	// Create a copy of expectedSolutions to track which have been found
	expectedFound := make([]bool, len(expectedSolutions))

	// Iterate through each collected solution
	for _, sol := range solutions {
		matched := false
		for i, expectedSol := range expectedSolutions {
			if !expectedFound[i] && slicesEqual(sol, expectedSol) {
				expectedFound[i] = true
				matched = true
				break
			}
		}
		if !matched {
			t.Errorf("Unexpected solution found: %v", sol)
		}
	}

	// Check if all expected solutions were found
	for i, found := range expectedFound {
		if !found {
			t.Errorf("Expected solution %v not found", expectedSolutions[i])
		}
	}
}

func TestSolveDLX_WithEmptyMatrix(t *testing.T) {
	matrix := [][]int{}

	sparseMatrix := SparseMatrixFromArray(matrix)

	solutionsChan := SolveDLX(context.Background(), sparseMatrix, -1*time.Second)

	// Collect all final solutions into a slice
	var solutions [][]int
	for sol := range solutionsChan {
		if sol.IsFinal {
			var solutionIndices []int
			for _, row := range sol.Matrix {
				index, found := FindRowIndex(sparseMatrix, row)
				if !found {
					t.Errorf("Row %v not found in the matrix", row)
					continue
				}
				solutionIndices = append(solutionIndices, index)
			}
			solutions = append(solutions, solutionIndices)
		}
	}

	// Check if the number of solutions matches
	if len(solutions) != 0 {
		t.Errorf("Expected 0 solutions, got %d", len(solutions))
	}
}
