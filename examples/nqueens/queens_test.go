package main

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"sort"
	"testing"
	"time"

	goverture "github.com/goverture/exact_cover"
)

// canonicalSolution returns a slice of string-ified row-column lists.
// Example: a solution with rows {0:1,5:1} and {1:1,7:1} becomes ["[0 5]", "[1 7]"] (and then sorted).
func canonicalSolution(sol goverture.Solution) []string {
	matrix := sol.Matrix
	rows := make([]string, 0, len(matrix))
	for _, sparseRow := range matrix {
		// Collect column indices in sorted order
		cols := make([]int, 0, len(sparseRow))
		for c := range sparseRow {
			cols = append(cols, c)
		}
		sort.Ints(cols)
		// Turn the list of column indices into a string representation
		rows = append(rows, fmt.Sprintf("%v", cols))
	}
	// Sort the row-representations so row order doesn't matter
	sort.Strings(rows)
	return rows
}

// canonicalSolutions turns a list of solutions into a list of canonical forms, then sorts it.
func canonicalSolutions(solutions []goverture.Solution) [][]string {
	canons := make([][]string, len(solutions))
	for i, sol := range solutions {
		canons[i] = canonicalSolution(sol)
	}
	// Sort the solutions themselves (outer slice) so solution order doesn't matter
	sort.Slice(canons, func(i, j int) bool {
		// Compare canons[i] vs canons[j] lexicographically
		si, sj := canons[i], canons[j]
		// Compare lengths first
		if len(si) != len(sj) {
			return len(si) < len(sj)
		}
		// Compare row-by-row
		for idx := range si {
			if si[idx] < sj[idx] {
				return true
			} else if si[idx] > sj[idx] {
				return false
			}
		}
		return false
	})
	return canons
}

func TestNQueensSolver_small(t *testing.T) {
	// N=4 should have 2 solutions
	testN := 4
	expectedSolutionCount := 2

	// Your function that builds the matrix & secondary columns:
	choices, secondaryColumns := generateChoices(testN)
	sparseMatrix := goverture.SparseMatrixFromArray(choices)

	// Solve with DLX
	isSecondaryColumn := func(colIndex int) bool {
		ok, exists := secondaryColumns[colIndex]
		return exists && ok
	}
	solutionsChan := goverture.SolveDLXWithSecondary(context.Background(), sparseMatrix, isSecondaryColumn, -1*time.Second)

	// Collect solutions into a slice
	var solutions []goverture.Solution
	for sol := range solutionsChan {
		solutions = append(solutions, sol)
	}

	// We *expect* two solutions, which we originally identified by row indices:
	expectedSolutionsIndices := [][]int{
		{1, 7, 8, 14},  // solution #1
		{2, 4, 11, 13}, // solution #2
	}
	var expectedSolutions []goverture.Solution
	for _, rowIndices := range expectedSolutionsIndices {
		var sol goverture.SparseMatrix
		for _, idx := range rowIndices {
			sol = append(sol, sparseMatrix[idx])
		}
		expectedSolutions = append(
			expectedSolutions,
			goverture.Solution{
				IsFinal: true,
				Matrix:  sol,
			},
		)
	}

	// Quick sanity check: we expect 2 solutions
	if len(solutions) != expectedSolutionCount {
		t.Errorf("Expected %d solutions, got %d", expectedSolutionCount, len(solutions))
	}

	// Convert both actual and expected solutions to canonical forms
	gotCanon := canonicalSolutions(solutions)
	expCanon := canonicalSolutions(expectedSolutions)

	// Compare them as sets (really, sorted slices of sorted rows)
	if !reflect.DeepEqual(gotCanon, expCanon) {
		t.Errorf("Solutions mismatch!\nGot: %#v\nExpected: %#v", gotCanon, expCanon)
	}
}

