package graph

import (
	"fmt"
	"os"
	"time"

	"github.com/sdcoffey/olympus/Godeps/_workspace/src/github.com/stretchr/testify/assert"
	. "github.com/sdcoffey/olympus/Godeps/_workspace/src/gopkg.in/check.v1"
	"github.com/sdcoffey/olympus/graph"
)

func (suite *GraphTestSuite) TestNode_NewNode_hasUuidAndTimeStamp(t *C) {
	node, err := suite.ng.NewNode("root", graph.RootNodeId)
	assert.NoError(t, err)
	assert.NotEmpty(t, node.Id)
	assert.NotEmpty(t, node.MTime())
	assert.True(t, time.Now().Sub(node.MTime()) < time.Second, Equals, true)
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
	assert.NoError(t, err)

	assert.NotEmpty(t, node.Id)
	assert.Equal(t, graph.RootNodeId, node.Parent().Id)
	assert.Equal(t, "node", node.Name())
	assert.Equal(t, now, node.MTime())
	assert.Equal(t, os.FileMode(4), node.Mode())
	assert.Equal(t, "application/json", node.Type())
}

func (suite *GraphTestSuite) TestNodeWithName(t *C) {
	nodeInfo := graph.NodeInfo{
		Name:     "child",
		ParentId: graph.RootNodeId,
		Mode:     os.ModeDir,
		MTime:    time.Now(),
	}

	_, err := suite.ng.NewNodeWithNodeInfo(nodeInfo)
	assert.NoError(t, err)

	fetchedChild := suite.ng.NodeWithName(graph.RootNodeId, "child")
	assert.NotNil(t, fetchedChild)
	assert.Equal(t, "child", fetchedChild.Name())
	assert.Equal(t, graph.RootNodeId, fetchedChild.Parent().Id)
	assert.EqualValues(t, os.ModeDir, fetchedChild.Mode())
	assert.True(t, fetchedChild.MTime().Sub(time.Now()) < time.Second)
}

func (suite *GraphTestSuite) TestCreateDirectory(t *C) {
	child, err := suite.ng.CreateDirectory(graph.RootNodeId, "child")
	assert.NoError(t, err)
	assert.NotNil(t, child)
	assert.NotEmpty(t, child.Id)
	assert.Equal(t, graph.RootNodeId, child.Parent().Id)
	assert.Len(t, suite.ng.RootNode.Children(), 1)
}

func (suite *GraphTestSuite) TestCreateDirectory_returnsErrorWhenParentNotDir(t *C) {
	childNode, err := suite.ng.NewNode("child", graph.RootNodeId)
	assert.NoError(t, err)

	_, err = suite.ng.CreateDirectory(childNode.Id, "secondChild")
	assert.EqualError(t, err, "Cannot add node to a non-directory")
}

func (suite *GraphTestSuite) TestRemoveNode_throwsWhenDeletingRootNode(t *C) {
	err := suite.ng.RemoveNode(suite.ng.RootNode)
	assert.EqualError(t, err, "Cannot delete root node")
}

func (suite *GraphTestSuite) TestRemoveNode_deletesAllChildNodes(t *C) {
	child, _ := suite.ng.CreateDirectory(graph.RootNodeId, "child")
	child2, _ := suite.ng.CreateDirectory(graph.RootNodeId, "child2")
	suite.ng.CreateDirectory(child2.Id, "child3")

	assert.NoError(t, suite.ng.RemoveNode(child2))
	assert.Len(t, child2.Children(), 0)
	assert.Len(t, suite.ng.RootNode.Children(), 1)

	assert.NoError(t, suite.ng.RemoveNode(child))
	assert.Len(t, suite.ng.RootNode.Children(), 0)
}

func (suite *GraphTestSuite) TestNewNode_returnsAnErrorWhenDuplicateSiblingExists(t *C) {
	_, err := suite.ng.CreateDirectory(graph.RootNodeId, "child")
	t.Check(err, IsNil)

	_, err = suite.ng.NewNode("child", graph.RootNodeId)
	assert.EqualError(t, err, fmt.Sprintf("Node with name %s already exists in %s", "child", suite.ng.RootNode.Name()))
}

func (suite *GraphTestSuite) TestNewNode_throwsWhenParentDoesNotExist(t *C) {
	_, err := suite.ng.NewNode("file", "not-a-file")
	assert.EqualError(t, err, "Parent not-a-file does not exist")
}
