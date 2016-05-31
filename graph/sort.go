package graph

import "sort"

type sortType int

const (
	Alphabetical = iota
	ReverseAlphabetical
)

func Sort(nodes []*Node, sType sortType) {
	sort.Sort(nodeSorter{nodes, sType})
}

type nodeSorter struct {
	nodes []*Node
	sType sortType
}

func (b nodeSorter) Len() int {
	return len(b.nodes)
}

func (b nodeSorter) Swap(i, j int) {
	b.nodes[i], b.nodes[j] = b.nodes[j], b.nodes[i]
}

func (a nodeSorter) Less(i, j int) bool {
	switch a.sType {
	case Alphabetical:
		return a.nodes[i].Name() < a.nodes[j].Name()
	case ReverseAlphabetical:
		return a.nodes[i].Name() > a.nodes[j].Name()
	default:
		return false
	}
}
