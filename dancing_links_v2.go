package goverture

import "strconv"

type Node struct {
	Top   int
	Ulink int
	Dlink int
}

type Column struct {
	Name  string
	Llink int
	Rlink int
}

// Hide an option
func hide(p int, nodes []Node) {
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
func unhide(p int, nodes []Node) {
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
func cover(i int, columns []Column, nodes []Node) {
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
func uncover(i int, columns []Column, nodes []Node) {
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
func selectMinColumn(columns []Column, nodes []Node) int {
    // Start with the first column right of root.
    best := columns[0].Rlink
    minCount := nodes[best].Top
    // Iterate through all columns until we circle back to the root (index 0).
    for j := columns[best].Rlink; j != 0; j = columns[j].Rlink {
        if nodes[j].Top < minCount {
            best = j
            minCount = nodes[j].Top
        }
    }
    return best
}

// Implementation of the Algorithm X ("Exact cover via dancing links") from Knuth's paper
func SolveExactCover(columns []Column, nodes []Node, solution []int, visitSolution func([]int)) {
	if columns[0].Rlink == 0 {
		visitSolution(solution)
		return
	}

	i := selectMinColumn(columns, nodes)
	cover(i, columns, nodes)
	x := nodes[i].Dlink

	for x != i {
		p := x + 1
		for p != x {
			j := nodes[p].Top
			if j <= 0 {
				p = nodes[p].Ulink
			} else {
				cover(j, columns, nodes)
				p = p + 1
			}
		}

		solution = append(solution, x)
		SolveExactCover(columns, nodes, solution, visitSolution)
		solution = solution[:len(solution)-1]

		// X6
		p = x - 1
		for p != x {
			j := nodes[p].Top
			if j <= 0 {
				p = nodes[p].Dlink
			} else {
				uncover(j, columns, nodes)
				p = p - 1
			}
		}

		x = nodes[x].Dlink
	}

	// X7
	uncover(i, columns, nodes)
}

func BuildDLX(options [][]int, isSecondaryColumn func(int) bool) ([]Column, []Node) {
	if len(options) == 0 {
		return []Column{}, []Node{}
	}

	columns := make([]Column, 1+len(options[0]))
	nodes := make([]Node, 1+len(options[0]))

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

	previousPrimaryColumnIndex := 0
	for i := 1; i < len(columns); i++ {
		name := "C" + strconv.Itoa(i)

		columns[i] = Column{
			Name:  name,
			Llink: i,
			Rlink: i, // Secundary columns are not linked
		}
		// We don't link secondary columns
		if !isSecondaryColumn(i-1) {
			columns[i].Llink = previousPrimaryColumnIndex
			columns[previousPrimaryColumnIndex].Rlink = i

			previousPrimaryColumnIndex = i
			columns[i].Rlink = 0 // point to root
		}

		nodes[i] = Node{
			Top:   0,
			Ulink: i, // point to itself
			Dlink: i, // point to itself
		}
	}
	columns[0].Llink = previousPrimaryColumnIndex

	var prevOptionFirstIndex int

	for optionIndex, option := range options {
		// Count how many items are in the option
		itemsCount := 0
		for _, i := range option {
			if i == 1 {
				itemsCount++
			}
		}

		// Insert a Spacer node
		elementCount := len(nodes)
		nodes = append(nodes, Node{
			Top:   -optionIndex,              // negative value to indicate that it is a spacer
			Ulink: prevOptionFirstIndex,      // address of the first node in the option before the spacer
			Dlink: elementCount + itemsCount, // address of the last node in the option after the spacer (ie the current option)
		})

		prevOptionFirstIndex = len(nodes)

		for i := 0; i < len(option); i++ {
			if option[i] == 1 {
				colindex := i + 1 // account for the root node at 0
				ulink := nodes[colindex].Ulink

				node := Node{
					Top:   colindex,
					Ulink: ulink,
					Dlink: colindex,
				}

				nodes = append(nodes, node)

				nodes[colindex].Ulink = len(nodes) - 1
				nodes[colindex].Top += 1
				nodes[ulink].Dlink = len(nodes) - 1
			}
		}
	}
	nodes = append(nodes, Node{
		Top:   -len(options),        // negative value to indicate that it is a spacer
		Ulink: prevOptionFirstIndex, // address of the first node in the option before the spacer
		Dlink: 0,                    // unused
	})

	return columns, nodes
}
