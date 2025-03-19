package goverture

import (
	"context"
	"strconv"
	"time"
)

// AppInt is the base integer type used throughout the application
type AppInt int32

type Node struct {
	Top   AppInt
	Ulink AppInt
	Dlink AppInt
}

type Column struct {
	Name  string
	Llink AppInt
	Rlink AppInt
}

// Hide an option
func hide(p AppInt, nodes []Node) {
	q := p + 1
	for q != p {
		x := nodes[q].Top
		u := nodes[q].Ulink
		d := nodes[q].Dlink
		if x <= 0 {
			q = u // q was a spacer
		} else {
			nodes[u].Dlink = d
			nodes[d].Ulink = u
			nodes[x].Top -= 1
			q = q + 1
		}
	}
}

// Unhide an option
func unhide(p AppInt, nodes []Node) {
	q := p - 1
	for q != p {
		x := nodes[q].Top
		u := nodes[q].Ulink
		d := nodes[q].Dlink
		if x <= 0 {
			q = d // q was a spacer
		} else {
			nodes[u].Dlink = q
			nodes[d].Ulink = q
			nodes[x].Top += 1
			q = q - 1
		}
	}
}

// Cover an item
func cover(i AppInt, columns []Column, nodes []Node) {
	p := nodes[i].Dlink
	for p != i {
		hide(p, nodes)
		p = nodes[p].Dlink
	}

	l := columns[i].Llink
	r := columns[i].Rlink
	columns[l].Rlink = r
	columns[r].Llink = l
}

// Uncover an item
func uncover(i AppInt, columns []Column, nodes []Node) {
	l := columns[i].Llink
	r := columns[i].Rlink
	columns[l].Rlink = i
	columns[r].Llink = i

	p := nodes[i].Ulink
	for p != i {
		unhide(p, nodes)
		p = nodes[p].Ulink
	}
}

// selectMinColumn finds the column (header) with the smallest node count.
func selectMinColumn(columns []Column, nodes []Node) AppInt {
	best := columns[0].Rlink
	minCount := nodes[best].Top
	for j := columns[best].Rlink; j != 0; j = columns[j].Rlink {
		if nodes[j].Top < minCount {
			best = j
			minCount = nodes[j].Top
		}

		if minCount == 0 {
			return best
		}
	}
	return best
}

type SearchState struct {
	Columns   []Column
	Nodes     []Node
	Solution  []AppInt
	Solutions chan []AppInt
	Ticker    <-chan time.Time
	Level     int

	// Internal worker management
	ActiveWorkerChannel chan struct{}
	MaxWorkers          int
}

func CopySearchState(state *SearchState) *SearchState {
	// Copy columns
	columnsCopy := make([]Column, len(state.Columns))
	copy(columnsCopy, state.Columns)

	// Copy nodes
	nodesCopy := make([]Node, len(state.Nodes))
	copy(nodesCopy, state.Nodes)

	// Copy solution (+ room for the next item)
	solutionCopy := make([]AppInt, len(state.Solution)+1)
	copy(solutionCopy, state.Solution)

	// Return a new SearchState with copied data
	return &SearchState{
		Columns:             columnsCopy,
		Nodes:               nodesCopy,
		Solution:            solutionCopy,
		Solutions:           state.Solutions,
		Ticker:              state.Ticker,
		Level:               state.Level,
		ActiveWorkerChannel: state.ActiveWorkerChannel,
		MaxWorkers:          state.MaxWorkers,
	}
}

func coverOption(x AppInt, state *SearchState) {
	p := x + 1
	for p != x {
		j := state.Nodes[p].Top
		if j <= 0 {
			p = state.Nodes[p].Ulink
		} else {
			cover(j, state.Columns, state.Nodes)
			p = p + 1
		}
	}
}

func uncoverOption(x AppInt, state *SearchState) {
	p := x - 1
	for p != x {
		j := state.Nodes[p].Top
		if j <= 0 {
			p = state.Nodes[p].Dlink
		} else {
			uncover(j, state.Columns, state.Nodes)
			p = p - 1
		}
	}
}

