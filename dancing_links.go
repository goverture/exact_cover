package goverture

import (
	"context"
	"fmt"
	"math"
)

// node represents each '1' in the matrix
type node struct {
	C          *column // Column header
	RowID      int     // Identifier for the original row
	L, R, U, D *node   // Left, Right, Up, Down pointers
}

// Column represents the column headers
type column struct {
	node                 // Embedding node
	S            int     // Size: number of 1s in the column
	N            string  // Name of the column
	IsPrimary    bool    // Indicates if the column is primary (true) or secondary (false)
	PrimaryLeft  *column // Left pointer in primary columns list
	PrimaryRight *column // Right pointer in primary columns list
}

type NodeVisitor func(depth int)

// InitializeRoot creates and initializes the root header
func InitializeRoot() *column {
	root := &column{
		N: "root",
	}
	// Initialize root's embedded node pointers to point to itself
	root.L = &root.node
	root.R = &root.node
	root.U = &root.node
	root.D = &root.node
	// Initialize root's primary columns pointers to point to itself
	root.PrimaryLeft = root
	root.PrimaryRight = root

	return root
}

// CreateColumns creates and links all column headers horizontally and maintains the primary columns list
func CreateColumns(root *column, columnNames []string, isPrimary []bool) []*column {
	var prevColumn *column
	columns := make([]*column, 0, len(columnNames))
	var prevPrimary *column = root

	for idx, name := range columnNames {
		col := &column{
			N:         name,
			IsPrimary: isPrimary[idx],
		}
		// Initialize the column's embedded node pointers to point to itself
		col.U = &col.node
		col.D = &col.node
		col.C = col
		col.S = 0

		columns = append(columns, col)

		// Link horizontally to form the header list
		if prevColumn != nil {
			col.L = &prevColumn.node
			prevColumn.R = &col.node
		} else {
			// First column, link to root
			col.L = &root.node
			root.R = &col.node
		}

		prevColumn = col

		// Also link into the primary columns list if primary
		if col.IsPrimary {
			// Link into primary columns list
			col.PrimaryLeft = prevPrimary
			col.PrimaryRight = prevPrimary.PrimaryRight
			prevPrimary.PrimaryRight.PrimaryLeft = col
			prevPrimary.PrimaryRight = col
			prevPrimary = col
		} else {
			// Non-primary columns do not go into the primary columns list
			col.PrimaryLeft = nil
			col.PrimaryRight = nil
		}
	}

	// Complete the circular linkage by linking the last column back to root
	if prevColumn != nil {
		prevColumn.R = &root.node
		root.L = &prevColumn.node
	}

	return columns
}

// AddNodes adds all nodes to the Dancing Links structure based on the matrix
func AddNodes(matrix [][]int, columns []*column) {
	for rowIndex, row := range matrix {
		var prevNode *node
		for j, val := range row {
			if val == 1 {
				col := columns[j]
				node := &node{
					C:     col,
					RowID: rowIndex, // Assign the row index here
				}
				// Insert into column (vertical linkage)
				node.U = col.U
				node.D = &col.node
				col.U.D = node
				col.U = node
				col.S++

				// Link horizontally in the row
				if prevNode != nil {
					node.L = prevNode
					node.R = prevNode.R
					prevNode.R.L = node
					prevNode.R = node
				} else {
					// First node in the row points to itself
					node.L = node
					node.R = node
				}
				prevNode = node
			}
		}
	}
}

// BuildDLX constructs the Dancing Links structure from the exact cover matrix
func BuildDLX(matrix [][]int, secondaryColumns map[int]bool) *column {
	root := InitializeRoot()
	if len(matrix) == 0 {
		return root // Empty matrix, return root as is
	}
	numCols := len(matrix[0])

	// Generate column names and primary status
	columnNames := make([]string, numCols)
	isPrimary := make([]bool, numCols)
	for i := 0; i < numCols; i++ {
		columnNames[i] = fmt.Sprintf("C%d", i+1)
		isPrimary[i] = !secondaryColumns[i]
	}

	columns := CreateColumns(root, columnNames, isPrimary)
	AddNodes(matrix, columns)
	return root
}

// Cover removes a column from the header list and primary columns list
func Cover(col *column) {
	// If the column is primary, remove it from the primary columns list
	if col.IsPrimary {
		col.PrimaryRight.PrimaryLeft = col.PrimaryLeft
		col.PrimaryLeft.PrimaryRight = col.PrimaryRight
	}

	// Remove the column header from the header list
	col.R.L = col.L
	col.L.R = col.R

	// Iterate through each node in the column
	for i := col.D; i != &col.node; i = i.D {
		// Remove the node's row from other columns
		for j := i.R; j != i; j = j.R {
			j.D.U = j.U
			j.U.D = j.D
			j.C.S--
		}
	}
}

