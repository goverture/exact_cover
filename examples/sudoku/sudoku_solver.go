package main

import (
	"fmt"

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
		{5, 3, 0, 0, 7, 0, 0, 0, 0},
		{6, 0, 0, 1, 9, 5, 0, 0, 0},
		{0, 9, 8, 0, 0, 0, 0, 6, 0},
		{8, 0, 0, 0, 6, 0, 0, 0, 3},
		{4, 0, 0, 8, 0, 3, 0, 0, 1},
		{7, 0, 0, 0, 2, 0, 0, 0, 6},
		{0, 6, 0, 0, 0, 0, 2, 8, 0},
		{0, 0, 0, 4, 1, 9, 0, 0, 5},
		{0, 0, 0, 0, 8, 0, 0, 7, 9},
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

	columns, nodes := goverture.BuildDLX(choices)
	visitor := func(solution []int) {
		optionIndex := make([]int, len(solution))
		for i, x := range solution {
			for nodes[x].Top > 0 {
				x = x - 1
			}
			optionIndex[i] = -nodes[x].Top
		}
		fmt.Println("Solution found : ")

		// Initialize an empty Sudoku grid
		var solvedGrid [Size][Size]int

		// Start with the initial grid
		for r := 0; r < Size; r++ {
			for c := 0; c < Size; c++ {
				solvedGrid[r][c] = testGrid[r][c]
			}
		}

		// Iterate over each row in the solution
		for _, i := range optionIndex {
			// Map the choice index to the corresponding cell and number
			choice := choiceToCell[i]
			solvedGrid[choice.Row][choice.Col] = choice.Num
		}

		// Display the solved grid
		printGrid(solvedGrid)
		fmt.Println("--------")
	}

	goverture.SolveExactCover(columns, nodes, []int{}, visitor)
}
