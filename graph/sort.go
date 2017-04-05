package graph

import "sort"

type Sorter int

const (
	Alphabetical Sorter = iota + 1
	DateModified
)

func Reversed(s Sorter) Sorter {
	return Sorter(-int(s))
}

func Sort(nodes []*Node, sType Sorter) {
	sort.Slice(nodes, func(i, j int) bool {
		switch sType {
		case Alphabetical:
			return nodes[i].Name() < nodes[j].Name()
		case -Alphabetical:
			return nodes[i].Name() > nodes[j].Name()
		case DateModified:
			return nodes[i].MTime().Before(nodes[j].MTime())
		case -DateModified:
			return nodes[i].MTime().After(nodes[j].MTime())
		default:
			return false
		}
	})
}
