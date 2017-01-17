package graph

import (
	"fmt"
	"os"
	"time"

	"github.com/cayleygraph/cayley"
	"github.com/cayleygraph/cayley/quad"
	. "github.com/sdcoffey/olympus/checkers"
	"github.com/sdcoffey/olympus/graph"
	"github.com/sdcoffey/olympus/graph/testutils"
	. "gopkg.in/check.v1"
)

func (suite *GraphTestSuite) TestNode_NodeInfo(t *C) {
	child, err := suite.ng.NewNode("child.txt", graph.RootNodeId, os.ModeDir)
	t.Check(err, IsNil)

	info := child.NodeInfo()
	t.Check(info.Id, Equals, child.Id)
	t.Check(info.ParentId, Equals, graph.RootNodeId)
	t.Check(info.Name, Equals, "child.txt")
	t.Check(info.Type, Equals, "text/plain")
	t.Check(info.MTime, WithinNow, time.Second)
	t.Check(info.Mode, Equals, child.Mode())
}

func (suite *GraphTestSuite) TestName_returnsName(t *C) {
	node, err := suite.ng.NewNode("A cool folder", graph.RootNodeId, os.FileMode(0755))
	t.Check(err, IsNil)
	t.Check(node.Name(), Equals, "A cool folder")
}

func (suite *GraphTestSuite) TestExists_returnsTrueIfName(t *C) {
	node, err := suite.ng.NewNode("name", graph.RootNodeId, os.FileMode(0755))
	t.Check(err, IsNil)
	t.Check(node.Exists(), Equals, true)
}

func (suite *GraphTestSuite) TestType(t *C) {
	node, err := suite.ng.NewNode("style.css", graph.RootNodeId, os.FileMode(0755))
	t.Check(err, IsNil)
	t.Check(node.Type(), Equals, "text/css")
}

func (suite *GraphTestSuite) TestSize(t *C) {
	node, err := suite.ng.NewNode("child", graph.RootNodeId, os.FileMode(0755))
	t.Check(err, IsNil)

	t.Check(node.WriteData(testutils.RandDat(graph.MEGABYTE), 0), IsNil)
	t.Check(node.WriteData(testutils.RandDat(1024), graph.MEGABYTE), IsNil)

	t.Check(node.Size(), Equals, int64(graph.MEGABYTE+1024))
}

func (suite *GraphTestSuite) TestMode(t *C) {
	node, err := suite.ng.NewNode("child", graph.RootNodeId, os.FileMode(0755))
	t.Check(err, IsNil)
	t.Check(node.SetMode(os.ModeDir), IsNil)
	t.Check(node.Mode(), Equals, os.ModeDir)
}

func (suite *GraphTestSuite) TestIsDir_returnsTrueForCorrectMode(t *C) {
	node, err := suite.ng.NewNode("child", graph.RootNodeId, os.ModeDir)
	t.Check(err, IsNil)
	t.Check(node.IsDir(), Equals, true)
}

func (suite *GraphTestSuite) TestIsDir_returnsFalseForIncorrectMode(t *C) {
	node, err := suite.ng.NewNode("child", graph.RootNodeId, os.FileMode(0755))
	t.Check(err, IsNil)
	t.Check(node.SetMode(123), IsNil)
	t.Check(node.IsDir(), Equals, false)
}

func (suite *GraphTestSuite) TestModTime(t *C) {
	node, err := suite.ng.NewNode("child", graph.RootNodeId, os.FileMode(0755))
	t.Check(err, IsNil)
	t.Check(time.Now().Sub(node.MTime()) < time.Second, IsTrue)
}

func (suite *GraphTestSuite) TestChildren_returnsCorrectChildren(t *C) {
	_, err := suite.ng.NewNode("child1", graph.RootNodeId, os.FileMode(0755))
	t.Check(err, IsNil)

	childNode2, err := suite.ng.NewNode("child2", graph.RootNodeId, os.FileMode(0755))
	t.Check(err, IsNil)
	t.Check(childNode2.SetMode(os.ModeDir), IsNil)

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

	_, err = suite.ng.NewNode("child3", childNode2.Id, os.FileMode(0755))
	t.Check(err, IsNil)

	children = childNode2.Children()
	t.Check(children, HasLen, 1)
	t.Check(children[0].Parent().Id, Matches, childNode2.Id)
	t.Check(children[0].Name(), Matches, "child3")
}

