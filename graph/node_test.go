package graph

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/google/cayley"
	"github.com/sdcoffey/olympus/env"
	"github.com/stretchr/testify/assert"
)

func TestNewNode_hasUuidAndTimeStamp(t *testing.T) {
	ng := testInit()

	node := newNode("root", ng)
	assert.NotEmpty(t, node.Id)
	assert.NotEmpty(t, node.mTime)
	assert.True(t, time.Now().Sub(node.mTime) < time.Second)
}

func TestNodeInfo(t *testing.T) {
	ng := testInit()

	now := time.Now()

	child, err := makeNode("child", ng.RootNode.Id, 1024, now, ng)
	assert.NoError(t, err)
	child.mimeType = "application/octet-stream"
	child.Save()

	info := child.NodeInfo()
	assert.Equal(t, child.Id, info.Id)
	assert.Equal(t, ng.RootNode.Id, info.ParentId)
	assert.Equal(t, "child", info.Name)
	assert.Equal(t, "application/octet-stream", info.Type)
	assert.EqualValues(t, 1024, info.Size)
	assert.Equal(t, now.Unix(), info.MTime.Unix())
	assert.EqualValues(t, child.Mode(), info.Mode)
}

func TestName(t *testing.T) {
	ng := testInit()

	node := newNode("child", ng)
	node.parentId = ng.RootNode.Id
	node.name = "A cool folder"
	node.Save()

	assert.Equal(t, "A cool folder", node.Name())
}

func TestExists_returnsFalseIfNoName(t *testing.T) {
	ng := testInit()

	node := newNode("", ng)
	assert.False(t, node.Exists())
}

func TestExists_returnsTrueIfName(t *testing.T) {
	ng := testInit()

	node := newNode("name", ng)
	node.parentId = ng.RootNode.Id
	assert.NoError(t, node.Save())

	assert.True(t, node.Exists())
}

func TestType(t *testing.T) {
	ng := testInit()

	node := newNode("style.css", ng)
	node.parentId = ng.RootNode.Id
	node.mimeType = "text/css"
	node.size = 100
	assert.NoError(t, node.Save())

	assert.EqualValues(t, "text/css", node.Type())
}

func TestSize(t *testing.T) {
	ng := testInit()

	node := newNode("child", ng)
	node.parentId = ng.RootNode.Id
	node.size = 1024
	assert.NoError(t, node.Save())

	assert.EqualValues(t, 1024, node.Size())
}

func TestMode(t *testing.T) {
	ng := testInit()

	node := newNode("child", ng)
	node.parentId = ng.RootNode.Id
	node.mode = os.ModeDir
	assert.NoError(t, node.Save())

	assert.EqualValues(t, os.ModeDir, node.Mode())
}

func TestIsDir_returnsTrueForCorrectMode(t *testing.T) {
	ng := testInit()

	node := newNode("child", ng)
	node.mode = os.ModeDir
	node.parentId = ng.RootNode.Id
	assert.NoError(t, node.Save())

	assert.True(t, node.IsDir())
}

func TestIsDir_returnsFalseForIncorrectMode(t *testing.T) {
	ng := testInit()

	node := newNode("child", ng)
	node.mode = 123
	node.parentId = ng.RootNode.Id
	assert.NoError(t, node.Save())

	assert.False(t, node.IsDir())
}

func TestModTime(t *testing.T) {
	ng := testInit()

	node := newNode("child", ng)
	node.mTime = time.Now()
	node.parentId = ng.RootNode.Id
	assert.NoError(t, node.Save())

	assert.NotEmpty(t, node.MTime())
}

func TestChildren_returnsCorrectChildren(t *testing.T) {
	ng := testInit()

	childNode1 := newNode("child1", ng)
	childNode1.parentId = ng.RootNode.Id
	childNode1.Save()

	childNode2 := newNode("child2", ng)
	childNode2.mode |= os.ModeDir
	childNode2.parentId = ng.RootNode.Id
	childNode2.Save()

	children := ng.RootNode.Children()
	assert.EqualValues(t, 2, len(children))

	for idx, child := range children {
		assert.Equal(t, ng.RootNode.Id, child.parentId)
		if idx == 0 {
			assert.Equal(t, "child1", child.Name())
		} else {
			assert.Equal(t, "child2", child.Name())
		}
	}

	childNode3 := newNode("child3", ng)
	childNode3.parentId = childNode2.Id
	assert.NoError(t, childNode3.Save())

	children = childNode2.Children()
	assert.EqualValues(t, 1, len(children))
	assert.EqualValues(t, childNode2.Id, children[0].parentId)
	assert.EqualValues(t, "child3", children[0].Name())
}

