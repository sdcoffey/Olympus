package graph

import (
	"fmt"
	"os"
	"time"

	"github.com/sdcoffey/olympus/Godeps/_workspace/src/github.com/google/cayley"
	"github.com/sdcoffey/olympus/Godeps/_workspace/src/github.com/stretchr/testify/assert"
	"github.com/sdcoffey/olympus/graph"
	. "gopkg.in/check.v1"
)

func (suite *GraphTestSuite) TestNode_NodeInfo(t *C) {
	now := time.Now()

	child, err := makeNode("child.txt", suite.ng.RootNode.Id, now, suite.ng)
	assert.NoError(t, err)

	info := child.NodeInfo()
	assert.Equal(t, child.Id, info.Id)
	assert.Equal(t, suite.ng.RootNode.Id, info.ParentId)
	assert.Equal(t, "child.txt", info.Name)
	assert.Equal(t, "text/plain", info.Type)
	assert.Equal(t, now.Unix(), info.MTime.Unix())
	assert.EqualValues(t, child.Mode(), info.Mode)
}

func (suite *GraphTestSuite) TestName_returnsName(t *C) {
	node, err := suite.ng.NewNode("A cool folder", graph.RootNodeId)
	assert.NoError(t, err)

	assert.Equal(t, "A cool folder", node.Name())
}

func (suite *GraphTestSuite) TestExists_returnsTrueIfName(t *C) {
	node, err := suite.ng.NewNode("name", graph.RootNodeId)
	assert.NoError(t, err)
	assert.True(t, node.Exists())
}

func (suite *GraphTestSuite) TestType(t *C) {
	node, err := suite.ng.NewNode("style.css", graph.RootNodeId)
	assert.NoError(t, err)
	assert.Equal(t, "text/css", node.Type())
}

func (suite *GraphTestSuite) TestSize(t *C) {
	node, err := suite.ng.NewNode("child", graph.RootNodeId)
	assert.NoError(t, err)

	assert.NoError(t, node.WriteData(RandDat(graph.MEGABYTE), 0))
	assert.NoError(t, node.WriteData(RandDat(1024), graph.MEGABYTE))

	assert.EqualValues(t, graph.MEGABYTE+1024, node.Size())
}

func (suite *GraphTestSuite) TestMode(t *C) {
	node, err := suite.ng.NewNode("child", graph.RootNodeId)
	assert.NoError(t, err)
	assert.NoError(t, node.Chmod(os.ModeDir))
	assert.EqualValues(t, os.ModeDir, node.Mode())
}

func (suite *GraphTestSuite) TestIsDir_returnsTrueForCorrectMode(t *C) {
	node, err := suite.ng.NewNode("child", graph.RootNodeId)
	assert.NoError(t, err)
	assert.NoError(t, node.Chmod(os.ModeDir))

	assert.True(t, node.IsDir())
}

func (suite *GraphTestSuite) TestIsDir_returnsFalseForIncorrectMode(t *C) {
	node, err := suite.ng.NewNode("child", graph.RootNodeId)
	assert.NoError(t, err)
	assert.NoError(t, node.Chmod(123))

	assert.False(t, node.IsDir())
}

func (suite *GraphTestSuite) TestModTime(t *C) {
	node, err := suite.ng.NewNode("child", graph.RootNodeId)
	assert.NoError(t, err)

	assert.True(t, time.Now().Sub(node.MTime()) < time.Second, Equals, true)
}

func (suite *GraphTestSuite) TestChildren_returnsCorrectChildren(t *C) {
	_, err := suite.ng.NewNode("child1", graph.RootNodeId)
	assert.NoError(t, err)

	childNode2, err := suite.ng.NewNode("child2", graph.RootNodeId)
	assert.NoError(t, err)
	assert.NoError(t, childNode2.Chmod(os.ModeDir))

	children := suite.ng.RootNode.Children()
	assert.EqualValues(t, 2, len(children))

	for idx, child := range children {
		assert.Equal(t, graph.RootNodeId, child.Parent().Id)
		if idx == 0 {
			assert.Equal(t, "child1", child.Name())
		} else {
			assert.Equal(t, "child2", child.Name())
		}
	}

	_, err = suite.ng.NewNode("child3", childNode2.Id)
	assert.NoError(t, err)

	children = childNode2.Children()
	assert.EqualValues(t, 1, len(children))
	assert.EqualValues(t, childNode2.Id, children[0].Parent().Id)
	assert.EqualValues(t, "child3", children[0].Name())
}

