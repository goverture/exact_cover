package goverture

import (
	"context"
	"fmt"
	"math"
	"sort"
)

// -- ADDED: Type definitions for SparseRow and SparseMatrix --
type SparseRow map[int]int
type SparseMatrix []SparseRow

// node represents each '1' in the matrix
type node struct {
	C          *column // Column header
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

	Index int // In order to rebuild the original matrix row
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

// This helper ensures each column is created once and returns the *column.
func getOrCreateColumn(colIndex int, columnsMap map[int]*column, secondaryColumns map[int]bool) *column {
	if c, ok := columnsMap[colIndex]; ok {
		return c
	}
	// Create a new column header
	c := &column{
		N:         fmt.Sprintf("C%d", colIndex+1),
		IsPrimary: !secondaryColumns[colIndex], // false => secondary, true => primary
		Index:     colIndex,
	}
	// Point its own Up/Down to itself (isolated vertical ring)
	c.U = &c.node
	c.D = &c.node
	c.C = c
	columnsMap[colIndex] = c
	return c
}

// BuildDLXAsNeeded constructs the Dancing Links structure from the sparse matrix
// by creating columns lazily (on-demand) as rows are processed.
func BuildDLXAsNeeded(matrixChan <-chan SparseRow, secondaryColumns map[int]bool) *column {
	// 1) Create the root header
	root := InitializeRoot()

	// 2) Keep a map from colIndex -> *column
	columnsMap := make(map[int]*column)

	// 3) Build all nodes while reading the rows
	for sparseRow := range matrixChan {
		var firstNodeInRow *node // We'll link row nodes circularly
		var lastNodeInRow *node

		for colIndex, val := range sparseRow {
			if val == 1 {
				col := getOrCreateColumn(colIndex, columnsMap, secondaryColumns)
				// Create the node
				newNode := &node{C: col}

				// -- Vertical insertion into the column --
				// Link into the column ring (at the bottom)
				newNode.U = col.U
				newNode.D = &col.node
				col.U.D = newNode
				col.U = newNode
				col.S++

				// -- Horizontal insertion in the row --
				if firstNodeInRow == nil {
					// First node in this row
					firstNodeInRow = newNode
					lastNodeInRow = newNode
					// Row is circular, so point to itself
					newNode.L = newNode
					newNode.R = newNode
				} else {
					// Insert to the right of lastNodeInRow
					newNode.L = lastNodeInRow
					newNode.R = lastNodeInRow.R
					lastNodeInRow.R.L = newNode
					lastNodeInRow.R = newNode
					lastNodeInRow = newNode
				}
			}
		}
	}

	// 4) Now link all columns to each other (and to the root) in sorted order of colIndex
	//    This ensures a stable left-right traversal
	type colWithIndex struct {
		index  int
		column *column
	}
	colList := make([]colWithIndex, 0, len(columnsMap))
	for idx, c := range columnsMap {
		colList = append(colList, colWithIndex{index: idx, column: c})
	}
	// Sort by column index ascending
	sort.Slice(colList, func(i, j int) bool {
		return colList[i].index < colList[j].index
	})

	var prevCol *column = nil
	for _, cwi := range colList {
		col := cwi.column
		if prevCol == nil {
			// First actual column links to the root
			col.L = &root.node
			root.R = &col.node
		} else {
			col.L = &prevCol.node
			prevCol.R = &col.node
		}
		prevCol = col

		// Also link into the primary columns list if needed
		if col.IsPrimary {
			// Insert into the primary columns circular list to the right of `root`
			col.PrimaryLeft = root
			col.PrimaryRight = root.PrimaryRight
			root.PrimaryRight.PrimaryLeft = col
			root.PrimaryRight = col
		}
	}

	// Complete circular linkage from last column back to root
	if prevCol != nil {
		prevCol.R = &root.node
		root.L = &prevCol.node
	}

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

func RebuildRowFromNode(n *node) SparseRow {
	row := make(SparseRow)
	current := n

	for {
		row[current.C.Index] = 1
		current = current.R
		if current == n {
			break
		}
	}
	return row
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
		if len(solution) == 0 {
			return
		}

		// Found a solution
		currentSolution := make(SparseMatrix, len(solution))
		for i, nd := range solution {
			currentSolution[i] = RebuildRowFromNode(nd)
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
		search(ctx, root, solution, solutions, depth+1, visit)

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
		matrixChan := make(chan SparseRow)
		go func() {
			for _, row := range matrix {
				matrixChan <- row
			}
			close(matrixChan)
		}()

		// root := BuildDLX(matrix, secondaryColumns)
		root := BuildDLXAsNeeded(matrixChan, secondaryColumns)
		var solution []*node
		search(ctx, root, solution, solutions, 0, visitor) // Start with depth 0
		fmt.Printf("Total nodes visited: %d\n", *totalNodes)
		close(solutions)
	}()
	return solutions
}

// SolveDLX initiates the DLX search and returns a channel of solutions (each a SparseMatrix).
func SolveDLX(ctx context.Context, matrix SparseMatrix) <-chan SparseMatrix {
	matrixChan := make(chan SparseRow)
	go func() {
		for _, row := range matrix {
			matrixChan <- row
		}
		close(matrixChan)
	}()

	return SolveDLXWithChannel(ctx, matrixChan)
}

func SolveDLXWithChannel(ctx context.Context, matrixChan <-chan SparseRow) <-chan SparseMatrix {
	solutions := make(chan SparseMatrix)
	visitor, totalNodes := createNodeCounter()

	go func() {
		secondaryColumns := make(map[int]bool) // it's empty
		root := BuildDLXAsNeeded(matrixChan, secondaryColumns)
		var solution []*node
		search(ctx, root, solution, solutions, 0, visitor) // Start with depth 0
		fmt.Printf("Total nodes visited: %d\n", *totalNodes)
		close(solutions)
	}()
	return solutions
}