func TestParent(t *testing.T) {
	ng := testInit()

	rootNode := ng.RootNode

	assert.Nil(t, rootNode.Parent())

	childNode := newNode("child", ng)
	childNode.parentId = rootNode.Id
	assert.NoError(t, childNode.Save())

	assert.Equal(t, rootNode.Id, childNode.Parent().Id)
	assert.Equal(t, rootNode.Name(), childNode.Parent().Name())
	assert.EqualValues(t, rootNode.Mode(), childNode.Parent().Mode())
}

func TestSave(t *testing.T) {
	ng := testInit()

	mTime := time.Now()

	node := newNode("child", ng)
	node.parentId = ng.RootNode.Id
	node.size = 1024
	node.mode = os.FileMode(0755)
	node.mTime = mTime
	node.mimeType = "application/json"

	assert.NoError(t, node.Save())

	assertProperty := func(expected string, actual cayley.Iterator) {
		assert.True(t, cayley.RawNext(actual))
		assert.Equal(t, expected, ng.NameOf(actual.Result()))
	}

	it := cayley.StartPath(ng, node.Id).Out(nameLink).BuildIterator()
	assertProperty("child", it)

	it = cayley.StartPath(ng, node.Id).Out(sizeLink).BuildIterator()
	assertProperty("1024", it)

	it = cayley.StartPath(ng, node.Id).Out(mTimeLink).BuildIterator()
	assertProperty(mTime.Format(timeFormat), it)

	it = cayley.StartPath(ng, node.Id).Out(modeLink).BuildIterator()
	assertProperty(fmt.Sprint(0755), it)

	it = cayley.StartPath(ng, node.Id).Out(typeLink).BuildIterator()
	assertProperty("application/json", it)
}

func TestSave_overwriteExistingProperty(t *testing.T) {
	ng := testInit()

	node := newNode("root", ng)
	node.parentId = ng.RootNode.Id
	node.mode = 6
	node.size = 1024
	node.mimeType = "video/mp4"

	assert.NoError(t, node.Save())

	node.name = "root2"
	node.size = 1025
	node.mode = 7
	node.mimeType = "audio/mp3"

	assert.NoError(t, node.Save())
	assert.Equal(t, "root2", node.Name())
	assert.EqualValues(t, 1025, node.Size())
	assert.EqualValues(t, 7, node.Mode())
	assert.Equal(t, "audio/mp3", node.Type())
}

func TestSave_returnsErrorWhenFileHasNoName(t *testing.T) {
	ng := testInit()

	node := newNode("", ng)
	assert.EqualError(t, node.Save(), "Cannot add nameless file")
}

func TestSave_returnsErrorWhenMTimeIsAfterNow(t *testing.T) {
	ng := testInit()

	node := newNode("child", ng)
	node.mTime = time.Now().Add(time.Second * 100)
	assert.EqualError(t, node.Save(), "Cannot set futuristic mTime")
}

func TestSave_returnsErrorWhenFileHasNegativeSize(t *testing.T) {
	ng := testInit()

	node := newNode("child", ng)
	node.size = -1
	assert.EqualError(t, node.Save(), "File cannot have negative size")
}

func TestSave_returnsErrorWhenDirHasNonZeroSize(t *testing.T) {
	ng := testInit()

	node := newNode("child", ng)
	node.mode = os.ModeDir
	node.size = 1
	assert.EqualError(t, node.Save(), "Dir cannot have non-zero size")
}

func TestSave_returnsErrorWhenAddingNodeWithoutParent(t *testing.T) {
	ng := testInit()

	node := newNode("child", ng)
	assert.EqualError(t, node.Save(), "Cannot add file without parent")
}

func TestWriteData_writesDataToCorrectBlock(t *testing.T) {
	ng := testInit()

	child, _ := makeNode("child", ng.RootNode.Id, 1024, time.Now(), ng)
	dat := RandDat(1024)
	fingerprint := Hash(dat)

	assert.NoError(t, child.WriteData(dat, 0))

	blocks := child.Blocks()
	assert.Len(t, blocks, 1)
	if len(blocks) > 0 {
		assert.Equal(t, fingerprint, blocks[0].Hash)
	}
}

func TestWriteData_throwsOnInvalidBlockOffset(t *testing.T) {
	ng := testInit()

	child, _ := makeNode("child", ng.RootNode.Id, 1024, time.Now(), ng)
	dat := RandDat(1024)

	assert.EqualError(t, child.WriteData(dat, 1), fmt.Sprint("1 is not a valid offset for block size ", BLOCK_SIZE))
}

func TestWriteData_throwsIfDataGreaterThanSize(t *testing.T) {
	ng := testInit()

	child, _ := makeNode("child", ng.RootNode.Id, 1024, time.Now(), ng)
	dat := RandDat(1025)

	assert.EqualError(t, child.WriteData(dat, 0), "Cannot write data that exceeds the size of file")
}