func (suite *GraphTestSuite) TestParent(t *C) {
	rootNode := suite.ng.RootNode
	t.Check(rootNode.Parent(), IsNil)

	childNode, err := suite.ng.NewNode("child", graph.RootNodeId, os.FileMode(0755))
	t.Check(err, IsNil)

	t.Check(childNode.Parent().Id, Matches, rootNode.Id)
	t.Check(childNode.Parent().Name(), Matches, rootNode.Name())
	t.Check(childNode.Parent().Mode(), Equals, rootNode.Mode())
}

func (suite *GraphTestSuite) TestWriteData_writesDataToCorrectBlock(t *C) {
	child, err := suite.ng.NewNode("child", graph.RootNodeId, os.FileMode(0755))
	t.Check(err, IsNil)
	dat := testutils.RandDat(1024)
	fingerprint := graph.Hash(dat)

	t.Check(child.WriteData(dat, 0), IsNil)

	blocks, err := child.Blocks()
	t.Check(err, IsNil)
	t.Check(blocks, HasLen, 1)
	if len(blocks) > 0 {
		t.Check(blocks[0].Hash, Matches, fingerprint)
	}
}

func (suite *GraphTestSuite) TestWriteData_throwsOnInvalidBlockOffset(t *C) {
	child, err := suite.ng.NewNode("child", graph.RootNodeId, os.FileMode(0755))
	t.Check(err, IsNil)
	dat := testutils.RandDat(1024)

	t.Check(child.WriteData(dat, 1), ErrorMatches, fmt.Sprint("1 is not a valid offset for block size ", graph.BLOCK_SIZE))
}

func (suite *GraphTestSuite) TestWriteData_throwsIfNodeIsDir(t *C) {
	child, err := suite.ng.NewNode("child", graph.RootNodeId, os.ModeDir)
	t.Check(err, IsNil)

	dat := testutils.RandDat(graph.BLOCK_SIZE)
	err = child.WriteData(dat, 0)

	t.Check(err, ErrorMatches, "Cannot write data to directory")
}

func (suite *GraphTestSuite) TestWriteData_removesExistingFingerprintForOffset(t *C) {
	child, err := suite.ng.NewNode("child", graph.RootNodeId, os.FileMode(0755))
	t.Check(err, IsNil)
	dat := testutils.RandDat(1024)

	t.Check(child.WriteData(dat, 0), IsNil)

	dat = testutils.RandDat(1024)
	fingerprint := graph.Hash(dat)
	t.Check(child.WriteData(dat, 0), IsNil)

	it := cayley.StartPath(suite.ng, quad.String(child.Id)).Out("offset-0").BuildIterator()
	t.Check(it.Next(), Equals, true)
	t.Check(quad.NativeOf(suite.ng.NameOf(it.Result())), Equals, fingerprint)
}

func (suite *GraphTestSuite) TestWriteData_SizeChanges(t *C) {
	child, err := suite.ng.NewNode("child", graph.RootNodeId, os.FileMode(0755))
	t.Check(err, IsNil)

	dat := testutils.RandDat(graph.BLOCK_SIZE)
	t.Check(child.WriteData(dat, 0), IsNil)

	t.Check(child.Size(), Equals, int64(graph.MEGABYTE))

	dat = testutils.RandDat(graph.BLOCK_SIZE)
	t.Check(child.WriteData(dat, graph.BLOCK_SIZE), IsNil)
	t.Check(child.Size(), Equals, int64(graph.MEGABYTE*2))
}

