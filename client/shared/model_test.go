package shared

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"testing"

	"github.com/google/cayley"
	"github.com/sdcoffey/olympus/env"
	"github.com/sdcoffey/olympus/graph"
	"github.com/stretchr/testify/assert"
)

func TestModel_returnsErrIfRootDoestNotExist(t *testing.T) {
	ng := testInit()

	apiClient := newFakeApiClient()
	rootNode := ng.NodeWithId("not-an-id")
	model := newModel(apiClient, rootNode, ng)

	assert.EqualError(t, model.init(), "Root with id: not-an-id does not exist")
}

func TestModel_doesNotReturnErrorIfRootExists(t *testing.T) {
	ng := testInit()

	apiClient := newFakeApiClient()
	model := newModel(apiClient, ng.RootNode, ng)
	assert.NoError(t, model.init())
}

func TestModel_initRefreshesCache(t *testing.T) {
	ng := testInit()

	apiClient := newFakeApiClient()
	model := newModel(apiClient, ng.RootNode, ng)

	for i := 0; i < 3; i++ {
		_, err := apiClient.ng.CreateDirectory(apiClient.ng.RootNode, fmt.Sprint(i))
		assert.NoError(t, err)
	}

	err := model.init()
	assert.Nil(t, err)

	assert.EqualValues(t, 3, len(model.Root.Children()))
}

func TestModel_refreshRemovesLocalItemsDeletedRemotely(t *testing.T) {
	ng := testInit()

	apiClient := newFakeApiClient()
	model := newModel(apiClient, ng.RootNode, ng)

	for i := 0; i < 3; i++ {
		_, err := apiClient.ng.CreateDirectory(apiClient.ng.RootNode, fmt.Sprint(i))
		assert.NoError(t, err)
	}

	assert.NoError(t, model.init())

	assert.EqualValues(t, 3, len(model.Root.Children()))

	children := apiClient.ng.NodeWithId(ng.RootNode.Id).Children()
	assert.NoError(t, apiClient.ng.RemoveNode(children[0]))

	assert.NoError(t, model.Refresh())
	assert.EqualValues(t, 2, len(model.Root.Children()))
}

func TestModel_findNodeByNameReturnsCorrectnode(t *testing.T) {
	ng := testInit()

	apiClient := newFakeApiClient()
	model := newModel(apiClient, ng.RootNode, ng)

	for i := 0; i < 3; i++ {
		_, err := apiClient.ng.CreateDirectory(apiClient.ng.RootNode, fmt.Sprint(i))
		assert.NoError(t, err)
	}

	assert.NoError(t, model.init())

	node := model.FindNodeByName("1")
	assert.Equal(t, ng.RootNode.Id, node.Parent().Id)
	assert.EqualValues(t, 0, node.Size())
}

type fakeApiClient struct {
	ng *graph.NodeGraph
}

func newFakeApiClient() *fakeApiClient {
	client := new(fakeApiClient)
	client.ng = testInit()

	return client
}

func (client fakeApiClient) ListNodes(parentId string) ([]graph.NodeInfo, error) {
	node := client.ng.NodeWithId(parentId)
	if !node.Exists() {
		return make([]graph.NodeInfo, 0), errors.New("node with id: " + parentId + " does not exist")
	}

	children := node.Children()
	nodeInfos := make([]graph.NodeInfo, len(children))
	for i, child := range children {
		nodeInfos[i] = child.NodeInfo()
	}
	return nodeInfos, nil
}

func (client fakeApiClient) MoveNode(nodeid, newParentId, newName string) error {
	return nil
}

func (client fakeApiClient) RemoveNode(nodeId string) error {
	return nil
}

func (client fakeApiClient) CreateNode(info graph.NodeInfo) (graph.NodeInfo, error) {
	return graph.NodeInfo{}, nil
}

func (client fakeApiClient) UpdateNode(info graph.NodeInfo) error {
	return nil
}

func (client fakeApiClient) HasBlocks(nodeId string, blocks []string) ([]string, error) {
	return []string{}, nil
}

func (client fakeApiClient) SendBlock(nodeId string, block graph.BlockInfo, data io.Reader) error {
	return nil
}

func testInit() *graph.NodeGraph {
	if dir, err := ioutil.TempDir(os.TempDir(), ".olympus"); err != nil {
		panic(err)
	} else {
		os.Setenv("OLYMPUS_HOME", dir)
		if err = env.InitializeEnvironment(); err != nil {
			panic(err)
		}
		handle, _ := cayley.NewMemoryGraph()
		if ng, err := graph.NewGraph(handle); err != nil {
			panic(err)
		} else {
			return ng
		}
	}
}
