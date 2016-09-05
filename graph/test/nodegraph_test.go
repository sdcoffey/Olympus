package graph

import (
	"os"
	"time"

	. "github.com/sdcoffey/olympus/Godeps/_workspace/src/gopkg.in/check.v1"
	"github.com/sdcoffey/olympus/graph"
)

func (suite *GraphTestSuite) TestNode_NewNode_hasUuidAndTimeStamp(t *C) {
	node, err := suite.ng.NewNode("root", graph.RootNodeId)
	t.Check(err, IsNil)
	t.Check(node.Id, Not(Equals), "")
	t.Check(node.MTime(), Not(Equals), "")
	t.Check(time.Now().Sub(node.MTime()) < time.Second, Equals, true)
}

func (suite *GraphTestSuite) TestNodeWithNodeInfo(t *C) {
	now := time.Now()
	info := graph.NodeInfo{
		ParentId: graph.RootNodeId,
		Name:     "node",
		Size:     1,
		MTime:    now,
		Mode:     4,
		Type:     "application/json",
	}

	node, err := suite.ng.NewNodeWithNodeInfo(info)
	t.Check(err, IsNil)

	t.Check(node.Id, Not(Equals), "")
	t.Check(node.Parent().Id, Equals, graph.RootNodeId)
	t.Check(node.Name(), Equals, "node")
	t.Check(node.MTime(), Equals, now)
	t.Check(node.Mode(), Equals, os.FileMode(4))
	t.Check(node.Type(), Equals, "application/json")
}

func (suite *GraphTestSuite) TestNodeWithName(t *C) {
	nodeInfo := graph.NodeInfo{
		Name:     "child",
		ParentId: graph.RootNodeId,
		Mode:     os.ModeDir,
		MTime:    time.Now(),
	}

	_, err := suite.ng.NewNodeWithNodeInfo(nodeInfo)
	t.Check(err, IsNil)

	fetchedChild := suite.ng.NodeWithName(graph.RootNodeId, "child")
	t.Check(fetchedChild, Not(IsNil))
	t.Check(fetchedChild.Name(), Equals, "child")
	t.Check(fetchedChild.Parent().Id, Equals, graph.RootNodeId)
	t.Check(fetchedChild.Mode(), Equals, os.ModeDir)
	t.Check(fetchedChild.MTime().Sub(time.Now()) < time.Second, Equals, true)
}

func (suite *GraphTestSuite) TestCreateDirectory(t *C) {
	child, err := suite.ng.CreateDirectory(graph.RootNodeId, "child")
	t.Check(err, IsNil)
	t.Check(t, Not(IsNil))
	t.Check(child.Id, Not(Equals), "")
	t.Check(child.Parent().Id, Equals, graph.RootNodeId)
	t.Check(suite.ng.RootNode.Children(), HasLen, 1)
}

func (suite *GraphTestSuite) TestCreateDirectory_returnsErrorWhenParentNotDir(t *C) {
	childNode, err := suite.ng.NewNode("child", graph.RootNodeId)
	t.Check(err, IsNil)

	_, err = suite.ng.CreateDirectory(childNode.Id, "secondChild")
	t.Check(err, ErrorMatches, "Cannot add node to a non-directory")
}

func (suite *GraphTestSuite) TestRemoveNode_throwsWhenDeletingRootNode(t *C) {
	err := suite.ng.RemoveNode(suite.ng.RootNode)
	t.Check(err, ErrorMatches, "Cannot delete root node")
}

func (suite *GraphTestSuite) TestRemoveNode_deletesAllChildNodes(t *C) {
	child, _ := suite.ng.CreateDirectory(graph.RootNodeId, "child")
	child2, _ := suite.ng.CreateDirectory(graph.RootNodeId, "child2")
	suite.ng.NewNodeWithNodeInfo(graph.NodeInfo{
		ParentId: child2.Id,
		Name:     "nestedChild.txt",
		Mode:     755,
	})

	t.Check(suite.ng.RemoveNode(child2), IsNil)
	t.Check(child2.Children(), HasLen, 0)
	t.Check(suite.ng.RootNode.Children(), HasLen, 1)

	t.Check(suite.ng.RemoveNode(child), IsNil)
	t.Check(suite.ng.RootNode.Children(), HasLen, 0)
}

func (suite *GraphTestSuite) TestNewNode_returnsAnErrorWhenDuplicateSiblingExists(t *C) {
	_, err := suite.ng.CreateDirectory(graph.RootNodeId, "child")
	t.Check(err, IsNil)

	_, err = suite.ng.NewNode("child", graph.RootNodeId)
	t.Check(err, ErrorMatches, "Node with name child already exists in root")
}

func (suite *GraphTestSuite) TestNewNode_throwsWhenParentDoesNotExist(t *C) {
	_, err := suite.ng.NewNode("file", "not-a-file")
	t.Check(err, ErrorMatches, "Parent not-a-file does not exist")
}
