package app

import "github.com/thegriglat/platui/internal/plane"

// Node is a task in the subtask tree.
type Node struct {
	Item     plane.WorkItem
	Children []*Node
	Depth    int
}

// Tree builds a forest by ParentID within items; a parent outside the list makes the task
// a root; tasks on a cycle become roots. Order is as in the input.
func Tree(items []plane.WorkItem) []*Node {
	nodes := make(map[string]*Node, len(items))
	for _, it := range items {
		nodes[it.ID] = &Node{Item: it}
	}
	parent := func(id string) *Node {
		n := nodes[id]
		if n.Item.ParentID == "" || n.Item.ParentID == id {
			return nil
		}
		return nodes[n.Item.ParentID]
	}
	onCycle := func(id string) bool {
		cur := id
		for i := 0; i <= len(nodes); i++ {
			p := parent(cur)
			if p == nil {
				return false
			}
			if p.Item.ID == id {
				return true
			}
			cur = p.Item.ID
		}
		return false // the chain ran into someone else's cycle: id is its "tail", not on the cycle
	}
	var roots []*Node
	for _, it := range items {
		n := nodes[it.ID]
		if p := parent(it.ID); p != nil && !onCycle(it.ID) {
			p.Children = append(p.Children, n)
		} else {
			roots = append(roots, n)
		}
	}
	var setDepth func(n *Node, d int)
	setDepth = func(n *Node, d int) {
		n.Depth = d
		for _, c := range n.Children {
			setDepth(c, d+1)
		}
	}
	for _, r := range roots {
		setDepth(r, 0)
	}
	return roots
}
