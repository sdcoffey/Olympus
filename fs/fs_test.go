package fs

import (
	"github.com/google/cayley"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
	"time"
)

func TestMkDir(t *testing.T) {
	testInit()

	root, _ := RootNode()

	child, err := root.MkDir("child")
	assert.Nil(t, err)
	assert.NotEmpty(t, child.Id)
	assert.NotNil(t, child)
	assert.Equal(t, root.Id, child.Parent().Id)
	assert.EqualValues(t, 1, len(root.Children()))
}

func TestMkDir_returnsErrorWhenParentNotDir(t *testing.T) {
	testInit()

	rootNode := newFile("root")
	rootNode.Save()

	child, err := rootNode.MkDir("child")
	assert.NotNil(t, err)
	assert.Nil(t, child)
}

func TestRootNode_CreatesRootNode(t *testing.T) {
	testInit()

	root, err := RootNode()

	assert.Nil(t, err)
	assert.NotNil(t, root)
	assert.Equal(t, "rootNode", root.Id)
	assert.EqualValues(t, 700|os.ModeDir, root.Mode())

	it := cayley.StartPath(GlobalFs().Graph, "rootNode").Out(nameLink).BuildIterator()
	assert.True(t, cayley.RawNext(it))
	assert.Equal(t, "root", GlobalFs().Graph.NameOf(it.Result()))
}

func TestRm_throwsWhenDeletingRootNode(t *testing.T) {
	testInit()

	root, _ := RootNode()

	err := Rm(root)
	assert.NotNil(t, err)
}

func TestRm_deletesAllChildNodes(t *testing.T) {
	testInit()

	root, _ := RootNode()

	child, _ := root.MkDir("child")
	child2, _ := root.MkDir("child2")
	child2.MkDir("child3")

	err := Rm(child2)
	assert.Nil(t, err)
	assert.EqualValues(t, 0, len(child2.Children()))

	assert.EqualValues(t, 1, len(root.Children()))

	fetchedChild3 := FileWithName(child2.Id, "child3")
	assert.Nil(t, fetchedChild3)

	err = Rm(child)
	assert.Nil(t, err)

	assert.EqualValues(t, 0, len(root.Children()))

	fetchedChild := FileWithName(root.Id, "child")
	assert.Nil(t, fetchedChild)
}

func TestMv_movesNodeSuccessfully(t *testing.T) {
	testInit()

	root, _ := RootNode()

	child1, _ := root.MkDir("child1")
	child2, _ := root.MkDir("child2")

	assert.EqualValues(t, 2, len(root.Children()))

	err := child2.Mv(child2.Name(), child1.Id)
	assert.Nil(t, err)

	assert.EqualValues(t, 1, len(root.Children()))
	assert.EqualValues(t, 1, len(child1.Children()))

	child := child1.Children()[0]
	assert.Equal(t, child2.Id, child.Id)
	assert.Equal(t, child1.Id, child.parentId)
}

func TestMv_renamesNodeSuccessfully(t *testing.T) {
	testInit()

	root, _ := RootNode()

	child, _ := root.MkDir("child")
	assert.Equal(t, "child", child.Name())

	err := child.Mv("THE child", root.Id)
	assert.Nil(t, err)
	assert.Equal(t, "THE child", child.Name())
	assert.Equal(t, root.Id, child.Parent().Id)
}

func TestMv_throwsWhenMovingRootNode(t *testing.T) {
	testInit()

	root, _ := RootNode()

	err := root.Mv("root", "abcd-new-parent")
	assert.NotNil(t, err)
}

func TestMv_throwsWhenMovingNodeInsideItself(t *testing.T) {
	testInit()

	root, _ := RootNode()

	file, err := root.MkDir("child")
	err = file.Mv(file.Name(), file.Id)

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "move file inside itself")
}

func TestChmod_chmodsSuccessfully(t *testing.T) {
	testInit()

	root, _ := RootNode()

	child := newFile("child")
	root.addChild(child)

	err := child.Chmod(os.FileMode(777))
	assert.Nil(t, err)
	assert.EqualValues(t, 777, child.Mode())
}

func TestChmod_throwsWhenChangingDirToFile(t *testing.T) {
	testInit()

	root, _ := RootNode()

	child, _ := root.MkDir("child")
	err := child.Chmod(1)
	assert.NotNil(t, err)
}

func TestChmod_throwWhenChangingFileToDir(t *testing.T) {
	testInit()

	root, _ := RootNode()

	child := newFile("file")
	root.addChild(child)

	err := child.Chmod(os.ModeDir)
	assert.NotNil(t, err)
}

func TestAddChild_returnsAnErrorWhenDuplicateSiblingExists(t *testing.T) {
	testInit()

	file, _ := RootNode()
	child1 := newFile("child")
	err := file.addChild(child1)
	assert.Nil(t, err)

	child2 := newFile("child")
	err = file.addChild(child2)
	assert.NotNil(t, err)
}

func TestAddChild_throwsWhenParentDoesNotExist(t *testing.T) {
	testInit()

	orphan := newFile("file")
	notFile := FileWithId("not-a-file")
	err := notFile.addChild(orphan)
	assert.NotNil(t, err)
}

func TestMkFile_mksFile(t *testing.T) {
	testInit()

	root, _ := RootNode()
	now := time.Now()
	child, err := MkFile("child", root.Id, 1024, now)
	assert.Nil(t, err)
	assert.NotNil(t, child)

	assert.Equal(t, "child", child.Name())
	assert.EqualValues(t, 1024, child.Size())
	assert.EqualValues(t, now.Unix(), child.ModTime().Unix())
	assert.False(t, child.IsDir())
}

func TestTouch_updatesMTime(t *testing.T) {
	testInit()

	then := time.Now().Add(-10 * time.Second)
	child, _ := MkFile("child", rootNode.Id, 1024, then)

	now := time.Now()
	err := child.Touch(now)
	assert.Nil(t, err)

	assert.EqualValues(t, now.Unix(), child.ModTime().Unix())
}

func TestTouch_throwsIfDateInFuture(t *testing.T) {
	testInit()

	child, _ := MkFile("child", rootNode.Id, 1024, time.Now())

	err := child.Touch(time.Now().Add(1 * time.Microsecond))
	assert.NotNil(t, err)
}
