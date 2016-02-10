package graph

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/google/cayley"
)

const RootNodeId = "rootNode"

type NodeGraph struct {
	*cayley.Handle
	RootNode *Node
}

func NewGraph(graph *cayley.Handle) (*NodeGraph, error) {
	ng := &NodeGraph{graph, nil}

	root := newNode("root", ng)
	root.Id = RootNodeId
	root.mode |= os.ModeDir

	ng.RootNode = root

	if err := root.Save(); err != nil {
		return nil, err
	} else {
		ng.RootNode = root
	}

	return ng, nil
}

func (ng *NodeGraph) NodeWithId(id string) *Node {
	return &Node{Id: id, graph: ng}
}

func (ng *NodeGraph) NodeWithName(parentId, name string) *Node {
	namePath := cayley.StartPath(ng, name).In(nameLink)
	parentpath := cayley.StartPath(ng, parentId).In(parentLink)

	it := namePath.And(parentpath).BuildIterator()
	if cayley.RawNext(it) {
		return ng.NodeWithId(ng.NameOf(it.Result()))
	}

	return nil
}

func (ng *NodeGraph) NodeWithNodeInfo(info NodeInfo) *Node {
	node := ng.NodeWithId(info.Id)
	node.name = info.Name
	node.mode = os.FileMode(info.Mode)
	node.size = info.Size
	node.mTime = info.MTime
	node.parentId = info.ParentId

	return node
}

func (ng *NodeGraph) RemoveNode(nd *Node) (err error) {
	if nd.Id == ng.RootNode.Id && nd.Parent() == nil {
		return errors.New("Cannot delete root node")
	}

	children := nd.Children()
	if len(children) > 0 {
		for i := 0; i < len(children) && err == nil; i++ {
			err = ng.RemoveNode(children[i])
		}
	}

	return ng.removeNode(nd)
}

func (ng *NodeGraph) CreateDirectory(parent *Node, name string) (child *Node, err error) {
	child = newNode(name, ng)
	child.mode |= os.ModeDir

	if err = ng.addNode(parent, child); err != nil {
		return
	}
	return
}

func (ng *NodeGraph) MoveNode(nd *Node, newName, newParentId string) (err error) {
	if nd.Parent() == nil {
		return errors.New("Cannot move root node")
	} else if newParentId == nd.Id {
		return errors.New("Cannot move node inside itself")
	} else if newParentId == nd.Parent().Id && newName == nd.Name() {
		return nil
	}

	if nd.Name() != newName {
		nd.name = newName
	}

	newParent := ng.NodeWithId(newParentId)
	if err = ng.addNode(newParent, nd); err != nil {
		nd.name = ""
		return
	}

	return nil
}

func (ng *NodeGraph) addNode(parent, child *Node) (err error) {
	if !parent.IsDir() {
		return errors.New("Cannot add node to a non-directory")
	} else if parent.Exists() && ng.NodeWithName(parent.Id, child.name) != nil {
		return errors.New(fmt.Sprintf("Node with name %s already exists in %s", child.name, parent.Name()))
	} else if !parent.Exists() {
		return errors.New(fmt.Sprint("Parent ", parent.Name(), " does not exist"))
	}

	child.parentId = parent.Id
	return child.Save()
}

func (ng *NodeGraph) removeNode(nd *Node) (err error) {
	if len(nd.Children()) > 0 {
		return errors.New("Can't delete node with children, must delete children first")
	}

	transaction := cayley.NewTransaction()
	if nd.Mode() > 0 {
		transaction.RemoveQuad(cayley.Quad(nd.Id, modeLink, fmt.Sprint(int(nd.Mode())), ""))
	}
	if !nd.MTime().IsZero() {
		transaction.RemoveQuad(cayley.Quad(nd.Id, mTimeLink, fmt.Sprint(nd.MTime().Format(timeFormat)), ""))
	}
	if nd.Name() != "" {
		transaction.RemoveQuad(cayley.Quad(nd.Id, nameLink, nd.Name(), ""))
	}
	if nd.Parent() != nil {
		transaction.RemoveQuad(cayley.Quad(nd.Id, parentLink, nd.Parent().Id, ""))
	}
	if nd.Size() > 0 {
		transaction.RemoveQuad(cayley.Quad(nd.Id, sizeLink, fmt.Sprint(nd.Size()), ""))
	}

	err = ng.ApplyTransaction(transaction)

	if err == nil {
		nd.mode = os.FileMode(0)
		nd.size = 0
		nd.mTime = time.Time{}
		nd.name = ""
		nd.parentId = ""
		nd.Id = ""
	}

	return
}
