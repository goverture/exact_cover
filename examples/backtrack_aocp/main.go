package main

import "strconv"

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
				node := Node{
					top: i,
					ulink: 0,
					dlink: 0,
				}
				nodes = append(nodes, node)
			}
		}
	}
	// TODO: Add a spacer node at the end of the list

	println("foo")

}
