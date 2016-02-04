package fs

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

func TestNewFile_hasUuidAndTimeStamp(t *testing.T) {
	testInit()

	file := newFile("root")
	assert.NotEmpty(t, file.Id)
	assert.NotEmpty(t, file.mTime)
}

func TestFileWithFileInfo(t *testing.T) {
	testInit()

	now := time.Now()
	fileInfo := FileInfo{
		Id:       "abc",
		ParentId: "parent",
		Name:     "file",
		Size:     1,
		MTime:    now,
		Attr:     4,
	}

	file := FileWithFileInfo(fileInfo)
	assert.Equal(t, "abc", file.Id)
	assert.Equal(t, "parent", file.parentId)
	assert.Equal(t, "file", file.name)
	assert.EqualValues(t, 1, file.size)
	assert.Equal(t, now, file.mTime)
	assert.EqualValues(t, 4, file.mode)
}

func TestFileInfo(t *testing.T) {
	testInit()

	now := time.Now()
	child, _ := MkFile("child", rootNode.Id, 1024, now)
	info := child.FileInfo()
	assert.Equal(t, child.Id, info.Id)
	assert.Equal(t, "rootNode", info.ParentId)
	assert.Equal(t, "child", info.Name)
	assert.EqualValues(t, 1024, info.Size)
	assert.Equal(t, now.Unix(), info.MTime.Unix())
	assert.EqualValues(t, child.Mode(), info.Attr)
}

func TestFileWithName(t *testing.T) {
	testInit()

	file := newFile("root")
	file.mode = os.ModeDir
	file.Save()

	child := newFile("child")
	child.parentId = file.Id
	child.Save()

	fetchedChild := FileWithName(file.Id, "child")
	assert.NotNil(t, fetchedChild)
	assert.Equal(t, "child", fetchedChild.Name())
}

func TestName(t *testing.T) {
	testInit()

	file := newFile("root")
	file.name = "A cool folder"
	file.Save()

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
	file.Save()

	fromDisk := FileWithId(file.Id)
	assert.EqualValues(t, file.size, fromDisk.Size())
}

func TestMode(t *testing.T) {
	testInit()

	file := newFile("root")
	file.mode = os.ModeDir
	file.Save()

	fromDisk := FileWithId(file.Id)
	assert.EqualValues(t, file.mode, fromDisk.Mode())
}

func TestIsDir(t *testing.T) {
	testInit()

	file := newFile("root")
	file.mode = os.ModeDir
	file.Save()

	fromDisk := FileWithId(file.Id)
	assert.True(t, fromDisk.IsDir())
}

func TestModTime(t *testing.T) {
	testInit()

	file := newFile("root")
	file.Save()

	fromDisk := FileWithId(file.Id)
	assert.NotEmpty(t, fromDisk.ModTime())
	assert.True(t, time.Now().Sub(fromDisk.ModTime()) < time.Second)
}

func TestChildren(t *testing.T) {
	testInit()

	rootNode, _ := RootNode()

	childNode1 := newFile("child1")
	childNode1.parentId = rootNode.Id
	childNode1.Save()

	childNode2 := newFile("child2")
	childNode2.mode |= os.ModeDir
	childNode2.parentId = rootNode.Id
	childNode2.Save()

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
	childNode3.Save()

	children = childNode2.Children()
	assert.EqualValues(t, 1, len(children))
	assert.EqualValues(t, childNode2.Id, children[0].parentId)
	assert.EqualValues(t, childNode3.name, children[0].Name())
}

func TestParent(t *testing.T) {
	testInit()

	rootNode := newFile("root")
	rootNode.mode = os.ModeDir
	rootNode.Save()

	assert.Nil(t, rootNode.Parent())

	childNode := newFile("child")
	childNode.parentId = rootNode.Id
	childNode.Save()

	assert.Equal(t, rootNode.Id, childNode.Parent().Id)
	assert.Equal(t, rootNode.name, childNode.Parent().Name())
	assert.EqualValues(t, rootNode.mode, childNode.Parent().Mode())
}

func TestSave(t *testing.T) {
	testInit()

	mTime := time.Now()

	file := newFile("root")
	file.size = 1024
	file.mode = os.ModeSticky
	file.mTime = mTime

	err := file.Save()
	assert.Nil(t, err)

	g := GlobalFs()
	assertProperty := func(expected string, actual cayley.Iterator) {
		assert.True(t, cayley.RawNext(actual))
		assert.Equal(t, expected, g.NameOf(actual.Result()))
	}

	it := cayley.StartPath(g, file.Id).Out(nameLink).BuildIterator()
	assertProperty("root", it)

	it = cayley.StartPath(g, file.Id).Out(sizeLink).BuildIterator()
	assertProperty("1024", it)

	it = cayley.StartPath(g, file.Id).Out(mTimeLink).BuildIterator()
	assertProperty(fmt.Sprint(mTime.Unix()), it)

	it = cayley.StartPath(g, file.Id).Out(modeLink).BuildIterator()
	assertProperty(fmt.Sprint(int(os.ModeSticky)), it)
}

