package main

import (
	"context"
	"log"
	"reflect"
	"testing"
	"time"

	goverture "github.com/goverture/exact_cover"
)

// TestNQueensSolver tests the SolveDLX function with the 4-Queens problem.
func TestNQueensSolver_small(t *testing.T) {
	// Define N for testing
	testN := 4

	// Define expected number of solutions for N=4 (which is 2)
	expectedSolutionCount := 2

	// Generate the exact cover matrix and choiceToCell mapping for NTest
	choices, secondaryColumns := generateChoices(testN)

	// Call SolveDLX with the exact cover matrix
	solutionsChan := goverture.SolveDLXWithSecondary(context.Background(), choices, secondaryColumns)

	// Collect all solutions into a slice
	var solutions [][][]int
	for sol := range solutionsChan {
		solutions = append(solutions, sol)
	}

	// Define expected solutions as slices of row indices
	// For N=4, the two solutions correspond to the following queen placements:
	// Solution 1: (0,1), (1,3), (2,0), (3,2)
	// Solution 2: (0,2), (1,0), (2,3), (3,1)
	expectedSolutionsIndices := [][]int{
		{1, 7, 8, 14}, // These indices need to match the exact cover matrix
		{2, 4, 11, 13},
	}
	expectedSolutions := make([][][]int, 0)
	for _, indices := range expectedSolutionsIndices {
		solution := make([][]int, 0)
		for _, index := range indices {
			solution = append(solution, choices[index])
		}
		expectedSolutions = append(expectedSolutions, solution)
	}

	// Check if the number of solutions matches
	if len(solutions) != expectedSolutionCount {
		t.Errorf("Expected %d solutions, got %d", expectedSolutionCount, len(solutions))
	}

	// Verify each solution
	for _, sol := range solutions {
		matched := false
		for _, expectedSol := range expectedSolutions {
			if reflect.DeepEqual(sol, expectedSol) {
				matched = true
				break
			}
		}
		if !matched {
			t.Errorf("Unexpected solution found: %v", sol)
		}
	}

	// Ensure all expected solutions were found
	for _, expectedSol := range expectedSolutions {
		found := false
		for _, sol := range solutions {
			if reflect.DeepEqual(sol, expectedSol) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected solution %v not found", expectedSol)
		}
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

		// Call SolveDLXWithSecondary with the exact cover matrix
		solutionsChan := goverture.SolveDLXWithSecondary(context.Background(), choices, secondaryColumns)

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
	testCases := []struct {
		N                  int
		ExpectedComplexity float64
	}{
		{N: 4, ExpectedComplexity: 6},
		{N: 14, ExpectedComplexity: 2332361},
		{N: 20, ExpectedComplexity: 182540060494},
	}

	for _, tc := range testCases {
		// Generate the exact cover matrix and choiceToCell mapping for the current N
		choices, secondaryColumns := generateChoices(tc.N)

		// Estimate the complexity of the DLX algorithm
		estimatedComplexity := goverture.EstimateDLXWithSecondary(choices, secondaryColumns, 1000)

		// Check if the estimated complexity matches the expected value
		if estimatedComplexity != tc.ExpectedComplexity {
			t.Errorf("For N=%d, expected complexity %.0f, got %.0f", tc.N, tc.ExpectedComplexity, estimatedComplexity)
		}
	}
}

func TestCancellation(t *testing.T) {
	N := 20 // Large N to demonstrate cancellation
	choices, secondaryColumns := generateChoices(N)

	// Create a cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure resources are cleaned up

	// Start the solver in a goroutine
	solutionsChan := goverture.SolveDLXWithSecondary(ctx, choices, secondaryColumns)

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

	t.Errorf("Test completed without cancellation, found %d solutions", solCount)
}

func TestTimeoutCancellation(t *testing.T) {
	N := 20 // Large N to demonstrate timeout cancellation
	choices, secondaryColumns := generateChoices(N)

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel() // Ensure resources are cleaned up

	// Start the solver in a goroutine
	solutionsChan := goverture.SolveDLXWithSecondary(ctx, choices, secondaryColumns)

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