// Uncover restores a previously covered column and updates the primary columns list
func Uncover(col *column) {
	// Iterate through each node in the column in reverse
	for i := col.U; i != &col.node; i = i.U {
		// Restore the node's row to other columns
		for j := i.L; j != i; j = j.L {
			j.C.S++
			j.D.U = j
			j.U.D = j
		}
	}

	// Restore the column header to the header list
	col.R.L = &col.node
	col.L.R = &col.node

	// If the column is primary, restore it to the primary columns list
	if col.IsPrimary {
		col.PrimaryRight.PrimaryLeft = col
		col.PrimaryLeft.PrimaryRight = col
	}
}

// chooseColumn selects the primary column with the smallest size (fewest 1s)
func chooseColumn(root *column) *column {
	minSize := math.MaxInt64
	var chosen *column
	for col := root.PrimaryRight; col != root; col = col.PrimaryRight {
		if col.S < minSize {
			minSize = col.S
			chosen = col
			if minSize == 0 {
				break // Can't get smaller than 0
			}
		}
	}
	return chosen
}

// getRow extracts the RowID from a node
func getRow(node *node) int {
	return node.RowID
}

// noPrimaryColumnsLeft checks if there are any primary columns left
func noPrimaryColumnsLeft(root *column) bool {
	return root.PrimaryRight == root
}

// search recursively finds all exact covers, with context for cancellation
func search(ctx context.Context, root *column, matrix [][]int, solution []*node, solutions chan<- [][]int, depth int, visit NodeVisitor) {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return
	default:
	}

	if noPrimaryColumnsLeft(root) {
		// Found a solution
		currentSolution := make([][]int, len(solution))
		for i, node := range solution {
			rowID := getRow(node)
			currentSolution[i] = matrix[rowID]
		}
		// Attempt to send the solution, respecting context cancellation
		select {
		case solutions <- currentSolution:
		case <-ctx.Done():
			return
		}
		return
	}

	// Choose the primary column with the smallest size (heuristic)
	col := chooseColumn(root)
	// If there are no 1s left in the column, it's a dead end, return early
	if col == nil || col.S == 0 {
		return
	}

	if visit != nil {
		visit(depth)
	}

	// Cover the chosen column
	Cover(col)

	// Iterate through each row in the column
	for i := col.D; i != &col.node; i = i.D {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			Uncover(col)
			return
		default:
		}
		// Add the row to the current solution
		solution = append(solution, i)

		// Cover all columns for each node in the row
		for j := i.R; j != i; j = j.R {
			Cover(j.C)
		}

		// Recurse with increased depth
		search(ctx, root, matrix, solution, solutions, depth+1, visit)

		// Backtrack: remove the row from the current solution
		solution = solution[:len(solution)-1]

		// Uncover all columns for each node in the row in reverse order
		for j := i.L; j != i; j = j.L {
			Uncover(j.C)
		}

		// Check for context cancellation
		select {
		case <-ctx.Done():
			Uncover(col)
			return
		default:
		}
	}

	// Uncover the chosen column
	Uncover(col)
}

func createNodeCounter() (NodeVisitor, *int64) {
	var totalNodes int64 = 0
	visitor := func(depth int) {
		totalNodes++
	}
	return visitor, &totalNodes
}

// SolveDLXWithSecondary initiates the DLX search with secondary columns and returns a channel of solutions
func SolveDLXWithSecondary(ctx context.Context, matrix [][]int, secondaryColumns map[int]bool) <-chan [][]int {
	solutions := make(chan [][]int)
	visitor, totalNodes := createNodeCounter()

	go func() {
		root := BuildDLX(matrix, secondaryColumns)
		var solution []*node
		search(ctx, root, matrix, solution, solutions, 0, visitor) // Start with depth 0
		fmt.Printf("Total nodes visited: %d\n", *totalNodes)
		close(solutions)
	}()
	return solutions
}

// SolveDLX initiates the DLX search and returns a channel of solutions
func SolveDLX(ctx context.Context, matrix [][]int) <-chan [][]int {
	solutions := make(chan [][]int)
	visitor, totalNodes := createNodeCounter()

	go func() {
		secondaryColumns := make(map[int]bool)
		for i := 0; i < len(matrix[0]); i++ {
			secondaryColumns[i] = false
		}
		root := BuildDLX(matrix, secondaryColumns)
		var solution []*node
		search(ctx, root, matrix, solution, solutions, 0, visitor) // Start with depth 0
		fmt.Printf("Total nodes visited: %d\n", *totalNodes)
		close(solutions)
	}()
	return solutions
}
