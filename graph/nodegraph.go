package graph

import (
	"errors"
	"fmt"
	"os"

	"time"

	"github.com/cayleygraph/cayley"
	"github.com/cayleygraph/cayley/quad"
	"github.com/pborman/uuid"
)

const RootNodeId = "rootNode"

type NodeGraph struct {
	*cayley.Handle
	RootNode *Node
}

func NewGraph(graph *cayley.Handle) (*NodeGraph, error) {
	ng := &NodeGraph{graph, nil}

	root := new(Node)
	root.Id = RootNodeId
	root.graph = ng
	root.propCache = make(map[string]interface{})

	now := time.Now().UTC()
	if err := graph.AddQuad(cayley.Triple(RootNodeId, nameLink, "root")); err != nil {
		return nil, err
	} else if err = graph.AddQuad(cayley.Triple(RootNodeId, modeLink, int(os.ModeDir))); err != nil {
		return nil, err
	} else if err = graph.AddQuad(cayley.Triple(RootNodeId, mTimeLink, now.Unix())); err != nil {
		return nil, err
	}

	root.propCache[nameLink] = "root"
	root.propCache[modeLink] = os.FileMode(os.ModeDir)
	root.propCache[mTimeLink] = now

	ng.RootNode = root

	return ng, nil
}

func (ng *NodeGraph) _newNode() *Node {
	nd := new(Node)
	nd.Id = uuid.New()
	nd.graph = ng
	nd.propCache = make(map[string]interface{})

	return nd
}

func (ng *NodeGraph) NewNode(name, parentId string, mode os.FileMode) (nd *Node, err error) {
	defer func() {
		if r := recover(); r != nil {
			nd = nil
			err = fmt.Errorf("Error creating new node: %s", err.Error())
		}
	}()

	ok := func(err error) {
		if err != nil {
			panic(err)
		}
	}

	nd = ng._newNode()

	ok(nd.SetName(name))
	ok(nd.Move(parentId))
	ok(nd.Touch(time.Now()))
	ok(nd.SetMode(mode))

	return nd, nil
}

func (ng *NodeGraph) NodeWithId(id string) *Node {
	return &Node{Id: id, graph: ng, propCache: make(map[string]interface{})}
}

func (ng *NodeGraph) NodeWithName(parentId, name string) *Node {
	namePath := cayley.StartPath(ng, quad.String(name)).In(nameLink)
	parentPath := cayley.StartPath(ng, quad.String(parentId)).In(parentLink)

	it := namePath.And(parentPath).BuildIterator()
	if it.Next() {
		return ng.NodeWithId(quad.NativeOf(ng.NameOf(it.Result())).(string))
	}

	return nil
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

func (ng *NodeGraph) removeNode(nd *Node) (err error) {
	if len(nd.Children()) > 0 {
		return errors.New("Can't delete node with children, must delete children first")
	}

	transaction := cayley.NewTransaction()
	if nd.Mode() > 0 {
		transaction.RemoveQuad(cayley.Triple(nd.Id, modeLink, int(nd.Mode())))
	}
	if !nd.MTime().IsZero() {
		transaction.RemoveQuad(cayley.Triple(nd.Id, mTimeLink, nd.MTime().Unix()))
	}
	if nd.Name() != "" {
		transaction.RemoveQuad(cayley.Triple(nd.Id, nameLink, nd.Name()))
	}
	if nd.Parent() != nil {
		transaction.RemoveQuad(cayley.Triple(nd.Id, parentLink, nd.Parent().Id))
	}

	return ng.ApplyTransaction(transaction)
}
