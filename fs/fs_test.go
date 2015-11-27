package fs

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestMkDir(t *testing.T) {
	testInit()

	rootNode := newFile("root")
	rootNode.mode = os.ModeDir
	rootNode.Write()

	child, err := MkDir(rootNode.Id, "child")
	assert.Nil(t, err)
	assert.NotNil(t, child)
	assert.Equal(t, rootNode.Id, child.Parent().Id)
	assert.EqualValues(t, 1, len(rootNode.Children()))
}

func TestMkDir_returnsErrorWhenParentNotDir(t *testing.T) {
	testInit()

	rootNode := newFile("root")
	rootNode.Write()

	child, err := MkDir(rootNode.Id, "child")
	assert.NotNil(t, err)
	assert.Nil(t, child)
}

func TestRm_deletesAllChildNodes(t *testing.T) {
	testInit()

	root := newFile("root")
	root.mode = os.ModeDir
	root.Write()

	child, _ := MkDir(root.Id, "child")
	child2, _ := MkDir(root.Id, "child2")
	MkDir(child2.Id, "child3")

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

	root := newFile("root")
	root.mode = os.ModeDir
	root.Write()

	child1, _ := MkDir(root.Id, "child1")
	child2, _ := MkDir(root.Id, "child2")

	assert.EqualValues(t, 2, len(root.Children()))

	err := Mv(child2, child2.Name(), child1.Id)
	assert.Nil(t, err)

	assert.EqualValues(t, 1, len(root.Children()))
	assert.EqualValues(t, 1, len(child1.Children()))

	child := child1.Children()[0]
	assert.Equal(t, child2.Id, child.Id)
	assert.Equal(t, child1.Id, child.parentId)
}

func TestMv_renamesNodeSuccessfully(t *testing.T) {
	testInit()

	root := newFile("root")
	root.mode = os.ModeDir
	root.Write()

	child, _ := MkDir(root.Id, "child")
	assert.Equal(t, "child", child.Name())

	err := Mv(child, "THE child", root.Id)
	assert.Nil(t, err)
	assert.Equal(t, "THE child", child.Name())
	assert.Equal(t, root.Id, child.Parent().Id)
}

func TestMv_throwsWhenMovingRootNode(t *testing.T) {
	testInit()

	root := newFile("root")
	root.mode = os.ModeDir
	root.Write()

	err := Mv(root, "root", "abcd-new-parent")
	assert.NotNil(t, err)
}

func TestMv_throwsWhenMovingNodeInsideItself(t *testing.T) {
	testInit()

	root := newFile("root")
	root.mode = os.ModeDir
	root.Write()

	file, err := MkDir(root.Id, "child")
	err = Mv(file, file.Name(), file.Id)

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "move file inside itself")
}
