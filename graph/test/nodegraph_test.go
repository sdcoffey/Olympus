package graph

import (
	"os"
	"time"

	"github.com/cayleygraph/cayley"
	. "github.com/sdcoffey/olympus/checkers"
	"github.com/sdcoffey/olympus/graph"
	. "gopkg.in/check.v1"
)

func (suite *GraphTestSuite) TestNewGraph_setsRootNode(t *C) {
	memGraph, err := cayley.NewMemoryGraph()
	t.Check(err, IsNil)
	oGraph, err := graph.NewGraph(memGraph)
	t.Check(err, IsNil)

	rootNode := oGraph.RootNode
	t.Check(rootNode, Not(IsNil))
	t.Check(rootNode.Id, Equals, graph.RootNodeId)
	t.Check(rootNode.Mode(), Equals, os.ModeDir)
	t.Check(time.Now().Sub(rootNode.MTime()) < time.Second, IsTrue)
	t.Check(rootNode.Name(), Equals, "root")
}

func (suite *GraphTestSuite) TestGraph_NodeWithId_ReturnsNodeWithIdSet(t *C) {
	node := suite.ng.NodeWithId("id")
	t.Check(node.Id, Equals, "id")
}

func (suite *GraphTestSuite) TestNode_NewNode_hasUuidAndTimeStamp(t *C) {
	node, err := suite.ng.NewNode("root", graph.RootNodeId, os.FileMode(0755))
	t.Check(err, IsNil)
	t.Check(node.Id, Not(Equals), "")
	t.Check(node.MTime(), Not(Equals), "")
	t.Check(time.Now().Sub(node.MTime()) < time.Second, Equals, true)
	t.Check(node.Mode(), Equals, os.FileMode(0755))
}

func (suite *GraphTestSuite) TestGraph_NodeWithName_FindsNodeWithNameAndParent(t *C) {
	_, err := suite.ng.NewNode("child", graph.RootNodeId, os.ModeDir)
	t.Check(err, IsNil)

	fetchedChild := suite.ng.NodeWithName(graph.RootNodeId, "child")
	t.Check(fetchedChild, Not(IsNil))
	t.Check(fetchedChild.Name(), Equals, "child")
	t.Check(fetchedChild.Parent().Id, Equals, graph.RootNodeId)
	t.Check(fetchedChild.Mode(), Equals, os.ModeDir)
	t.Check(fetchedChild.MTime().Sub(time.Now()) < time.Second, Equals, true)
}

func (suite *GraphTestSuite) TestRemoveNode_throwsWhenDeletingRootNode(t *C) {
	err := suite.ng.RemoveNode(suite.ng.RootNode)
	t.Check(err, ErrorMatches, "Cannot delete root node")
}

func (suite *GraphTestSuite) TestRemoveNode_deletesAllChildNodes(t *C) {
	child, _ := suite.ng.NewNode("child", graph.RootNodeId, os.ModeDir)
	child2, _ := suite.ng.NewNode("child2", graph.RootNodeId, os.ModeDir)
	suite.ng.NewNode("nestedChild.txt", child2.Id, os.FileMode(0755))

	t.Check(suite.ng.RemoveNode(child2), IsNil)
	t.Check(child2.Children(), HasLen, 0)
	t.Check(suite.ng.RootNode.Children(), HasLen, 1)

	t.Check(suite.ng.RemoveNode(child), IsNil)
	t.Check(suite.ng.RootNode.Children(), HasLen, 0)
}
