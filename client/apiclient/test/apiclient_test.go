package test

import (
	"bytes"
	"os"

	"time"

	. "github.com/sdcoffey/olympus/checkers"
	"github.com/sdcoffey/olympus/graph"
	"github.com/sdcoffey/olympus/graph/testutils"
	. "gopkg.in/check.v1"
)

func (suite *ApiClientTestSuite) TestApiClient_ListNodes_returnsEmptyListWhenNoNodes(t *C) {
	nodes, err := suite.client.ListNodes(graph.RootNodeId)
	t.Check(err, IsNil)
	t.Check(nodes, HasLen, 0)
}

func (suite *ApiClientTestSuite) TestApiClient_ListNodes_returnsNodes(t *C) {
	_, err := suite.ng.NewNode("child1", graph.RootNodeId, os.ModeDir)
	t.Check(err, IsNil)

	_, err = suite.ng.NewNode("child2", graph.RootNodeId, os.ModeDir)
	t.Check(err, IsNil)

	nodes, err := suite.client.ListNodes(graph.RootNodeId)
	t.Check(err, IsNil)

	t.Assert(nodes, HasLen, 2)

	t.Check(nodes[0].Name, Equals, "child1")
	t.Check(nodes[0].Mode&os.ModeDir > 0, IsTrue)
	t.Check(nodes[0].ParentId, Equals, graph.RootNodeId)

	t.Check(nodes[1].Name, Equals, "child2")
	t.Check(nodes[1].Mode&os.ModeDir > 0, IsTrue)
	t.Check(nodes[1].ParentId, Equals, graph.RootNodeId)
}

func (suite *ApiClientTestSuite) TestApiClient_ListBlocks_listsBlocks(t *C) {
	node, err := suite.ng.NewNode("thing.txt", graph.RootNodeId, os.FileMode(0755))
	t.Check(err, IsNil)

	dat1 := testutils.RandDat(graph.MEGABYTE)
	hash1 := graph.Hash(dat1)
	dat2 := testutils.RandDat(graph.MEGABYTE)
	hash2 := graph.Hash(dat2)

	t.Check(node.WriteData(dat1, 0), IsNil)
	t.Check(node.WriteData(dat2, graph.MEGABYTE), IsNil)

	blocks, err := suite.client.ListBlocks(node.Id)
	t.Check(err, IsNil)
	t.Check(blocks, HasLen, 2)
	t.Check(blocks[0].Hash, Equals, hash1)
	t.Check(blocks[0].Offset, Equals, int64(0))
	t.Check(blocks[1].Hash, Equals, hash2)
	t.Check(blocks[1].Offset, Equals, int64(graph.MEGABYTE))
}

func (suite *ApiClientTestSuite) TestApiClient_WriteBlock_writesBlock(t *C) {
	node, err := suite.ng.NewNode("thing.txt", graph.RootNodeId, os.FileMode(0755))
	t.Check(err, IsNil)

	dat := testutils.RandDat(graph.MEGABYTE)
	buf := bytes.NewBuffer(dat)

	err = suite.client.WriteBlock(node.Id, 0, graph.Hash(dat), buf)
	t.Check(err, IsNil)

	blocks := node.Blocks()
	t.Assert(blocks, HasLen, 1)
	t.Check(blocks[0].Hash, Equals, graph.Hash(dat))
	t.Check(blocks[0].Offset, Equals, int64(0))
}

func (suite *ApiClientTestSuite) TestApiClient_RemoveNode_removesNode(t *C) {
	node, err := suite.ng.NewNode("thing.txt", graph.RootNodeId, os.FileMode(0755))
	t.Check(err, IsNil)

	t.Check(suite.client.RemoveNode(node.Id), IsNil)
	t.Check(suite.ng.RootNode.Children(), HasLen, 0)
}

func (suite *ApiClientTestSuite) TestApiClient_CreateNode_createsNode(t *C) {
	nodeInfo := graph.NodeInfo{
		ParentId: graph.RootNodeId,
		Name:     "file.txt",
		Mode:     0755,
	}
	returnedNodeInfo, err := suite.client.CreateNode(nodeInfo)
	t.Check(err, IsNil)

	createdNode := suite.ng.NodeWithId(returnedNodeInfo.Id)
	t.Check(createdNode.Name(), Equals, nodeInfo.Name)
	t.Check(createdNode.Mode(), Equals, os.FileMode(0755))
	t.Check(time.Now().UTC().Sub(createdNode.MTime()) < time.Second, IsTrue)
}

func (suite *ApiClientTestSuite) TestApiClient_UpdateNode_updatesNode(t *C) {
	node, err := suite.ng.NewNode("thing.txt", graph.RootNodeId, os.FileMode(0755))
	t.Check(err, IsNil)

	info := node.NodeInfo()
	info.Mode = 0700
	info.Name = "file.dat"

	err = suite.client.UpdateNode(info)
	t.Check(err, IsNil)

	node = suite.ng.NodeWithId(node.Id)
	t.Check(node.Name(), Equals, "file.dat")
	t.Check(node.Mode(), Equals, os.FileMode(0700))
}

func (suite *ApiClientTestSuite) TestApiClient_do_readsErrorFromBodyOnNonOkStatus(t *C) {
	_, err := suite.client.ListNodes("not-found")
	t.Check(err, ErrorMatches, "^no_such_node => not-found$")
}
