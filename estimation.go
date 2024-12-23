package goverture

import (
	"fmt"
	"math/rand"
	"time"
)

func EstimateDLX(matrix [][]int, numSamples int) float64 {
	secondaryColumns := make(map[int]bool)
	for i := 0; i < len(matrix[0]); i++ {
		secondaryColumns[i] = false
	}

	sparseMatrix := SparseMatrixFromArray(matrix)

	root := BuildDLX(sparseMatrix, secondaryColumns)
	rand.Seed(time.Now().UnixNano()) // Seed the random number generator

	var totalEstimate float64 = 0
	for i := 0; i < numSamples; i++ {
		estimate := estimateRandomWalk(root, 0)
		totalEstimate += estimate
	}
	averageEstimate := totalEstimate / float64(numSamples)
	fmt.Printf("Estimated total nodes: %.0f\n", averageEstimate)
	return averageEstimate
}

func EstimateDLXWithSecondary(matrix [][]int, secondaryColumns map[int]bool, numSamples int) float64 {
	sparseMatrix := SparseMatrixFromArray(matrix)

	root := BuildDLX(sparseMatrix, secondaryColumns)
	rand.Seed(time.Now().UnixNano()) // Seed the random number generator

	var totalEstimate float64 = 0
	for i := 0; i < numSamples; i++ {
		estimate := estimateRandomWalk(root, 0)
		totalEstimate += estimate
	}
	averageEstimate := totalEstimate / float64(numSamples)
	fmt.Printf("Estimated total nodes: %.0f\n", averageEstimate)
	return averageEstimate
}

func estimateRandomWalk(root *column, depth int) float64 {
	if noPrimaryColumnsLeft(root) {
		// Reached a solution
		return 1.0
	}

	// Choose the primary column with the smallest size (heuristic)
	col := chooseColumn(root)
	if col == nil || col.S == 0 {
		// Dead end
		return 1.0
	}

	// Collect the possible choices (rows)
	var choices []*node
	for i := col.D; i != &col.node; i = i.D {
		choices = append(choices, i)
	}
	numChoices := len(choices)

	if numChoices == 0 {
		// No possible choices, dead end
		return 1.0
	}

	// Randomly select one of the possible choices
	randIndex := rand.Intn(numChoices)
	chosenNode := choices[randIndex]

	// Cover the chosen column and related columns
	Cover(col)
	for j := chosenNode.R; j != chosenNode; j = j.R {
		Cover(j.C)
	}

	// Recurse deeper into the tree
	subtreeEstimate := estimateRandomWalk(root, depth+1)

	// Uncover the columns to restore the state
	for j := chosenNode.L; j != chosenNode; j = j.L {
		Uncover(j.C)
	}
	Uncover(col)

	// Calculate the estimate for this path
	estimate := float64(numChoices) * subtreeEstimate

	return estimate
}
