package graph

import (
	"fmt"
	"os"
	"time"

	"github.com/cayleygraph/cayley"
	"github.com/sdcoffey/olympus/graph"
	"github.com/sdcoffey/olympus/graph/testutils"
	. "gopkg.in/check.v1"
)

func (suite *GraphTestSuite) TestNode_NodeInfo(t *C) {
	now := time.Now()

	child, err := makeNode("child.txt", suite.ng.RootNode.Id, now, suite.ng)
	t.Check(err, IsNil)

	info := child.NodeInfo()
	t.Check(info.Id, Equals, child.Id)
	t.Check(info.ParentId, Equals, graph.RootNodeId)
	t.Check(info.Name, Equals, "child.txt")
	t.Check(info.Type, Equals, "text/plain")
	t.Check(info.MTime.Unix(), Equals, now.Unix())
	t.Check(info.Mode, Equals, child.Mode())
}

func (suite *GraphTestSuite) TestName_returnsName(t *C) {
	node, err := suite.ng.NewNode("A cool folder", graph.RootNodeId)
	t.Check(err, IsNil)
	t.Check(node.Name(), Equals, "A cool folder")
}

func (suite *GraphTestSuite) TestExists_returnsTrueIfName(t *C) {
	node, err := suite.ng.NewNode("name", graph.RootNodeId)
	t.Check(err, IsNil)
	t.Check(node.Exists(), Equals, true)
}

func (suite *GraphTestSuite) TestType(t *C) {
	node, err := suite.ng.NewNode("style.css", graph.RootNodeId)
	t.Check(err, IsNil)
	t.Check(node.Type(), Equals, "text/css")
}

func (suite *GraphTestSuite) TestSize(t *C) {
	node, err := suite.ng.NewNode("child", graph.RootNodeId)
	t.Check(err, IsNil)

	t.Check(node.WriteData(testutils.RandDat(graph.MEGABYTE), 0), IsNil)
	t.Check(node.WriteData(testutils.RandDat(1024), graph.MEGABYTE), IsNil)

	t.Check(node.Size(), Equals, int64(graph.MEGABYTE+1024))
}

func (suite *GraphTestSuite) TestMode(t *C) {
	node, err := suite.ng.NewNode("child", graph.RootNodeId)
	t.Check(err, IsNil)
	t.Check(node.Chmod(os.ModeDir), IsNil)
	t.Check(node.Mode(), Equals, os.ModeDir)
}

func (suite *GraphTestSuite) TestIsDir_returnsTrueForCorrectMode(t *C) {
	node, err := suite.ng.NewNode("child", graph.RootNodeId)
	t.Check(err, IsNil)
	t.Check(node.Chmod(os.ModeDir), IsNil)
	t.Check(node.IsDir(), Equals, true)
}

func (suite *GraphTestSuite) TestIsDir_returnsFalseForIncorrectMode(t *C) {
	node, err := suite.ng.NewNode("child", graph.RootNodeId)
	t.Check(err, IsNil)
	t.Check(node.Chmod(123), IsNil)
	t.Check(node.IsDir(), Equals, false)
}

func (suite *GraphTestSuite) TestModTime(t *C) {
	node, err := suite.ng.NewNode("child", graph.RootNodeId)
	t.Check(err, IsNil)
	t.Check(time.Now().Sub(node.MTime()) < time.Second, Equals, true)
}

func (suite *GraphTestSuite) TestChildren_returnsCorrectChildren(t *C) {
	_, err := suite.ng.NewNode("child1", graph.RootNodeId)
	t.Check(err, IsNil)

	childNode2, err := suite.ng.NewNode("child2", graph.RootNodeId)
	t.Check(err, IsNil)
	t.Check(childNode2.Chmod(os.ModeDir), IsNil)

	children := suite.ng.RootNode.Children()
	t.Check(children, HasLen, 2)

	for idx, child := range children {
		t.Check(child.Parent().Id, Equals, graph.RootNodeId)
		if idx == 0 {
			t.Check(child.Name(), Equals, "child1")
		} else {
			t.Check(child.Name(), Equals, "child2")
		}
	}

	_, err = suite.ng.NewNode("child3", childNode2.Id)
	t.Check(err, IsNil)

	children = childNode2.Children()
	t.Check(children, HasLen, 1)
	t.Check(children[0].Parent().Id, Matches, childNode2.Id)
	t.Check(children[0].Name(), Matches, "child3")
}

