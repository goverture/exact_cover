package main

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	goverture "github.com/goverture/exact_cover"
)

// Helper function to compare two Sudoku grids for equality.
func gridsEqual(a, b [Size][Size]int) bool {
	return reflect.DeepEqual(a, b)
}

// Helper function to reconstruct a Sudoku grid from a solution.
func reconstructGrid(solution [][]int, choices [][]int, choiceToCell []Choice) ([Size][Size]int, error) {
	var solvedGrid [Size][Size]int
	for _, row := range solution {
		choiceIndex, found := goverture.FindRowIndex(choices, row)
		if !found {
			return solvedGrid, fmt.Errorf("row %v not found in the exact cover matrix", row)
		}
		choice := choiceToCell[choiceIndex]
		solvedGrid[choice.Row][choice.Col] = choice.Num
	}
	return solvedGrid, nil
}

func TestSudokuSolver(t *testing.T) {
	// Define the initial Sudoku grid with some cells filled (0 represents empty cells)
	var testGrid = [Size][Size]int{
		{5, 3, 4, 6, 7, 8, 9, 1, 2},
		{6, 7, 2, 1, 9, 5, 3, 4, 8},
		{1, 9, 8, 3, 4, 2, 5, 6, 7},
		{8, 5, 9, 7, 6, 1, 4, 2, 3},
		{4, 2, 6, 8, 5, 3, 7, 9, 1},
		{7, 1, 3, 9, 2, 4, 8, 5, 6},
		{9, 6, 1, 5, 3, 7, 2, 8, 4},
		{2, 8, 7, 4, 1, 9, 6, 3, 5},
		{3, 4, 5, 2, 8, 6, 1, 7, 9},
	}

	// Define the expected solution(s)
	expectedSolutions := [][Size][Size]int{
		{
			{5, 3, 4, 6, 7, 8, 9, 1, 2},
			{6, 7, 2, 1, 9, 5, 3, 4, 8},
			{1, 9, 8, 3, 4, 2, 5, 6, 7},
			{8, 5, 9, 7, 6, 1, 4, 2, 3},
			{4, 2, 6, 8, 5, 3, 7, 9, 1},
			{7, 1, 3, 9, 2, 4, 8, 5, 6},
			{9, 6, 1, 5, 3, 7, 2, 8, 4},
			{2, 8, 7, 4, 1, 9, 6, 3, 5},
			{3, 4, 5, 2, 8, 6, 1, 7, 9},
		},
	}

	// Generate the exact cover matrix and choiceToCell mapping
	choices := make([][]int, 0)
	var choiceToCell []Choice

	for row := 0; row < Size; row++ {
		for col := 0; col < Size; col++ {
			for num := 1; num <= Size; num++ {
				if testGrid[row][col] != 0 && testGrid[row][col] != num {
					// The grid is constrained here
					continue
				}

				choice := make([]int, 4*Size*Size) // Initialized to 0

				// Set the cell constraint
				cellIndex := row*Size + col
				choice[cellIndex] = 1

				// Set the row constraint (there is a 'num' in the ith row)
				rowIndex := 81 + row*Size + (num - 1)
				choice[rowIndex] = 1

				// Set the column constraint (there is a 'num' in the jth column)
				colIndex := 162 + col*Size + (num - 1)
				choice[colIndex] = 1

				// Set the block constraint (there is a 'num' in the kth block)
				blockNum := (row/3)*3 + (col / 3)
				blockIndex := 243 + blockNum*Size + (num - 1)
				choice[blockIndex] = 1

				choices = append(choices, choice)

				// Map this choice to its corresponding cell and number
				choiceToCell = append(choiceToCell, Choice{
					Row: row,
					Col: col,
					Num: num,
				})
			}
		}
	}

	// Run the solver
	solutionsChan := goverture.SolveDLX(context.Background(), choices)

	// Collect all solutions into a slice
	var solutions [][][]int
	for sol := range solutionsChan {
		solutions = append(solutions, sol)
	}

	// Check if the number of solutions matches the expected number
	if len(solutions) != len(expectedSolutions) {
		t.Errorf("Expected %d solution(s), got %d", len(expectedSolutions), len(solutions))
	}

	// Reconstruct each solution grid and compare with expected solutions
	for _, sol := range solutions {
		// Reconstruct the grid from the solution
		reconstructedGrid, err := reconstructGrid(sol, choices, choiceToCell)
		if err != nil {
			t.Errorf("Error reconstructing grid: %v", err)
			continue
		}

		// Check if the reconstructed grid matches any expected solution
		matched := false
		for _, expected := range expectedSolutions {
			if gridsEqual(reconstructedGrid, expected) {
				matched = true
				break
			}
		}

		if !matched {
			t.Errorf("Found unexpected solution:\n%v", reconstructedGrid)
		}
	}

	// Additionally, ensure that all expected solutions were found
	for _, expected := range expectedSolutions {
		found := false
		for _, sol := range solutions {
			reconstructedGrid, err := reconstructGrid(sol, choices, choiceToCell)
			if err != nil {
				continue
			}
			if gridsEqual(reconstructedGrid, expected) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected solution not found:\n%v", expected)
		}
	}
}