func TestSave_overwriteExistingProperty(t *testing.T) {
	testInit()

	file := newFile("root")
	file.mode = 6
	file.size = 1024

	err := file.Save()
	assert.Nil(t, err)

	file.name = "root2"
	file.size = 1025
	file.mode = 7

	err = file.Save()
	assert.Nil(t, err)
	assert.Equal(t, "root2", file.Name())
	assert.EqualValues(t, 1025, file.Size())
	assert.EqualValues(t, 7, file.Mode())
}

func TestSave_returnsErrorWhenFileHasNoName(t *testing.T) {
	testInit()

	file := newFile("")
	err := file.Save()
	assert.NotNil(t, err)
}

func TestDelete(t *testing.T) {
	testInit()

	file := newFile("child")
	file.parentId = rootNode.Id
	file.size = 1024
	file.mode = 4
	file.mTime = time.Now()

	file.Save()

	err := file.delete()
	assert.Nil(t, err)

	assert.Zero(t, file.Name())
	assert.Zero(t, file.Size())
	assert.EqualValues(t, 0, file.Mode())
	assert.Zero(t, file.ModTime())

	fetchedFile := FileWithId(file.Id)
	assert.False(t, fetchedFile.Exists())
}

func TestDelete_returnsErrorWhenNodeHasChildren(t *testing.T) {
	testInit()

	file := newFile("root")
	file.mode = os.ModeDir
	file.Save()

	child := newFile("child")
	child.parentId = file.Id
	child.Save()

	err := file.delete()
	assert.NotNil(t, err)
}

func TestWriteData_writesDataToCorrectBlock(t *testing.T) {
	testInit()

	child, _ := MkFile("child", rootNode.Id, 1024, time.Now())
	dat := RandDat(1024)
	fingerprint := Hash(dat)

	err := child.WriteData(dat, 0)
	assert.NoError(t, err)

	blocks := child.Blocks()
	assert.Len(t, blocks, 1)
	if len(blocks) > 0 {
		assert.Equal(t, fingerprint, blocks[0].Hash)
	}
}

func TestWriteData_throwsOnInvalidBlockOffset(t *testing.T) {
	testInit()

	child, _ := MkFile("child", rootNode.Id, 1024, time.Now())
	dat := RandDat(1024)

	err := child.WriteData(dat, 1)
	assert.Error(t, err)
	assert.Equal(t, fmt.Sprint("1 is not a valid offset for block size ", BLOCK_SIZE), err.Error())
}

func TestWriteData_throwsIfDataGreaterThanSize(t *testing.T) {
	testInit()

	child, _ := MkFile("child", rootNode.Id, 1024, time.Now())
	dat := RandDat(1025)

	err := child.WriteData(dat, 0)
	assert.Error(t, err)
	assert.Equal(t, "Cannot write data that exceeds the size of file", err.Error())
}

func TestWriteData_removesExistingFingerprintForOffset(t *testing.T) {
	testInit()

	child, _ := MkFile("child", rootNode.Id, 1024, time.Now())
	dat := RandDat(1024)

	err := child.WriteData(dat, 0)
	assert.NoError(t, err)

	dat = RandDat(1024)
	fingerprint := Hash(dat)
	err = child.WriteData(dat, 0)

	it := cayley.StartPath(GlobalFs(), child.Id).Out("offset-0").BuildIterator()
	assert.True(t, cayley.RawNext(it))
	assert.Equal(t, fingerprint, GlobalFs().NameOf(it.Result()))
}

func TestBlockWithOffset_findsCorrectBlock(t *testing.T) {
	testInit()

	child, _ := MkFile("child", rootNode.Id, MEGABYTE*2, time.Now())
	data := RandDat(MEGABYTE)
	err := child.WriteData(data, 0)
	assert.NoError(t, err)

	data2 := RandDat(MEGABYTE)
	err = child.WriteData(data2, MEGABYTE)
	assert.NoError(t, err)

	foundBlock := child.BlockWithOffset(0)
	assert.Equal(t, Hash(data), string(foundBlock))

	foundBlock2 := child.BlockWithOffset(MEGABYTE)
	assert.Equal(t, Hash(data2), string(foundBlock2))
}

func TestBlockWithOffset_returnsEmptyStringForDir(t *testing.T) {
	testInit()

	child, err := rootNode.MkDir("child")
	assert.NoError(t, err)

	fingerprint := child.BlockWithOffset(0)
	assert.Equal(t, "", fingerprint)
}

func TestBlocks_returnsEmptySliceForDir(t *testing.T) {
	testInit()

	blocks := rootNode.Blocks()
	assert.EqualValues(t, 0, len(blocks))
}

func BenchmarkWrite(b *testing.B) {
	testInit()
	var lastId string
	var err error

	for i := 0; err == nil && i < b.N; i++ {
		file := newFile(fmt.Sprint(i))
		file.mode = os.ModeDir
		file.parentId = lastId
		err = file.Save()
	}
}

func testInit() string {
	if dir, err := ioutil.TempDir(os.TempDir(), ".olympus"); err != nil {
		panic(err)
	} else {
		os.Setenv("OLYMPUS_HOME", dir)
		if err = env.InitializeEnvironment(); err != nil {
			panic(err)
		}
		graph, _ := cayley.NewMemoryGraph()
		Init(graph)

		return dir
	}
}
