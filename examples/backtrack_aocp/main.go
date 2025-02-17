package main

import (
	"fmt"
	"strconv"
)

type Node struct {
	top   int
	ulink int
	dlink int
}

type Column struct {
	name string
	llink int
	rlink int
}

// Hide an option
func hide(p int, columns []Column, nodes []Node) {
	q := p + 1
	for q != p {
		x := nodes[q].top
		u := nodes[q].ulink
		d := nodes[q].dlink
		if x <= 0 {
			q = u // q was a spacer
		} else {
			nodes[u].dlink = d
			nodes[d].ulink = u
			nodes[x].top -= 1
			q = q + 1
		}
	}
}

// Unhide an option
func unhide(p int, columns []Column, nodes []Node) {
	q := p - 1
	for q != p {
		x := nodes[q].top
		u := nodes[q].ulink
		d := nodes[q].dlink
		if x <= 0 {
			q = d // q was a spacer
		} else {
			nodes[u].dlink = q
			nodes[d].ulink = q
			nodes[x].top += 1
			q = q - 1
		}
	}
}

// Cover an item
func cover(i int, columns []Column, nodes []Node) {
	p := nodes[i].dlink
	for p != i {
		hide(p, columns, nodes)
		p = nodes[p].dlink
	}

	l := columns[i].llink
	r := columns[i].rlink
	columns[l].rlink = r
	columns[r].llink = l
}

// Uncover an item
func uncover(i int, columns []Column, nodes []Node) {
	l := columns[i].llink
	r := columns[i].rlink
	columns[l].rlink = i
	columns[r].llink = i

	p := nodes[i].ulink
	for p != i {
		unhide(p, columns, nodes)
		p = nodes[p].ulink
	}
}

// Implementation of the Algorithm X ("Exact cover via dancing links") from Knuth's paper
func solveExactCover(columns []Column, nodes []Node, solution []int) {
	// l := 0 // level

	// X2
	if(columns[0].rlink == 0) {
		// all items have been covered
		fmt.Println("Solution found")
		fmt.Println(solution)
		return
	}

	// X3
	i := columns[0].rlink // TODO: better choice of i

	// X4
	cover(i, columns, nodes)
	x := nodes[i].dlink

	// X5
	for x != i {
		p := x + 1
		for p != x {
			j := nodes[p].top
			if j <= 0 {
				p = nodes[p].ulink
			} else {
				cover(j, columns, nodes)
				p = p + 1
			}
		}

		solution = append(solution, x)
		solveExactCover(columns, nodes, solution)
		solution = solution[:len(solution)-1]

		// X6
		p = x - 1
		for p != x {
			j := nodes[p].top
			if j <= 0 {
				p = nodes[p].dlink
			} else {
				uncover(j, columns, nodes)
				p = p - 1
			}
		}

		x = nodes[x].dlink
	}

	// X7
	uncover(i, columns, nodes)
}

func main() {
	columns := make([]Column, 1 + 7)
	nodes := make([]Node, 1 + 7)

	for i := range(columns) {
		var name string
		if i == 0 {
			name = "root"
		} else {
			name = "C" + strconv.Itoa(i)
		}

		columns[i] = Column{
			name: name,
			llink: (i - 1 + len(columns)) % len(columns), // Take care of negative modulo in go
			rlink: (i + 1) % len(columns),
		}
		nodes[i] = Node{
			top: 0,
			ulink: i, // point to itself
			dlink: i, // point to itself
		}
	}

	options := [6][7]int{
		{0,0,1,0,1,0,0},
		{1,0,0,1,0,0,1},
		{0,1,1,0,0,1,0},
		{1,0,0,1,0,1,0},
		{0,1,0,0,0,0,1},
		{0,0,0,1,1,0,1},
	}

	var prevOptionFirstIndex int
	
	for _, option := range(options) {
		// Count how many items are in the option
		itemsCount := 0
		for _, i := range(option) {
			if i == 1 {
				itemsCount++
			}
		}

		// Insert a Spacer node
		elementCount := len(nodes)
		nodes = append(nodes, Node{
			top: 0,
			ulink: prevOptionFirstIndex, // address of the first node in the option before the spacer
			dlink: elementCount + itemsCount, // address of the last node in the option after the spacer (ie the current option)
		})

		prevOptionFirstIndex = len(nodes)

		for i := range(len(option)) {
			if option[i] == 1 {
				colindex := i + 1 // account for the root node at 0
				ulink := nodes[colindex].ulink

				node := Node{
					top: colindex,
					ulink: ulink,
					dlink: colindex,
				}

				nodes = append(nodes, node)

				nodes[colindex].ulink = len(nodes) - 1
				nodes[colindex].top += 1
				nodes[ulink].dlink = len(nodes) - 1
			}
		}
	}
	// TODO: Add a spacer node at the end of the list
	nodes = append(nodes, Node{
		top: 0,
		ulink: prevOptionFirstIndex, // address of the first node in the option before the spacer
		dlink: 0,
	})

	solution := []int{}
	solveExactCover(columns, nodes, solution)

	fmt.Println("Done")

}