func TestWriteData_removesExistingFingerprintForOffset(t *testing.T) {
	ng := testInit()

	child, _ := makeNode("child", ng.RootNode.Id, 1024, time.Now(), ng)
	dat := RandDat(1024)

	assert.NoError(t, child.WriteData(dat, 0))

	dat = RandDat(1024)
	fingerprint := Hash(dat)
	assert.NoError(t, child.WriteData(dat, 0))

	it := cayley.StartPath(ng, child.Id).Out("offset-0").BuildIterator()
	assert.True(t, cayley.RawNext(it))
	assert.Equal(t, fingerprint, ng.NameOf(it.Result()))
}

func TestBlockWithOffset_findsCorrectBlock(t *testing.T) {
	ng := testInit()

	child, _ := makeNode("child", ng.RootNode.Id, MEGABYTE*2, time.Now(), ng)
	data := RandDat(MEGABYTE)
	assert.NoError(t, child.WriteData(data, 0))

	data2 := RandDat(MEGABYTE)
	assert.NoError(t, child.WriteData(data2, MEGABYTE))

	foundBlock := child.BlockWithOffset(0)
	assert.Equal(t, Hash(data), string(foundBlock))

	foundBlock2 := child.BlockWithOffset(MEGABYTE)
	assert.Equal(t, Hash(data2), string(foundBlock2))
}

func TestBlockWithOffset_returnsEmptyStringForDir(t *testing.T) {
	ng := testInit()

	fingerprint := ng.RootNode.BlockWithOffset(0)
	assert.Equal(t, "", fingerprint)
}

func TestBlocks_returnsEmptySliceForDir(t *testing.T) {
	ng := testInit()

	blocks := ng.RootNode.Blocks()
	assert.EqualValues(t, 0, len(blocks))
}

func TestChmod_chmodsSuccessfully(t *testing.T) {
	ng := testInit()

	child, err := makeNode("child", ng.RootNode.Id, 1, time.Now(), ng)
	assert.NoError(t, err)

	assert.NoError(t, child.Chmod(os.FileMode(0777)))
	assert.EqualValues(t, os.FileMode(0777), child.Mode())
}

func TestTouch_updatesMTime(t *testing.T) {
	ng := testInit()

	then := time.Now().Add(-10 * time.Second)
	child, _ := makeNode("child", ng.RootNode.Id, 1024, then, ng)

	now := time.Now()
	assert.NoError(t, child.Touch(now))

	assert.EqualValues(t, now.Unix(), child.MTime().Unix())
}

func TestTouch_throwsIfDateInFuture(t *testing.T) {
	ng := testInit()

	child, err := makeNode("child", ng.RootNode.Id, 1024, time.Now(), ng)
	assert.NoError(t, err)

	assert.EqualError(t, child.Touch(time.Now().Add(1*time.Second)), "Cannot set futuristic mTime")
}

func TestResize_resizesCorrectly(t *testing.T) {
	ng := testInit()

	child, _ := makeNode("child", ng.RootNode.Id, 1024, time.Now(), ng)
	assert.EqualValues(t, 1024, child.Size())

	assert.NoError(t, child.Resize(1025))
	assert.EqualValues(t, 1025, child.Size())
}

func TestNodeSeeker_readsCorrectData(t *testing.T) {
	ng := testInit()

	child, _ := makeNode("child", ng.RootNode.Id, 1024, time.Now(), ng)
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

func BenchmarkWrite(b *testing.B) {
	ng := testInit()
	var err error

	for i := 0; err == nil && i < b.N; i++ {
		node := newNode(fmt.Sprint(i), ng)
		node.mode = os.ModeDir
		node.parentId = ng.RootNode.Id
		err = node.Save()
	}

	if err != nil {
		panic(err)
	}
}

func BenchmarkName(b *testing.B) {
	ng := testInit()
	node := newNode("child", ng)
	node.parentId = ng.RootNode.Id
	node.Save()

	for i := 0; i < b.N; i++ {
		node.Name()
	}
}

func makeNode(name, parentId string, size int64, mTime time.Time, graph *NodeGraph) (*Node, error) {
	node := newNode(name, graph)
	node.parentId = parentId
	node.size = size
	node.mTime = mTime
	if err := node.Save(); err != nil {
		return nil, err
	} else {
		return node, nil
	}
}

func testInit() *NodeGraph {
	if dir, err := ioutil.TempDir(os.TempDir(), ".olympus"); err != nil {
		panic(err)
	} else {
		os.Setenv("OLYMPUS_HOME", dir)
		if err = env.InitializeEnvironment(); err != nil {
			panic(err)
		}
		handle, _ := cayley.NewMemoryGraph()
		if ng, err := NewGraph(handle); err != nil {
			panic(err)
		} else {
			return ng
		}
	}
}
