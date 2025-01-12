// main.go
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime/pprof"
	"time"

	goverture "github.com/goverture/exact_cover"
)

type Choice struct {
	Row int // Chessboard row (0-N-1)
	Col int // Chessboard column (0-N-1)
}

// printBoard displays the chessboard with queens placed.
func printBoard(board [][]int) {
	for i, row := range board {
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
				fmt.Print("Q ")
			}
		}
		fmt.Println()
	}
}

// reconstructBoard translates a solution from row indices to the chessboard.
func reconstructBoard(N int, solution []int, choices [][]int, choiceToCell []Choice) ([][]int, error) {
	var board = make([][]int, N)
	for _, rowIdx := range solution {
		if rowIdx < 0 || rowIdx >= len(choiceToCell) {
			return board, fmt.Errorf("invalid choice index: %d", rowIdx)
		}
		choice := choiceToCell[rowIdx]
		board[choice.Row][choice.Col] = 1 // Place a queen
	}
	return board, nil
}

func generateChoices(testN int) ([][]int, map[int]bool) {
	// Generate the exact cover matrix and choiceToCell mapping for NTest
	choices := make([][]int, 0)

	// Total constraints:
	// Rows: 0 to NTest-1
	// Columns: NTest to 2*NTest-1
	// Major Diagonals: 2*NTest to 4*NTest-2
	// Minor Diagonals: 4*NTest-1 to 6*NTest-3
	totalConstraints := 6*testN - 2

	secondaryColumns := make(map[int]bool)
	for i := 0; i < totalConstraints; i++ {
		secondaryColumns[i] = i >= 2*testN
	}

	for row := 0; row < testN; row++ {
		for col := 0; col < testN; col++ {
			choice := make([]int, totalConstraints) // Initialize all to 0

			// Row constraint
			choice[row] = 1

			// Column constraint
			choice[testN+col] = 1

			// Major Diagonal constraint
			majorDiag := 2*testN + (row - col + testN - 1)
			choice[majorDiag] = 1

			// Minor Diagonal constraint
			minorDiag := 4*testN - 1 + (row + col)
			choice[minorDiag] = 1

			choices = append(choices, choice)
		}
	}

	return choices, secondaryColumns
}

// go build -ldflags="-s -w" -o queens
func main() {
	f, _ := os.Create("cpu.prof")
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()

	// Define a flag to read the size of the chessboard from command line arguments
	size := flag.Int("size", 8, "Size of the chessboard (N x N)")
	flag.Parse()

	fmt.Printf("Solving the %d-Queens Problem:\n", *size)

	// Generate the exact cover matrix and choiceToCell mapping
	choices, secondaryColumns := generateChoices(*size)

	sparseMatrix := goverture.SparseMatrixFromArray(choices)

	// Start timer
	start := time.Now()

	// Call SolveDLX with the exact cover matrix
	isSecondaryColumn := func(colIndex int) bool {
		ok, exists := secondaryColumns[colIndex]
		return exists && ok
	}

	solutionsChan := goverture.SolveDLXWithSecondary(context.Background(), sparseMatrix, isSecondaryColumn, -1*time.Second)

	// Collect all solutions into a slice
	solCount := 0
	for sol := range solutionsChan {
		_ = sol
		solCount++
	}

	// Stop timer and calculate duration
	duration := time.Since(start)

	// Display the number of solutions found
	fmt.Printf("\nFound %d solution(s) in %v\n", solCount, duration)
}