func (suite *GraphTestSuite) TestParent(t *C) {
	rootNode := suite.ng.RootNode

	assert.Nil(t, rootNode.Parent())

	childNode, err := suite.ng.NewNode("child", graph.RootNodeId)
	assert.NoError(t, err)

	assert.Equal(t, rootNode.Id, childNode.Parent().Id)
	assert.Equal(t, rootNode.Name(), childNode.Parent().Name())
	assert.EqualValues(t, rootNode.Mode(), childNode.Parent().Mode())
}

func (suite *GraphTestSuite) TestSave(t *C) {
	mTime := time.Now()

	ni := graph.NodeInfo{
		Name:     "child",
		ParentId: graph.RootNodeId,
		Mode:     os.FileMode(0755),
		MTime:    mTime,
		Type:     "application/json",
	}

	node, err := suite.ng.NewNodeWithNodeInfo(ni)
	assert.NoError(t, err)

	assertProperty := func(expected string, actual cayley.Iterator) {
		assert.True(t, cayley.RawNext(actual))
		assert.Equal(t, expected, suite.ng.NameOf(actual.Result()))
	}

	it := cayley.StartPath(suite.ng, node.Id).Out("isNamed").BuildIterator()
	assertProperty("child", it)

	it = cayley.StartPath(suite.ng, node.Id).Out("hasMTime").BuildIterator()
	assertProperty(mTime.Format(time.RFC3339Nano), it)

	it = cayley.StartPath(suite.ng, node.Id).Out("hasMode").BuildIterator()
	assertProperty(fmt.Sprint(0755), it)

	it = cayley.StartPath(suite.ng, node.Id).Out("hasType").BuildIterator()
	assertProperty("application/json", it)
}

func (suite *GraphTestSuite) TestSave_returnsErrorWhenFileHasNoName(t *C) {
	_, err := suite.ng.NewNode("", graph.RootNodeId)
	assert.EqualError(t, err, "Cannot add nameless file")
}

func (suite *GraphTestSuite) TestWriteData_writesDataToCorrectBlock(t *C) {
	child, _ := makeNode("child", suite.ng.RootNode.Id, time.Now(), suite.ng)
	dat := RandDat(1024)
	fingerprint := graph.Hash(dat)

	assert.NoError(t, child.WriteData(dat, 0))

	blocks := child.Blocks()
	assert.Len(t, blocks, 1)
	if len(blocks) > 0 {
		assert.Equal(t, fingerprint, blocks[0].Hash)
	}
}

func (suite *GraphTestSuite) TestWriteData_throwsOnInvalidBlockOffset(t *C) {
	child, _ := makeNode("child", suite.ng.RootNode.Id, time.Now(), suite.ng)
	dat := RandDat(1024)

	assert.EqualError(t, child.WriteData(dat, 1), fmt.Sprint("1 is not a valid offset for block size ", graph.BLOCK_SIZE))
}

func (suite *GraphTestSuite) TestWriteData_removesExistingFingerprintForOffset(t *C) {
	child, _ := makeNode("child", suite.ng.RootNode.Id, time.Now(), suite.ng)
	dat := RandDat(1024)

	assert.NoError(t, child.WriteData(dat, 0))

	dat = RandDat(1024)
	fingerprint := graph.Hash(dat)
	assert.NoError(t, child.WriteData(dat, 0))

	it := cayley.StartPath(suite.ng, child.Id).Out("offset-0").BuildIterator()
	assert.True(t, cayley.RawNext(it))
	assert.Equal(t, fingerprint, suite.ng.NameOf(it.Result()))
}

func (suite *GraphTestSuite) TestWriteData_SizeChanges(t *C) {
	child, _ := makeNode("child", suite.ng.RootNode.Id, time.Now(), suite.ng)
	dat := RandDat(graph.BLOCK_SIZE)
	assert.NoError(t, child.WriteData(dat, 0))

	assert.EqualValues(t, graph.MEGABYTE, child.Size())

	dat = RandDat(graph.BLOCK_SIZE)
	assert.NoError(t, child.WriteData(dat, graph.BLOCK_SIZE))

	assert.EqualValues(t, graph.MEGABYTE*2, child.Size())
}