func (suite *GraphTestSuite) TestBlockWithOffset_findsCorrectBlock(t *C) {
	child, err := suite.ng.NewNode("child", graph.RootNodeId, os.FileMode(0755))
	t.Check(err, IsNil)

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

func (suite *GraphTestSuite) TestBlocks_throwsForDir(t *C) {
	blocks, err := suite.ng.RootNode.Blocks()
	t.Check(err, ErrorMatches, "Cannot fetch blocks for directory")
	t.Check(blocks, IsNil)
}

func (suite *GraphTestSuite) TestBlocks_returnsCorrectBlocks(t *C) {
	child, err := suite.ng.NewNode("child", graph.RootNodeId, os.FileMode(0755))
	t.Check(err, IsNil)

	block1 := testutils.RandDat(graph.MEGABYTE)
	block2 := testutils.RandDat(graph.MEGABYTE)
	block3 := testutils.RandDat(1)

	t.Check(child.WriteData(block1, 0), IsNil)
	t.Check(child.WriteData(block2, graph.MEGABYTE), IsNil)
	t.Check(child.WriteData(block3, graph.MEGABYTE * 2), IsNil)

	blocks, err := child.Blocks()
	t.Assert(err, IsNil)
	t.Assert(blocks, HasLen, 3)

	t.Check(blocks[0].Offset, Equals, int64(0))
	t.Check(blocks[1].Offset, Equals, int64(graph.MEGABYTE))
	t.Check(blocks[2].Offset, Equals, int64(graph.MEGABYTE * 2))

	t.Check(blocks[0].Hash, Equals, graph.Hash(block1))
	t.Check(blocks[1].Hash, Equals, graph.Hash(block2))
	t.Check(blocks[2].Hash, Equals, graph.Hash(block3))

	t.Check(blocks[0].Size, Equals, int64(graph.MEGABYTE))
	t.Check(blocks[1].Size, Equals, int64(graph.MEGABYTE))
	t.Check(blocks[2].Size, Equals, int64(1))
}

func (suite *GraphTestSuite) TestChmod_chmodsSuccessfully(t *C) {
	child, err := suite.ng.NewNode("child", graph.RootNodeId, os.FileMode(0755))
	t.Check(err, IsNil)

	t.Check(child.SetMode(os.FileMode(0777)), IsNil)
	t.Check(child.Mode(), Equals, os.FileMode(0777))
}

func (suite *GraphTestSuite) TestSetName_SetsNameSuccessfully(t *C) {
	child, err := suite.ng.NewNode("child", graph.RootNodeId, os.FileMode(0755))
	t.Check(err, IsNil)

	t.Check(child.Name(), Equals, "child")

	t.Check(child.SetName("child_renamed"), IsNil)
	t.Check(child.Name(), Equals, "child_renamed")
}

func (suite *GraphTestSuite) TestSetName_returnsErrorWhenFileHasNoName(t *C) {
	nd, err := suite.ng.NewNode("child", graph.RootNodeId, os.FileMode(0755))
	t.Check(err, IsNil)

	err = nd.SetName("")
	t.Check(err, ErrorMatches, ".*name cannot be blank")
}

func (suite *GraphTestSuite) TestSetName_throwsIfRenamingRootNode(t *C) {
	err := suite.ng.RootNode.SetName("notRoot")
	t.Check(err, ErrorMatches, "^.*cannot rename root node")
}

func (suite *GraphTestSuite) TestSetName_throwsIfSiblingWithSameName(t *C) {
	_, err := suite.ng.NewNode("child", graph.RootNodeId, os.FileMode(0755))
	t.Check(err, IsNil)

	child2, err := suite.ng.NewNode("child2", graph.RootNodeId, os.FileMode(0755))
	t.Check(err, IsNil)

	err = child2.SetName("child")
	t.Check(err, ErrorMatches, "^.*Node with name .* already exists in .*$")
}

func (suite *GraphTestSuite) TestMove_movesSuccessfully(t *C) {
	folder, err := suite.ng.NewNode("folder1", graph.RootNodeId, os.ModeDir)
	t.Check(err, IsNil)

	child, err := suite.ng.NewNode("child.txt", graph.RootNodeId, os.FileMode(0755))

	t.Check(child.Parent().Id, Equals, graph.RootNodeId)

	t.Check(child.Move(folder.Id), IsNil)
	t.Check(child.Parent().Id, Equals, folder.Id)
}

func (suite *GraphTestSuite) TestMove_throwsIfMovingRootNode(t *C) {
	folder, _ := suite.ng.NewNode("folder", graph.RootNodeId, os.ModeDir)
	t.Check(suite.ng.RootNode.Move(folder.Id), ErrorMatches, "^.*Cannot move root node")
}

func (suite *GraphTestSuite) TestMove_throwsIfMovingNodeInsideItself(t *C) {
	folder, _ := suite.ng.NewNode("folder", graph.RootNodeId, os.ModeDir)
	nestedFolder, _ := suite.ng.NewNode("_folder", folder.Id, os.FileMode(0755))

	t.Check(folder.Move(folder.Id), ErrorMatches, ".*Cannot move node inside itself")

	// Also shouldn't be able to move a folder into one of it's children
	t.Check(folder.Move(nestedFolder.Id), ErrorMatches, ".*Cannot move node inside itself")
}

func (suite *GraphTestSuite) TestMove_throwsIfMovingFolderIntoNonDir(t *C) {
	node1, _ := suite.ng.NewNode("node", graph.RootNodeId, os.FileMode(0755))
	node2, _ := suite.ng.NewNode("node2", graph.RootNodeId, os.FileMode(0755))

	err := node2.Move(node1.Id)
	t.Check(err, ErrorMatches, ".*Cannot add node to a non-directory")
}

func (suite *GraphTestSuite) TestMove_throwsIfParentDoesNotExist(t *C) {
	node, err := suite.ng.NewNode("node", graph.RootNodeId, os.FileMode(0755))
	t.Check(err, IsNil)

	err = node.Move("not-an-id")
	t.Check(err, ErrorMatches, ".*Parent does not exist")
}

func (suite *GraphTestSuite) TestMove_returnsAnErrorWhenDuplicateSiblingExists(t *C) {
	firstChild, err := suite.ng.NewNode("firstChild", graph.RootNodeId, os.ModeDir)
	t.Check(err, IsNil)

	secondChild, err := suite.ng.NewNode("firstChild", firstChild.Id, os.ModeDir)
	t.Check(err, IsNil)

	err = secondChild.Move(graph.RootNodeId)
	t.Check(err, ErrorMatches, "^.*Node with name .* already exists in root")
}

func (suite *GraphTestSuite) TestSetMode_throwsIfNewModeIsDirAndHasSize(t *C) {
	child, err := suite.ng.NewNode("child", graph.RootNodeId, os.FileMode(0755))
	t.Check(err, IsNil)

	t.Check(child.WriteData(testutils.RandDat(graph.MEGABYTE), 0), IsNil)
	t.Check(child.SetMode(os.ModeDir), ErrorMatches, "File has size, cannot change to directory")
}

func (suite *GraphTestSuite) TestTouch_ignoredIfTimeIsZero(t *C) {
	child, err := suite.ng.NewNode("child", graph.RootNodeId, os.FileMode(0755))
	t.Check(err, IsNil)

	err = child.Touch(time.Time{})
	t.Check(err, IsNil)

	t.Check(child.MTime(), WithinNow, time.Second)
}

func (suite *GraphTestSuite) TestTouch_updatesMTime(t *C) {
	child, err := suite.ng.NewNode("child", graph.RootNodeId, os.FileMode(0755))
	t.Check(err, IsNil)

	then := time.Now().Add(-10 * time.Second)
	t.Check(child.Touch(then), IsNil)
	t.Check(child.MTime(), EqualTime, then, time.Second)
}

func (suite *GraphTestSuite) TestTouch_throwsIfDateInFuture(t *C) {
	child, err := suite.ng.NewNode("child", graph.RootNodeId, os.FileMode(0755))
	t.Check(err, IsNil)

	t.Check(child.Touch(time.Now().Add(time.Second)), ErrorMatches, "Cannot set modified time in the future")
}

func (suite *GraphTestSuite) TestNodeSeeker_readsCorrectData(t *C) {
	child, err := suite.ng.NewNode("child", graph.RootNodeId, os.FileMode(0755))
	t.Check(err, IsNil)

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

	for i := 0; err == nil && i < t.N; i++ {
		_, err = suite.ng.NewNode(fmt.Sprint(i), graph.RootNodeId, os.FileMode(0755))
	}

	if err != nil {
		panic(err)
	}
}

func (suite *GraphTestSuite) BenchmarkName(t *C) {
	node, err := suite.ng.NewNode("child", graph.RootNodeId, os.FileMode(0755))
	t.Check(err, IsNil)

	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		node.Name()
	}
}

func (suite *GraphTestSuite) BenchmarkSize(t *C) {
	node, err := suite.ng.NewNode("child", graph.RootNodeId, os.FileMode(0755))
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
