package main

import (
	"context"
	"fmt"
	"time"

	goverture "github.com/goverture/exact_cover"
)

// Define the size of the Sudoku grid
const Size = 9

// Choice represents a possible placement of a number in a cell.
type Choice struct {
	Row int // Sudoku grid row (0-8)
	Col int // Sudoku grid column (0-8)
	Num int // Number to place (1-9)
}

func printGrid(grid [Size][Size]int) {
	for i, row := range grid {
		if i%3 == 0 && i != 0 {
			fmt.Println("------+-------+------")
		}
		for j, val := range row {
			if j%3 == 0 && j != 0 {
				fmt.Print("| ")
			}
			if val == 0 {
				fmt.Print(". ")
			} else {
				fmt.Printf("%d ", val)
			}
		}
		fmt.Println()
	}
}

func main() {
	fmt.Println("Initial Sudoku Grid:")
	var testGrid = [Size][Size]int{
		{0, 0, 0, 6, 7, 8, 9, 1, 2},
		{6, 7, 2, 1, 9, 5, 3, 4, 8},
		{1, 9, 8, 3, 4, 2, 5, 6, 7},
		{8, 5, 9, 7, 6, 1, 4, 2, 3},
		{4, 2, 6, 8, 5, 3, 7, 9, 1},
		{7, 1, 3, 9, 2, 0, 8, 5, 6},
		{9, 6, 1, 5, 3, 7, 2, 8, 4},
		{2, 8, 7, 4, 1, 9, 6, 3, 5},
		{0, 0, 0, 0, 0, 0, 0, 0, 0},
	}

	printGrid(testGrid)

	// Generate the constraints for the empty Sudoku grid
	// We have 729 possible choices (9x9x9) for each cell
	// and 324 constraints (9*9 for each cell, row, column and block)
	choices := make([][]int, 0)
	var choiceToCell []Choice // Mapping from choice index to (Row, Col, Num)

	for row := 0; row < 9; row++ {
		for col := 0; col < 9; col++ {
			for num := 1; num <= 9; num++ {
				if testGrid[row][col] != 0 && testGrid[row][col] != num {
					// the grid is constrained here
					continue
				}

				choice := make([]int, 4*Size*Size) // it's initialized to 0

				// set the cell constraint
				cell_index := row*Size + col
				choice[cell_index] = 1

				// set the row contraint (ie there is a 'n' in the ith row)
				row_index := 81 + row*Size + (num - 1)
				choice[row_index] = 1

				// set the column contraint (ie there is a 'n' in the jth column)
				col_index := 162 + col*Size + (num - 1)
				choice[col_index] = 1

				// set the block constraint (ie there is a 'n' in the kth block)
				block_num := (row/3)*3 + (col / 3)
				block_index := 243 + block_num*Size + (num - 1)
				choice[block_index] = 1

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

	sparseMatrix := goverture.SparseMatrixFromArray(choices)

	solutionsChan := goverture.SolveDLX(context.Background(), sparseMatrix, -1*time.Second)

	// Collect all solutions into a slice
	var solutions []goverture.Solution
	for sol := range solutionsChan {
		solutions = append(solutions, sol)
	}

	// Define the number of expected solutions
	expectedSolutionsCount := 1 // Typically, a Sudoku puzzle has a unique solution

	if len(solutions) != expectedSolutionsCount {
		fmt.Printf("Expected %d solution(s), got %d\n", expectedSolutionsCount, len(solutions))
	} else {
		fmt.Printf("Found %d solution(s)\n", len(solutions))
	}

	// slicesEqual := func(a, b []int) bool {
	// 	if len(a) != len(b) {
	// 		return false
	// 	}
	// 	for i := range a {
	// 		if a[i] != b[i] {
	// 			return false
	// 		}
	// 	}
	// 	return true
	// }

	// findRowIndex finds the index of a given row in the matrix.
	// findRowIndex := func(matrix [][]int, row []int) (int, bool) {
	// 	for i, r := range matrix {
	// 		if slicesEqual(r, row) {
	// 			return i, true
	// 		}
	// 	}
	// 	return -1, false
	// }

	// Process each solution to reconstruct and display the completed Sudoku grid
	for idx, sol := range solutions {
		// Initialize an empty Sudoku grid
		var solvedGrid [Size][Size]int

		// Start with the initial grid
		for r := 0; r < Size; r++ {
			for c := 0; c < Size; c++ {
				solvedGrid[r][c] = testGrid[r][c]
			}
		}

		// Iterate over each row in the solution
		for _, row := range sol.Matrix {
			// Find the index of this row in the exact cover matrix
			choiceIndex, found := goverture.FindRowIndex(sparseMatrix, row)
			if !found {
				fmt.Printf("Row %v not found in the matrix\n", row)
				continue
			}

			// Map the choice index to the corresponding cell and number
			choice := choiceToCell[choiceIndex]
			solvedGrid[choice.Row][choice.Col] = choice.Num
		}

		// Display the solved grid
		fmt.Printf("\nSolution %d:\n", idx+1)
		printGrid(solvedGrid)
	}
}