func (suite *GraphTestSuite) TestBlockWithOffset_findsCorrectBlock(t *C) {
	child, _ := makeNode("child", suite.ng.RootNode.Id, time.Now(), suite.ng)
	data := RandDat(graph.MEGABYTE)
	assert.NoError(t, child.WriteData(data, 0))

	data2 := RandDat(graph.MEGABYTE)
	assert.NoError(t, child.WriteData(data2, graph.MEGABYTE))

	foundBlock := child.BlockWithOffset(0)
	assert.Equal(t, graph.Hash(data), string(foundBlock))

	foundBlock2 := child.BlockWithOffset(graph.MEGABYTE)
	assert.Equal(t, graph.Hash(data2), string(foundBlock2))
}

func (suite *GraphTestSuite) TestBlockWithOffset_returnsEmptyStringForDir(t *C) {
	fingerprint := suite.ng.RootNode.BlockWithOffset(0)
	assert.Equal(t, "", fingerprint)
}

func (suite *GraphTestSuite) TestBlocks_returnsEmptySliceForDir(t *C) {
	blocks := suite.ng.RootNode.Blocks()
	assert.Len(t, blocks, 0)
}

func (suite *GraphTestSuite) TestBlocks_returnsCorrectBlocks(t *C) {
	child, err := makeNode("child", suite.ng.RootNode.Id, time.Now(), suite.ng)
	assert.NoError(t, err)

	block1 := RandDat(graph.MEGABYTE)
	block2 := RandDat(graph.MEGABYTE)

	assert.NoError(t, child.WriteData(block1, 0))
	assert.NoError(t, child.WriteData(block2, graph.MEGABYTE))

	blocks := child.Blocks()
	assert.Len(t, blocks, 2)

	assert.EqualValues(t, 0, blocks[0].Offset)
	assert.EqualValues(t, graph.MEGABYTE, blocks[1].Offset)

	assert.Equal(t, graph.Hash(block1), blocks[0].Hash)
	assert.Equal(t, graph.Hash(block2), blocks[1].Hash)
}

func (suite *GraphTestSuite) TestChmod_chmodsSuccessfully(t *C) {
	child, err := makeNode("child", graph.RootNodeId, time.Now(), suite.ng)
	assert.NoError(t, err)

	assert.NoError(t, child.Chmod(os.FileMode(0777)))
	assert.EqualValues(t, os.FileMode(0777), child.Mode())
}

func (suite *GraphTestSuite) TestRename_renamesSuccessfully(t *C) {
	child, err := makeNode("child", graph.RootNodeId, time.Now(), suite.ng)
	assert.NoError(t, err)

	assert.Equal(t, "child", child.Name())

	assert.NoError(t, child.Rename("child_renamed"))
	assert.Equal(t, "child_renamed", child.Name())
}

func (suite *GraphTestSuite) TestRename_changesType(t *C) {
	child, err := makeNode("child.txt", graph.RootNodeId, time.Now(), suite.ng)
	assert.NoError(t, err)

	assert.Equal(t, "child.txt", child.Name())
	assert.Equal(t, "text/plain", child.Type())

	assert.NoError(t, child.Rename("child.json"))
	assert.Equal(t, "child.json", child.Name())
	assert.Equal(t, "application/json", child.Type())
}

func (suite *GraphTestSuite) TestMove_movesSuccessfully(t *C) {
	folder, err := suite.ng.CreateDirectory(graph.RootNodeId, "folder1")
	assert.NoError(t, err)

	nodeInfo := graph.NodeInfo{
		Name:     "child.txt",
		ParentId: graph.RootNodeId,
		Mode:     0755,
	}

	child, err := suite.ng.NewNodeWithNodeInfo(nodeInfo)
	assert.NoError(t, err)

	assert.Equal(t, graph.RootNodeId, child.Parent().Id)

	assert.NoError(t, child.Move(folder.Id))
	assert.Equal(t, folder.Id, child.Parent().Id)
}

func (suite *GraphTestSuite) TestMove_throwsIfMovingRootNode(t *C) {
	folder, _ := suite.ng.CreateDirectory(graph.RootNodeId, "folder")
	assert.EqualError(t, suite.ng.RootNode.Move(folder.Id), "Cannot move root node")
}

