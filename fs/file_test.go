package fs

import (
	"fmt"
	"github.com/google/cayley"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
	"time"
)

func testInit() {
	graph, _ := cayley.NewMemoryGraph()
	Init(graph)
}

func TestNewFile_hasUuidAndTimeStamp(t *testing.T) {
	testInit()

	file := newFile("root")
	assert.NotEmpty(t, file.Id)
	assert.NotEmpty(t, file.mTime)
}

func TestFileWithName(t *testing.T) {
	testInit()

	file := newFile("root")
	file.mode = os.ModeDir
	file.Write()

	child := newFile("child")
	child.parentId = file.Id
	child.Write()

	fetchedChild := FileWithName(file.Id, "child")
	assert.NotNil(t, fetchedChild)
	assert.Equal(t, "child", fetchedChild.Name())
}

func TestName(t *testing.T) {
	testInit()

	file := newFile("root")
	file.name = "A cool folder"
	file.Write()

	fromDisk := FileWithId(file.Id)

	assert.Equal(t, file.name, fromDisk.Name())
}

func TestExists(t *testing.T) {
	testInit()

	file := newFile("")
	assert.False(t, file.Exists())
}

func TestSize(t *testing.T) {
	testInit()

	file := newFile("root")
	file.size = 1024
	file.Write()

	fromDisk := FileWithId(file.Id)
	assert.EqualValues(t, file.size, fromDisk.Size())
}

func TestMode(t *testing.T) {
	testInit()

	file := newFile("root")
	file.mode = os.ModeDir
	file.Write()

	fromDisk := FileWithId(file.Id)
	assert.EqualValues(t, file.mode, fromDisk.Mode())
}

func TestIsDir(t *testing.T) {
	testInit()

	file := newFile("root")
	file.mode = os.ModeDir
	file.Write()

	fromDisk := FileWithId(file.Id)
	assert.True(t, fromDisk.IsDir())
}

func TestModTime(t *testing.T) {
	testInit()

	file := newFile("root")
	file.Write()

	fromDisk := FileWithId(file.Id)
	assert.NotEmpty(t, fromDisk.ModTime())
	assert.True(t, time.Now().Sub(fromDisk.ModTime()) < time.Second)
}

func TestChildren(t *testing.T) {
	testInit()

	rootNode := newFile("root")
	rootNode.mode = os.ModeDir
	rootNode.Write()

	childNode1 := newFile("child1")
	childNode1.parentId = rootNode.Id
	childNode1.Write()

	childNode2 := newFile("child2")
	childNode2.mode = os.ModeDir
	childNode2.parentId = rootNode.Id
	childNode2.Write()

	children := rootNode.Children()
	assert.EqualValues(t, 2, len(children))

	for idx, child := range children {
		assert.Equal(t, rootNode.Id, child.parentId)
		if idx == 0 {
			assert.Equal(t, childNode1.name, child.Name())
		} else {
			assert.Equal(t, childNode2.name, child.Name())
		}
	}

	childNode3 := newFile("child3")
	childNode3.parentId = childNode2.Id
	childNode3.Write()

	children = childNode2.Children()
	assert.EqualValues(t, 1, len(children))
	assert.EqualValues(t, childNode2.Id, children[0].parentId)
	assert.EqualValues(t, childNode3.name, children[0].Name())
}

func TestParent(t *testing.T) {
	testInit()

	rootNode := newFile("root")
	rootNode.mode = os.ModeDir
	rootNode.Write()

	assert.Nil(t, rootNode.Parent())

	childNode := newFile("child")
	childNode.parentId = rootNode.Id
	childNode.Write()

	assert.Equal(t, rootNode.Id, childNode.Parent().Id)
	assert.Equal(t, rootNode.name, childNode.Parent().Name())
	assert.EqualValues(t, rootNode.mode, childNode.Parent().Mode())
}

func TestWrite(t *testing.T) {
	testInit()

	file := newFile("root")
	file.mode = os.ModeDir
	file.size = 1024

	err := file.Write()
	assert.Nil(t, err)

	it := cayley.StartPath(GlobalFs().Graph, file.Id).Out(parentLink).
		Or(cayley.StartPath(GlobalFs().Graph, file.Id).Out(nameLink)).
		Or(cayley.StartPath(GlobalFs().Graph, file.Id).Out(sizeLink)).
		Or(cayley.StartPath(GlobalFs().Graph, file.Id).Out(modeLink)).
		Or(cayley.StartPath(GlobalFs().Graph, file.Id).Out(mTimeLink)).BuildIterator()

	for cayley.RawNext(it) {
		assert.NotEmpty(t, GlobalFs().Graph.NameOf(it.Result()))
	}
}

func TestWrite_returnsErrorWhenFileHasNoName(t *testing.T) {
	testInit()

	file := newFile("")
	err := file.Write()
	assert.NotNil(t, err)
}

func TestWrite_returnsAnErrorWhenDuplicateSiblingExists(t *testing.T) {
	testInit()

	file := newFile("root")
	file.mode = os.ModeDir
	file.Write()

	child1 := newFile("child")
	child1.parentId = file.Id
	child2 := newFile("child")
	child2.parentId = file.Id

	err := child1.Write()
	assert.Nil(t, err)
	err = child2.Write()
	assert.NotNil(t, err)
}

func TestDelete(t *testing.T) {
	testInit()

	file := newFile("root")
	file.mode = os.ModeDir
	file.size = 1024
	file.Write()

	err := file.delete()
	assert.Nil(t, err)

	assert.Zero(t, file.Name())
	assert.Zero(t, file.Size())
	assert.EqualValues(t, 1, file.Mode())
	assert.Zero(t, file.ModTime())

	fetchedFile := FileWithId(file.Id)
	assert.False(t, fetchedFile.Exists())
}

func TestDelete_returnsErrorWhenNodeHasChildren(t *testing.T) {
	testInit()

	file := newFile("root")
	file.mode = os.ModeDir
	file.Write()

	child := newFile("child")
	child.parentId = file.Id
	child.Write()

	err := file.delete()
	assert.NotNil(t, err)
}

func BenchmarkWrite(b *testing.B) {
	testInit()
	var lastId string
	var err error

	for i := 0; err == nil && i < b.N; i++ {
		file := newFile(fmt.Sprint(i))
		file.mode = os.ModeDir
		file.parentId = lastId
		err = file.Write()
	}
}
