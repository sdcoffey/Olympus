package test

import (
	"bytes"
	"os"

	"time"

	"github.com/sdcoffey/olympus/Godeps/_workspace/src/github.com/stretchr/testify/assert"
	. "github.com/sdcoffey/olympus/Godeps/_workspace/src/gopkg.in/check.v1"
	"github.com/sdcoffey/olympus/graph"
	"github.com/sdcoffey/olympus/graph/testutils"
)

func (suite *ApiClientTestSuite) TestApiClient_ListNodes_returnsEmptyListWhenNoNodes(t *C) {
	nodes, err := suite.client.ListNodes(graph.RootNodeId)
	assert.NoError(t, err)
	assert.Len(t, nodes, 0)
}

func (suite *ApiClientTestSuite) TestApiClient_ListNodes_returnsNodes(t *C) {
	_, err := suite.ng.CreateDirectory(graph.RootNodeId, "child1")
	assert.NoError(t, err)

	_, err = suite.ng.CreateDirectory(graph.RootNodeId, "child2")
	assert.NoError(t, err)

	nodes, err := suite.client.ListNodes(graph.RootNodeId)
	assert.NoError(t, err)

	assert.Len(t, nodes, 2)

	assert.Equal(t, "child1", nodes[0].Name)
	assert.True(t, nodes[0].Mode&os.ModeDir > 0)
	assert.Equal(t, graph.RootNodeId, nodes[0].ParentId)

	assert.Equal(t, "child2", nodes[1].Name)
	assert.True(t, nodes[1].Mode&os.ModeDir > 0)
	assert.Equal(t, graph.RootNodeId, nodes[1].ParentId)
}

func (suite *ApiClientTestSuite) TestApiClient_ListBlocks_listsBlocks(t *C) {
	node, err := suite.ng.NewNodeWithNodeInfo(graph.NodeInfo{
		ParentId: graph.RootNodeId,
		Name:     "thing.txt",
		Mode:     0755,
	})
	assert.NoError(t, err)

	dat1 := testutils.RandDat(graph.MEGABYTE)
	hash1 := graph.Hash(dat1)
	dat2 := testutils.RandDat(graph.MEGABYTE)
	hash2 := graph.Hash(dat2)

	assert.NoError(t, node.WriteData(dat1, 0))
	assert.NoError(t, node.WriteData(dat2, graph.MEGABYTE))

	blocks, err := suite.client.ListBlocks(node.Id)
	assert.NoError(t, err)
	assert.Len(t, blocks, 2)
	assert.Equal(t, hash1, blocks[0].Hash)
	assert.EqualValues(t, 0, blocks[0].Offset)
	assert.Equal(t, hash2, blocks[1].Hash)
	assert.EqualValues(t, graph.MEGABYTE, blocks[1].Offset)
}

func (suite *ApiClientTestSuite) TestApiClient_WriteBlock_writesBlock(t *C) {
	node, err := suite.ng.NewNodeWithNodeInfo(graph.NodeInfo{
		ParentId: graph.RootNodeId,
		Name:     "thing.txt",
		Mode:     0755,
	})
	assert.NoError(t, err)

	dat := testutils.RandDat(graph.MEGABYTE)
	buf := bytes.NewBuffer(dat)

	err = suite.client.WriteBlock(node.Id, 0, graph.Hash(dat), buf)
	assert.NoError(t, err)

	blocks := node.Blocks()
	assert.Len(t, blocks, 1)
	assert.Equal(t, graph.Hash(dat), blocks[0].Hash)
	assert.EqualValues(t, 0, blocks[0].Offset)
}

func (suite *ApiClientTestSuite) TestApiClient_RemoveNode_removesNode(t *C) {
	node, err := suite.ng.NewNodeWithNodeInfo(graph.NodeInfo{
		ParentId: graph.RootNodeId,
		Name:     "thing.txt",
		Mode:     0755,
	})
	assert.NoError(t, err)

	assert.NoError(t, suite.client.RemoveNode(node.Id))
	assert.Len(t, suite.ng.RootNode.Children(), 0)
}

func (suite *ApiClientTestSuite) TestApiClient_CreateNode_createsNode(t *C) {
	nodeInfo := graph.NodeInfo{
		ParentId: graph.RootNodeId,
		Name:     "file.txt",
		Mode:     755,
	}
	returnedNodeInfo, err := suite.client.CreateNode(nodeInfo)
	assert.NoError(t, err)

	createdNode := suite.ng.NodeWithId(returnedNodeInfo.Id)
	assert.Equal(t, nodeInfo.Name, createdNode.Name())
	assert.EqualValues(t, int(755), int(createdNode.Mode()))
	assert.True(t, time.Now().Sub(createdNode.MTime()) < time.Second)
}

func (suite *ApiClientTestSuite) TestApiClient_UpdateNode_updatesNode(t *C) {
	node, err := suite.ng.NewNodeWithNodeInfo(graph.NodeInfo{
		ParentId: graph.RootNodeId,
		Name:     "thing.txt",
		Mode:     0755,
	})
	assert.NoError(t, err)

	info := node.NodeInfo()
	info.Mode = 700
	info.Name = "file.dat"

	err = suite.client.UpdateNode(info)
	assert.NoError(t, err)

	assert.Equal(t, "file.dat", node.Name())
	assert.EqualValues(t, 700, int(node.Mode()))
}

func (suite *ApiClientTestSuite) TestApiClient_do_readsErrorFromBodyOnNonOkStatus(t *C) {
	_, err := suite.client.ListNodes("not-found")
	assert.Contains(t, err.Error(), "Node with id: not-found does not exist")
}