// Implementation of the Algorithm X ("Exact cover via dancing links") from Knuth's paper
func SolveExactCoverParallel(ctx context.Context, state *SearchState) error {
	if state.Level == 0 {
		state.ActiveWorkerChannel = make(chan struct{}, state.MaxWorkers)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-state.Ticker:
		if len(state.Solution) > 0 {
			solutionCopy := make([]AppInt, len(state.Solution))
			copy(solutionCopy, state.Solution)

			go func(ctx context.Context) {
				select {
				case <-ctx.Done():
					return
				default:
				}
				optionIndex := make([]AppInt, len(solutionCopy))
				for i, x := range solutionCopy {
					for state.Nodes[x].Top > 0 {
						x = x - 1
					}
					optionIndex[i] = -AppInt(state.Nodes[x].Top)
				}
				state.Solutions <- optionIndex
			}(ctx)
		}
	default:
	}

	if state.Columns[0].Rlink == 0 {
		//fmt.Println("found a solution for real !")
		optionIndex := make([]AppInt, len(state.Solution))
		for i, x := range state.Solution {
			for state.Nodes[x].Top > 0 {
				x = x - 1
			}
			optionIndex[i] = -AppInt(state.Nodes[x].Top)
		}
		state.Solutions <- optionIndex

		return nil
	}

	i := selectMinColumn(state.Columns, state.Nodes)
	if state.Nodes[i].Top == 0 {
		// No solutions
		return nil
	}

	cover(i, state.Columns, state.Nodes)
	x := state.Nodes[i].Dlink

outerLoop:
	for x != i {
		select {
		case <-ctx.Done():
			break outerLoop
		default:
		}

		if state.Level <= 5 {
			select {
			case state.ActiveWorkerChannel <- struct{}{}:
				go func(ctx context.Context, x AppInt, newState *SearchState) {
					defer func() { <-newState.ActiveWorkerChannel }()

					select {
					case <-ctx.Done():
						return
					default:
					}
					//fmt.Println("New worker started")
					coverOption(x, newState)

					newState.Solution = append(newState.Solution, x)
					newState.Level++

					//fmt.Println("Starting new worker")
					SolveExactCoverParallel(ctx, newState)

					//fmt.Println("Popping from active worker channel")

					//fmt.Println("Worker done")
				}(ctx, x, CopySearchState(state))
			default:
				coverOption(x, state)

				state.Solution = append(state.Solution, x)
				state.Level++

				if err := SolveExactCoverParallel(ctx, state); err != nil {
					return err
				}

				state.Level--
				state.Solution = state.Solution[:len(state.Solution)-1]

				// X6
				uncoverOption(x, state)
			}
		} else {
			coverOption(x, state)

			state.Solution = append(state.Solution, x)
			state.Level++

			if err := SolveExactCoverParallel(ctx, state); err != nil {
				// return err
			}

			state.Level--
			state.Solution = state.Solution[:len(state.Solution)-1]

			// X6
			uncoverOption(x, state)
		}

		x = state.Nodes[x].Dlink
	}

	// X7
	uncover(i, state.Columns, state.Nodes)

	if state.Level == 0 {
		// Wait until all workers are done
		for {
			if len(state.ActiveWorkerChannel) == 0 {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
	return nil
}

func BuildDLX(itemsCount int, options <-chan SparseRow, isSecondaryColumn func(int) bool) ([]Column, []Node) {
	if itemsCount == 0 {
		return []Column{}, []Node{}
	}

	columns := make([]Column, 1+itemsCount) // +1 for root
	nodes := make([]Node, 1+itemsCount)     // +1 for root

	// Build the columns (horizontally linked)
	columns[0] = Column{
		Name:  "root",
		Llink: 0,
		Rlink: 0,
	}
	nodes[0] = Node{
		Top:   0,
		Ulink: 0,
		Dlink: 0,
	}

	previousPrimaryColumnIndex := AppInt(0)
	for i := 1; i < len(columns); i++ {
		name := "C" + strconv.Itoa(i)

		columns[i] = Column{
			Name:  name,
			Llink: AppInt(i),
			Rlink: AppInt(i), // Secondary columns are not linked
		}
		// We don't link secondary columns
		if !isSecondaryColumn(i - 1) {
			columns[i].Llink = previousPrimaryColumnIndex
			columns[previousPrimaryColumnIndex].Rlink = AppInt(i)

			previousPrimaryColumnIndex = AppInt(i)
			columns[i].Rlink = 0 // point to root
		}

		nodes[i] = Node{
			Top:   0,
			Ulink: AppInt(i), // point to itself
			Dlink: AppInt(i), // point to itself
		}
	}
	columns[0].Llink = previousPrimaryColumnIndex

	var prevOptionFirstIndex AppInt
	optionIndex := 0
	for option := range options {
		itemsCount := len(option)
		elementCount := len(nodes)

		// Insert a Spacer node
		nodes = append(nodes, Node{
			Top:   AppInt(-optionIndex),              // negative value to indicate that it is a spacer
			Ulink: prevOptionFirstIndex,              // address of the first node in the option before the spacer
			Dlink: AppInt(elementCount + itemsCount), // address of the last node in the option after the spacer
		})

		prevOptionFirstIndex = AppInt(len(nodes))

		for i := range option {
			colindex := AppInt(i + 1) // account for the root node at 0
			ulink := nodes[colindex].Ulink

			node := Node{
				Top:   colindex,
				Ulink: ulink,
				Dlink: colindex,
			}

			nodes = append(nodes, node)

			nodes[colindex].Ulink = AppInt(len(nodes) - 1)
			nodes[colindex].Top += 1
			nodes[ulink].Dlink = AppInt(len(nodes) - 1)
		}

		optionIndex++
	}
	nodes = append(nodes, Node{
		Top:   AppInt(-len(options)), // negative value to indicate that it is a spacer
		Ulink: prevOptionFirstIndex,  // address of the first node in the option before the spacer
		Dlink: 0,                     // unused
	})

	return columns, nodes
}