func (suite *GraphTestSuite) TestParent(t *C) {
	rootNode := suite.ng.RootNode
	t.Check(rootNode.Parent(), IsNil)

	childNode, err := suite.ng.NewNode("child", graph.RootNodeId)
	t.Check(err, IsNil)

	t.Check(childNode.Parent().Id, Matches, rootNode.Id)
	t.Check(childNode.Parent().Name(), Matches, rootNode.Name())
	t.Check(childNode.Parent().Mode(), Equals, rootNode.Mode())
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
	t.Check(err, IsNil)

	assertProperty := func(expected string, actual cayley.Iterator) {
		t.Check(cayley.RawNext(actual), Equals, true)
		t.Check(suite.ng.NameOf(actual.Result()), Equals, expected)
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
	t.Check(err, ErrorMatches, "Cannot add nameless file")
}

func (suite *GraphTestSuite) TestWriteData_writesDataToCorrectBlock(t *C) {
	child, _ := makeNode("child", suite.ng.RootNode.Id, time.Now(), suite.ng)
	dat := testutils.RandDat(1024)
	fingerprint := graph.Hash(dat)

	t.Check(child.WriteData(dat, 0), IsNil)

	blocks := child.Blocks()
	t.Check(blocks, HasLen, 1)
	if len(blocks) > 0 {
		t.Check(blocks[0].Hash, Matches, fingerprint)
	}
}

func (suite *GraphTestSuite) TestWriteData_throwsOnInvalidBlockOffset(t *C) {
	child, _ := makeNode("child", suite.ng.RootNode.Id, time.Now(), suite.ng)
	dat := testutils.RandDat(1024)

	t.Check(child.WriteData(dat, 1), ErrorMatches, fmt.Sprint("1 is not a valid offset for block size ", graph.BLOCK_SIZE))
}

func (suite *GraphTestSuite) TestWriteData_removesExistingFingerprintForOffset(t *C) {
	child, _ := makeNode("child", suite.ng.RootNode.Id, time.Now(), suite.ng)
	dat := testutils.RandDat(1024)

	t.Check(child.WriteData(dat, 0), IsNil)

	dat = testutils.RandDat(1024)
	fingerprint := graph.Hash(dat)
	t.Check(child.WriteData(dat, 0), IsNil)

	it := cayley.StartPath(suite.ng, child.Id).Out("offset-0").BuildIterator()
	t.Check(cayley.RawNext(it), Equals, true)
	t.Check(suite.ng.NameOf(it.Result()), Equals, fingerprint)
}

func (suite *GraphTestSuite) TestWriteData_SizeChanges(t *C) {
	child, _ := makeNode("child", suite.ng.RootNode.Id, time.Now(), suite.ng)
	dat := testutils.RandDat(graph.BLOCK_SIZE)
	t.Check(child.WriteData(dat, 0), IsNil)

	t.Check(child.Size(), Equals, int64(graph.MEGABYTE))

	dat = testutils.RandDat(graph.BLOCK_SIZE)
	t.Check(child.WriteData(dat, graph.BLOCK_SIZE), IsNil)
	t.Check(child.Size(), Equals, int64(graph.MEGABYTE*2))
}

func (suite *GraphTestSuite) TestBlockWithOffset_findsCorrectBlock(t *C) {
	child, _ := makeNode("child", suite.ng.RootNode.Id, time.Now(), suite.ng)
	data := testutils.RandDat(graph.MEGABYTE)
	t.Check(child.WriteData(data, 0), IsNil)

	data2 := testutils.RandDat(graph.MEGABYTE)
	t.Check(child.WriteData(data2, graph.MEGABYTE), IsNil)

	foundBlock := child.BlockWithOffset(0)
	t.Check(string(foundBlock), Equals, graph.Hash(data))

	foundBlock2 := child.BlockWithOffset(graph.MEGABYTE)
	t.Check(string(foundBlock2), Equals, graph.Hash(data2))
}

func (suite *GraphTestSuite) TestBlockWithOffset_returnsEmptyStringForDir(t *C) {
	fingerprint := suite.ng.RootNode.BlockWithOffset(0)
	t.Check(fingerprint, Equals, "")
}

func (suite *GraphTestSuite) TestBlocks_returnsEmptySliceForDir(t *C) {
	blocks := suite.ng.RootNode.Blocks()
	t.Check(blocks, HasLen, 0)
}

func (suite *GraphTestSuite) TestBlocks_returnsCorrectBlocks(t *C) {
	child, err := makeNode("child", suite.ng.RootNode.Id, time.Now(), suite.ng)
	t.Check(err, IsNil)

	block1 := testutils.RandDat(graph.MEGABYTE)
	block2 := testutils.RandDat(graph.MEGABYTE)

	t.Check(child.WriteData(block1, 0), IsNil)
	t.Check(child.WriteData(block2, graph.MEGABYTE), IsNil)

	blocks := child.Blocks()
	t.Assert(blocks, HasLen, 2)

	t.Check(blocks[0].Offset, Equals, int64(0))
	t.Check(blocks[1].Offset, Equals, int64(graph.MEGABYTE))

	t.Check(blocks[0].Hash, Equals, graph.Hash(block1))
	t.Check(blocks[1].Hash, Equals, graph.Hash(block2))
}

func (suite *GraphTestSuite) TestChmod_chmodsSuccessfully(t *C) {
	child, err := makeNode("child", graph.RootNodeId, time.Now(), suite.ng)
	t.Check(err, IsNil)
	t.Check(child.Chmod(os.FileMode(0777)), IsNil)
	t.Check(child.Mode(), Equals, os.FileMode(0777))
}

func (suite *GraphTestSuite) TestRename_renamesSuccessfully(t *C) {
	child, err := makeNode("child", graph.RootNodeId, time.Now(), suite.ng)
	t.Check(err, IsNil)
	t.Check(child.Name(), Equals, "child")

	t.Check(child.Rename("child_renamed"), IsNil)
	t.Check(child.Name(), Equals, "child_renamed")
}

func (suite *GraphTestSuite) TestRename_changesType(t *C) {
	child, err := makeNode("child.txt", graph.RootNodeId, time.Now(), suite.ng)
	t.Check(err, IsNil)

	t.Check("child.txt", Equals, child.Name())
	t.Check("text/plain", Equals, child.Type())

	t.Check(child.Rename("child.json"), IsNil)
	t.Check("child.json", Equals, child.Name())
	t.Check("application/json", Equals, child.Type())
}

func (suite *GraphTestSuite) TestMove_movesSuccessfully(t *C) {
	folder, err := suite.ng.CreateDirectory(graph.RootNodeId, "folder1")
	t.Check(err, IsNil)

	nodeInfo := graph.NodeInfo{
		Name:     "child.txt",
		ParentId: graph.RootNodeId,
		Mode:     0755,
	}

	child, err := suite.ng.NewNodeWithNodeInfo(nodeInfo)

	t.Check(child.Parent().Id, Equals, graph.RootNodeId)

	t.Check(child.Move(folder.Id), IsNil)
	t.Check(child.Parent().Id, Equals, folder.Id)
}

func (suite *GraphTestSuite) TestMove_throwsIfMovingRootNode(t *C) {
	folder, _ := suite.ng.CreateDirectory(graph.RootNodeId, "folder")
	t.Check(suite.ng.RootNode.Move(folder.Id), ErrorMatches, "Cannot move root node")
}

func (suite *GraphTestSuite) TestMove_throwsIfMovingNodeInsideItself(t *C) {
	folder, _ := suite.ng.CreateDirectory(graph.RootNodeId, "folder")
	nestedFolder, _ := suite.ng.CreateDirectory(folder.Id, "_folder")

	t.Check(folder.Move(folder.Id), ErrorMatches, "Cannot move node inside itself")

	// Also shouldn't be able to move a folder into one of it's children
	t.Check(folder.Move(nestedFolder.Id), ErrorMatches, "Cannot move node inside itself")
}

func (suite *GraphTestSuite) TestChmod_throwsIfNewModeIsDirAndHasSize(t *C) {
	child, err := makeNode("child", graph.RootNodeId, time.Now(), suite.ng)
	t.Check(err, IsNil)
	t.Check(child.WriteData(testutils.RandDat(graph.MEGABYTE), 0), IsNil)

	t.Check(child.Chmod(os.ModeDir), ErrorMatches, "File has size, cannot change to directory")
}

func (suite *GraphTestSuite) TestTouch_updatesMTime(t *C) {
	then := time.Now().Add(-10 * time.Second)
	child, _ := makeNode("child", suite.ng.RootNode.Id, then, suite.ng)

	now := time.Now()
	t.Check(child.Touch(now), IsNil)
	t.Check(child.MTime().Unix(), Equals, now.Unix())
}

func (suite *GraphTestSuite) TestTouch_throwsIfDateInFuture(t *C) {
	child, err := makeNode("child", suite.ng.RootNode.Id, time.Now(), suite.ng)
	t.Check(err, IsNil)
	t.Check(child.Touch(time.Now().Add(time.Second)), ErrorMatches, "Cannot set modified time in the future")
}

func (suite *GraphTestSuite) TestNodeSeeker_readsCorrectData(t *C) {
	child, _ := makeNode("child", suite.ng.RootNode.Id, time.Now(), suite.ng)
	dat := testutils.RandDat(1024)

	t.Check(child.WriteData(dat, 0), IsNil)

	nodeSeeker := child.ReadSeeker()
	offset, err := nodeSeeker.Seek(0, 0)
	t.Check(err, IsNil)
	t.Check(offset, Equals, int64(0))

	p := make([]byte, 1)
	nodeSeeker.Read(p) // expect 1 byte to be read from front of file

	t.Check(p[0], Equals, dat[0])

	offset, err = nodeSeeker.Seek(512, 0)
	t.Check(err, IsNil)
	t.Check(offset, Equals, int64(512))

	p = make([]byte, 25)
	nodeSeeker.Read(p)
	t.Check(p, DeepEquals, dat[offset:int(offset)+len(p)])
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
	t.Check(err, IsNil)

	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		node.Name()
	}
}

func (suite *GraphTestSuite) BenchmarkSize(t *C) {
	node, err := suite.ng.NewNode("child", graph.RootNodeId)
	t.Check(err, IsNil)

	for i := 0; i < 10; i++ {
		err := node.WriteData(testutils.RandDat(graph.MEGABYTE), int64(i*graph.MEGABYTE))
		t.Check(err, IsNil)
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
		Mode:     0755,
	}
	if node, err := ng.NewNodeWithNodeInfo(nodeInfo); err != nil {
		return nil, err
	} else {
		return node, nil
	}
}