// TestNQueensSolver_LargeN tests the N-Queens solver for multiple values of N, including large N.
func TestNQueensSolver_LargeN(t *testing.T) {
	// Define a slice of test cases
	testCases := []struct {
		N                     int
		ExpectedSolutionCount uint64
	}{
		{N: 4, ExpectedSolutionCount: 2},
		{N: 14, ExpectedSolutionCount: 365596},
		// {N: 20, ExpectedSolutionCount: 39029188884},
	}

	// Define a maximum limit for counting solutions to prevent excessive computation
	const maxSolutions uint64 = 1000000

	for _, tc := range testCases {
		// Generate the exact cover matrix and choiceToCell mapping for the current N
		choices, secondaryColumns := generateChoices(tc.N)

		sparseMatrix := goverture.SparseMatrixFromArray(choices)

		// Call SolveDLXWithSecondary with the exact cover matrix
		isSecondaryColumn := func(colIndex int) bool {
			ok, exists := secondaryColumns[colIndex]
			return exists && ok
		}
		solutionsChan := goverture.SolveDLXWithSecondary(context.Background(), sparseMatrix, isSecondaryColumn, -1*time.Second)

		// Initialize a counter for solutions
		var solCount uint64 = 0

		// Iterate over the solutions channel and count the solutions
		for range solutionsChan {
			solCount++
			// Early termination to prevent excessive computation for large N
			if solCount > tc.ExpectedSolutionCount {
				break
			}
		}

		// Check if the number of solutions matches the expected count
		if solCount != tc.ExpectedSolutionCount {
			if tc.N == 20 && solCount > tc.ExpectedSolutionCount {
				t.Errorf("For N=%d, solution count exceeded expected count of %d", tc.N, tc.ExpectedSolutionCount)
			} else {
				t.Errorf("For N=%d, expected %d solutions, got %d", tc.N, tc.ExpectedSolutionCount, solCount)
			}
		}
	}
}

func TestNQueensSolver_ComplexityEstimate(t *testing.T) {
	t.Skip("Skipping cancellation test, should be a range")

	// Define a slice of test cases
	// testCases := []struct {
	// 	N                  int
	// 	ExpectedComplexity float64
	// }{
	// 	{N: 4, ExpectedComplexity: 6},
	// 	{N: 14, ExpectedComplexity: 2332361},
	// 	{N: 20, ExpectedComplexity: 182540060494},
	// }

	// for _, tc := range testCases {
	// 	// Generate the exact cover matrix and choiceToCell mapping for the current N
	// 	choices, secondaryColumns := generateChoices(tc.N)


	// 	// Check if the estimated complexity matches the expected value
	// 	if estimatedComplexity != tc.ExpectedComplexity {
	// 		t.Errorf("For N=%d, expected complexity %.0f, got %.0f", tc.N, tc.ExpectedComplexity, estimatedComplexity)
	// 	}
	// }
}

func TestCancellation(t *testing.T) {
	N := 20 // Large N to demonstrate cancellation
	choices, secondaryColumns := generateChoices(N)

	// Create a cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure resources are cleaned up

	// Start the solver in a goroutine
	sparseMatrix := goverture.SparseMatrixFromArray(choices)

	isSecondaryColumn := func(colIndex int) bool {
		ok, exists := secondaryColumns[colIndex]
		return exists && ok
	}
	solutionsChan := goverture.SolveDLXWithSecondary(ctx, sparseMatrix, isSecondaryColumn, -1*time.Second)

	// Initialize a counter for solutions
	var solCount uint64 = 0

	// Create a channel to signal when to cancel
	cancelChan := make(chan struct{})

	// Start a goroutine to cancel after counting a few solutions
	go func() {
		time.Sleep(2 * time.Second) // Wait for 2 seconds before cancelling
		cancel()
		close(cancelChan)
	}()

	// Iterate over the solutions channel and count the solutions
	for {
		select {
		case _, ok := <-solutionsChan:
			if !ok {
				// Channel closed
				break
			}
			solCount++
		case <-cancelChan:
			// Cancellation signal received
			log.Printf("Cancellation signal received after finding %d solutions", solCount)
			return
		}
	}
}

func TestTimeoutCancellation(t *testing.T) {
	N := 20 // Large N to demonstrate timeout cancellation
	choices, secondaryColumns := generateChoices(N)

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel() // Ensure resources are cleaned up

	sparseMatrix := goverture.SparseMatrixFromArray(choices)

	// Start the solver in a goroutine
	isSecondaryColumn := func(colIndex int) bool {
		ok, exists := secondaryColumns[colIndex]
		return exists && ok
	}
	solutionsChan := goverture.SolveDLXWithSecondary(ctx, sparseMatrix, isSecondaryColumn, -1*time.Second)

	// Initialize a counter for solutions
	var solCount uint64 = 0

	// Iterate over the solutions channel and count the solutions
	for range solutionsChan {
		solCount++
	}

	// Check if the context deadline was exceeded
	if ctx.Err() == context.DeadlineExceeded {
		t.Logf("Timeout reached after finding %d solutions", solCount)
	} else {
		t.Errorf("Test completed without timeout, found %d solutions", solCount)
	}
}
