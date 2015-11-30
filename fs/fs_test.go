package fs

import (
	"github.com/google/cayley"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestMkDir(t *testing.T) {
	testInit()

	root, _ := RootNode()

	child, err := MkDir(root.Id, "child")
	assert.Nil(t, err)
	assert.NotEmpty(t, child.Id)
	assert.NotNil(t, child)
	assert.Equal(t, root.Id, child.Parent().Id)
	assert.EqualValues(t, 1, len(root.Children()))
}

func TestMkDir_returnsErrorWhenParentNotDir(t *testing.T) {
	testInit()

	rootNode := newFile("root")
	rootNode.Write()

	child, err := MkDir(rootNode.Id, "child")
	assert.NotNil(t, err)
	assert.Nil(t, child)
}

func TestRootNode_CreatesRootNode(t *testing.T) {
	testInit()

	root, err := RootNode()

	assert.Nil(t, err)
	assert.NotNil(t, root)
	assert.Equal(t, "rootNode", root.Id)
	assert.EqualValues(t, os.ModeDir, root.Mode())

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

	root, _ := RootNode()

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

	root, _ := RootNode()

	child, _ := MkDir(root.Id, "child")
	assert.Equal(t, "child", child.Name())

	err := Mv(child, "THE child", root.Id)
	assert.Nil(t, err)
	assert.Equal(t, "THE child", child.Name())
	assert.Equal(t, root.Id, child.Parent().Id)
}

func TestMv_throwsWhenMovingRootNode(t *testing.T) {
	testInit()

	root, _ := RootNode()

	err := Mv(root, "root", "abcd-new-parent")
	assert.NotNil(t, err)
}

func TestMv_throwsWhenMovingNodeInsideItself(t *testing.T) {
	testInit()

	root, _ := RootNode()

	file, err := MkDir(root.Id, "child")
	err = Mv(file, file.Name(), file.Id)

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "move file inside itself")
}
