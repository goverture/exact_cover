package goverture

import (
	"context"
	"fmt"
	"math"
)

// -- ADDED: Type definitions for SparseRow and SparseMatrix --
type SparseRow map[int]int
type SparseMatrix []SparseRow

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

// CreateColumns creates and links all column headers horizontally
// and maintains the primary columns list
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
			col.PrimaryLeft = prevPrimary
			col.PrimaryRight = prevPrimary.PrimaryRight
			prevPrimary.PrimaryRight.PrimaryLeft = col
			prevPrimary.PrimaryRight = col
			prevPrimary = col
		} else {
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

// AddNodes adds all nodes to the Dancing Links structure based on the sparse matrix
func AddNodes(matrix SparseMatrix, columns []*column) {
	for rowIndex, row := range matrix {
		var prevNode *node
		// row is a map[int]int, so colIndex -> val
		for colIndex, val := range row {
			if val == 1 {
				col := columns[colIndex]
				newNode := &node{
					C:     col,
					RowID: rowIndex, // store which original row this belongs to
				}
				// Insert into column (vertical linkage)
				newNode.U = col.U
				newNode.D = &col.node
				col.U.D = newNode
				col.U = newNode
				col.S++

				// Link horizontally in the row
				if prevNode != nil {
					newNode.L = prevNode
					newNode.R = prevNode.R
					prevNode.R.L = newNode
					prevNode.R = newNode
				} else {
					// First node in the row points to itself
					newNode.L = newNode
					newNode.R = newNode
				}
				prevNode = newNode
			}
		}
	}
}

// BuildDLX constructs the Dancing Links structure from the sparse matrix.
// `secondaryColumns` is a map of colIndex -> bool indicating if a column is secondary.
func BuildDLX(matrix SparseMatrix, secondaryColumns map[int]bool) *column {
	root := InitializeRoot()
	if len(matrix) == 0 {
		return root // Empty matrix, return root as is
	}

	// 1) Find the largest column index across all rows in the sparse matrix
	maxColIndex := 0
	for _, row := range matrix {
		for colIndex := range row {
			if colIndex > maxColIndex {
				maxColIndex = colIndex
			}
		}
	}
	numCols := maxColIndex + 1

	// 2) Generate column names and primary status
	columnNames := make([]string, numCols)
	isPrimary := make([]bool, numCols)
	for i := 0; i < numCols; i++ {
		columnNames[i] = fmt.Sprintf("C%d", i+1)
		isPrimary[i] = !secondaryColumns[i] // false => secondary, true => primary
	}

	// 3) Create columns, then add nodes
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

// NodeVisitor is called at each depth for debugging or counting
type NodeVisitor func(depth int)

// search recursively finds all exact covers, with context for cancellation
// NOTE: We'll now send out solutions as a channel of SparseMatrix
func search(
	ctx context.Context,
	root *column,
	matrix SparseMatrix,
	solution []*node,
	solutions chan<- SparseMatrix, // channel of SparseMatrix
	depth int,
	visit NodeVisitor,
) {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return
	default:
	}

	if noPrimaryColumnsLeft(root) {
		// Found a solution
		currentSolution := make(SparseMatrix, len(solution))
		for i, nd := range solution {
			rowID := getRow(nd)
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
	// If there are no 1s left in the column, it's a dead end
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

// createNodeCounter is just a helper to track how many nodes we visit
func createNodeCounter() (NodeVisitor, *int64) {
	var totalNodes int64
	visitor := func(depth int) {
		totalNodes++
	}
	return visitor, &totalNodes
}

// SolveDLXWithSecondary initiates the DLX search with secondary columns
// and returns a channel of solutions (each a SparseMatrix).
func SolveDLXWithSecondary(ctx context.Context, matrix SparseMatrix, secondaryColumns map[int]bool) <-chan SparseMatrix {
	solutions := make(chan SparseMatrix)
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

// SolveDLX initiates the DLX search and returns a channel of solutions (each a SparseMatrix).
func SolveDLX(ctx context.Context, matrix SparseMatrix) <-chan SparseMatrix {
	solutions := make(chan SparseMatrix)
	visitor, totalNodes := createNodeCounter()

	if len(matrix) == 0 {
		close(solutions)
		return solutions
	}

	go func() {
		// Mark all columns as primary by default
		secondaryColumns := make(map[int]bool)
		// We do not know the max col index until we scan the matrix, so let's do that
		maxColIndex := 0
		for _, row := range matrix {
			for colIndex := range row {
				if colIndex > maxColIndex {
					maxColIndex = colIndex
				}
			}
		}
		for i := 0; i <= maxColIndex; i++ {
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
