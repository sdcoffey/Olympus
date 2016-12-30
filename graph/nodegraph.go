package graph

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/cayleygraph/cayley"
	"github.com/pborman/uuid"
	"github.com/sdcoffey/olympus/util"
)

const RootNodeId = "rootNode"

type NodeGraph struct {
	*cayley.Handle
	RootNode *Node
}

func NewGraph(graph *cayley.Handle) (*NodeGraph, error) {
	ng := &NodeGraph{graph, nil}

	root := new(Node)
	root.name = "root"
	root.Id = RootNodeId
	root.mode |= os.ModeDir
	root.graph = ng

	ng.RootNode = root

	if err := root.save(); err != nil {
		return nil, err
	} else {
		ng.RootNode = root
	}

	return ng, nil
}

func (ng *NodeGraph) NewNode(name, parentId string) (*Node, error) {
	node := new(Node)
	node.parentId = parentId
	node.Id = uuid.New()
	node.mTime = time.Now()
	node.name = name
	node.graph = ng

	node.mimeType = util.MimeType(name)

	return node, ng.AddNode(ng.NodeWithId(parentId), node)
}

func (ng *NodeGraph) NewNodeWithNodeInfo(info NodeInfo) (*Node, error) {
	node := ng.NodeWithNodeInfo(info)
	node.Id = uuid.New()

	if info.Type != "" {
		node.mimeType = info.Type
	} else {
		node.mimeType = util.MimeType(info.Name)
	}

	return node, ng.AddNode(ng.NodeWithId(info.ParentId), node)
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
	node.mTime = info.MTime
	node.parentId = info.ParentId
	node.mimeType = info.Type
	node.graph = ng

	return node
}

func (ng *NodeGraph) RemoveNode(nd *Node) (err error) {
	if nd.Id == RootNodeId {
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

func (ng *NodeGraph) CreateDirectory(parentId, name string) (*Node, error) {
	var info NodeInfo
	info.ParentId = parentId
	info.Mode = os.ModeDir
	info.Name = name

	return ng.NewNodeWithNodeInfo(info)
}

func (ng *NodeGraph) AddNode(parent, child *Node) error {
	if parent == nil || !parent.Exists() {
		return fmt.Errorf("Parent %s does not exist", parent.Id)
	} else if !parent.IsDir() {
		return errors.New("Cannot add node to a non-directory")
	} else if parent.Exists() && ng.NodeWithName(parent.Id, child.name) != nil {
		return fmt.Errorf("Node with name %s already exists in %s", child.name, parent.Name())
	}

	child.parentId = parent.Id
	return child.save()
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
	if nd.Type() != "" {
		transaction.RemoveQuad(cayley.Quad(nd.Id, typeLink, nd.Type(), ""))
	}

	err = ng.ApplyTransaction(transaction)

	if err == nil {
		nd.mode = os.FileMode(0)
		nd.mTime = time.Time{}
		nd.name = ""
		nd.parentId = ""
		nd.Id = ""
	}

	return
}
