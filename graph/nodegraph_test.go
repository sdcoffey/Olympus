package graph

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNodeWithNodeInfo(t *testing.T) {
	ng := testInit()

	now := time.Now()
	info := NodeInfo{
		Id:       "abc",
		ParentId: "parent",
		Name:     "node",
		Size:     1,
		MTime:    now,
		Mode:     4,
		Type:     "application/json",
	}

	node := ng.NodeWithNodeInfo(info)
	assert.Equal(t, "abc", node.Id)
	assert.Equal(t, "parent", node.parentId)
	assert.Equal(t, "node", node.name)
	assert.EqualValues(t, 1, node.size)
	assert.Equal(t, now, node.mTime)
	assert.EqualValues(t, 4, node.mode)
	assert.Equal(t, "application/json", node.mimeType)
}

func TestNodeWithName(t *testing.T) {
	ng := testInit()

	node := newNode("root", ng)
	node.parentId = ng.RootNode.Id
	node.mode = os.ModeDir
	assert.NoError(t, node.Save())

	child := newNode("child", ng)
	child.parentId = node.Id
	assert.NoError(t, child.Save())

	fetchedChild := ng.NodeWithName(node.Id, "child")
	assert.NotNil(t, fetchedChild)
	assert.Equal(t, "child", fetchedChild.Name())
}

func TestCreateDirectory(t *testing.T) {
	ng := testInit()

	child, err := ng.CreateDirectory(ng.RootNode, "child")
	assert.NoError(t, err)
	assert.NotNil(t, child)
	assert.NotEmpty(t, child.Id)
	assert.Equal(t, ng.RootNode.Id, child.Parent().Id)
	assert.EqualValues(t, 1, len(ng.RootNode.Children()))
}

func TestCreateDirectory_returnsErrorWhenParentNotDir(t *testing.T) {
	ng := testInit()

	childNode := newNode("child", ng)
	childNode.parentId = ng.RootNode.Id
	assert.NoError(t, childNode.Save())

	_, err := ng.CreateDirectory(childNode, "secondChild")
	assert.EqualError(t, err, "Cannot add node to a non-directory")
}

func TestRemoveNode_throwsWhenDeletingRootNode(t *testing.T) {
	ng := testInit()

	err := ng.RemoveNode(ng.RootNode)
	assert.EqualError(t, err, "Cannot delete root node")
}

func TestRemoveNode_deletesAllChildNodes(t *testing.T) {
	ng := testInit()

	child, _ := ng.CreateDirectory(ng.RootNode, "child")
	child2, _ := ng.CreateDirectory(ng.RootNode, "child2")
	ng.CreateDirectory(child2, "child3")

	assert.NoError(t, ng.RemoveNode(child2))
	assert.EqualValues(t, 0, len(child2.Children()))
	assert.EqualValues(t, 1, len(ng.RootNode.Children()))

	assert.NoError(t, ng.RemoveNode(child))

	assert.EqualValues(t, 0, len(ng.RootNode.Children()))
}

func TestMoveNode_movesNodeSuccessfully(t *testing.T) {
	ng := testInit()

	child1, err := ng.CreateDirectory(ng.RootNode, "child1")
	assert.NoError(t, err)
	child2, err := ng.CreateDirectory(ng.RootNode, "child2")
	assert.NoError(t, err)

	assert.EqualValues(t, 2, len(ng.RootNode.Children()))

	assert.NoError(t, ng.MoveNode(child2, child2.Name(), child1.Id))

	assert.EqualValues(t, 1, len(ng.RootNode.Children()))
	assert.EqualValues(t, 1, len(child1.Children()))

	child := child1.Children()[0]
	assert.Equal(t, child2.Id, child.Id)
	assert.Equal(t, child1.Id, child.parentId)
	assert.Equal(t, "child2", child.Name())
}

func TestMoveNode_renamesNodeSuccessfully(t *testing.T) {
	ng := testInit()

	child, _ := ng.CreateDirectory(ng.RootNode, "child")
	assert.Equal(t, "child", child.Name())

	assert.NoError(t, ng.MoveNode(child, "THE child", ng.RootNode.Id))
	assert.Equal(t, "THE child", child.Name())
	assert.Equal(t, ng.RootNode.Id, child.Parent().Id)
}

func TestMv_throwsWhenMovingRootNode(t *testing.T) {
	ng := testInit()

	assert.EqualError(t, ng.MoveNode(ng.RootNode, "root", "abcd-new-parent"), "Cannot move root node")
}

func TestMv_throwsWhenMovingNodeInsideItself(t *testing.T) {
	ng := testInit()

	node, err := ng.CreateDirectory(ng.RootNode, "child")
	assert.NoError(t, err)
	assert.EqualError(t, ng.MoveNode(node, node.Name(), node.Id), "Cannot move node inside itself")
}

func TestAddNode_returnsAnErrorWhenDuplicateSiblingExists(t *testing.T) {
	ng := testInit()

	_, err := ng.CreateDirectory(ng.RootNode, "child")
	assert.NoError(t, err)

	child2 := newNode("child", ng)
	assert.EqualError(t, ng.addNode(ng.RootNode, child2), fmt.Sprintf("Node with name %s already exists in %s", child2.name, ng.RootNode.Name()))
}

func TestAddChild_throwsWhenParentDoesNotExist(t *testing.T) {
	ng := testInit()

	orphan := newNode("file", ng)
	notnode := ng.NodeWithId("not-a-file")
	assert.EqualError(t, ng.addNode(notnode, orphan), "Cannot add node to a non-directory")
}

func TestRemoveNode_returnsErrorWhenNodeHasChildren(t *testing.T) {
	ng := testInit()

	child1, err := ng.CreateDirectory(ng.RootNode, "child")
	assert.NoError(t, err)

	child := newNode("child", ng)
	child.parentId = child1.Id
	child.Save()

	assert.EqualError(t, ng.removeNode(child1), "Can't delete node with children, must delete children first")
}
