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