func (suite *GraphTestSuite) TestMove_throwsIfMovingNodeInsideItself(t *C) {
	folder, _ := suite.ng.CreateDirectory(graph.RootNodeId, "folder")
	nestedFolder, _ := suite.ng.CreateDirectory(folder.Id, "_folder")

	assert.EqualError(t, folder.Move(folder.Id), "Cannot move node inside itself")

	// Also shouldn't be able to move a folder into one of it's children
	assert.EqualError(t, folder.Move(nestedFolder.Id), "Cannot move node inside itself")
}

func (suite *GraphTestSuite) TestChmod_throwsIfNewModeIsDirAndHasSize(t *C) {
	child, err := makeNode("child", graph.RootNodeId, time.Now(), suite.ng)
	assert.NoError(t, err)
	assert.NoError(t, child.WriteData(RandDat(graph.MEGABYTE), 0))

	assert.EqualError(t, child.Chmod(os.ModeDir), "File has size, cannot change to directory")
}

func (suite *GraphTestSuite) TestTouch_updatesMTime(t *C) {
	then := time.Now().Add(-10 * time.Second)
	child, _ := makeNode("child", suite.ng.RootNode.Id, then, suite.ng)

	now := time.Now()
	assert.NoError(t, child.Touch(now))

	assert.EqualValues(t, now.Unix(), child.MTime().Unix())
}

func (suite *GraphTestSuite) TestTouch_throwsIfDateInFuture(t *C) {
	child, err := makeNode("child", suite.ng.RootNode.Id, time.Now(), suite.ng)
	assert.NoError(t, err)
	assert.EqualError(t, child.Touch(time.Now().Add(time.Second)), "Cannot set modified time in the future")
}

func (suite *GraphTestSuite) TestNodeSeeker_readsCorrectData(t *C) {
	child, _ := makeNode("child", suite.ng.RootNode.Id, time.Now(), suite.ng)
	dat := RandDat(1024)

	assert.NoError(t, child.WriteData(dat, 0))

	nodeSeeker := child.ReadSeeker()
	offset, err := nodeSeeker.Seek(0, 0)
	assert.NoError(t, err)
	assert.EqualValues(t, 0, offset)

	p := make([]byte, 1)
	nodeSeeker.Read(p) // expect 1 byte to be read from front of file

	assert.EqualValues(t, dat[0], p[0])

	offset, err = nodeSeeker.Seek(512, 0)
	assert.NoError(t, err)
	assert.EqualValues(t, 512, offset)

	p = make([]byte, 25)
	nodeSeeker.Read(p)
	assert.Equal(t, dat[offset:int(offset)+len(p)], p)
}

func (suite *GraphTestSuite) BenchmarkWrite(t *C) {
	var err error
	var info graph.NodeInfo

	info.ParentId = graph.RootNodeId
	info.Mode = os.FileMode(0755)
	info.Type = "application/json"
	for i := 0; err == nil && i < t.N; i++ {
		info.Name = fmt.Sprint(i)
		_, err = suite.ng.NewNodeWithNodeInfo(info)
	}

	if err != nil {
		panic(err)
	}
}

func (suite *GraphTestSuite) BenchmarkName(t *C) {
	node, err := suite.ng.NewNode("child", graph.RootNodeId)
	assert.NoError(t, err)

	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		node.Name()
	}
}

func (suite *GraphTestSuite) BenchmarkSize(t *C) {
	node, err := suite.ng.NewNode("child", graph.RootNodeId)
	assert.NoError(t, err)

	for i := 0; i < 10; i++ {
		err := node.WriteData(RandDat(graph.MEGABYTE), int64(i*graph.MEGABYTE))
		assert.NoError(t, err)
	}

	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		node.Size()
	}
}

func makeNode(name, parentId string, mTime time.Time, ng *graph.NodeGraph) (*graph.Node, error) {
	nodeInfo := graph.NodeInfo{
		Name:     name,
		ParentId: parentId,
		MTime:    mTime,
	}
	if node, err := ng.NewNodeWithNodeInfo(nodeInfo); err != nil {
		return nil, err
	} else {
		return node, nil
	}
}
