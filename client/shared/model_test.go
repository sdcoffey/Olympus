package shared

import (
	"fmt"
	"testing"

	"net/http/httptest"

	"github.com/sdcoffey/olympus/Godeps/_workspace/src/github.com/stretchr/testify/assert"
	"github.com/sdcoffey/olympus/client/apiclient"
	"github.com/sdcoffey/olympus/graph"
	"github.com/sdcoffey/olympus/server/api"
)

func TestModel_returnsErrIfRootDoestNotExist(t *testing.T) {
	client, ng := setup()

	rootNode := ng.NodeWithId("not-an-id")
	model := newModel(client, rootNode, ng)

	assert.EqualError(t, model.init(), "Root with id: not-an-id does not exist")
}

func TestModel_doesNotReturnErrorIfRootExists(t *testing.T) {
	client, ng := setup()

	model := newModel(client, ng.RootNode, ng)
	assert.NoError(t, model.init())
}

func TestModel_initRefreshesCache(t *testing.T) {
	client, ng := setup()

	model := newModel(client, ng.RootNode, ng)

	for i := 0; i < 3; i++ {
		_, err := client.CreateNode(graph.NodeInfo{
			ParentId: graph.RootNodeId,
			Name:     fmt.Sprint(i),
		})
		assert.NoError(t, err)
	}

	err := model.init()
	assert.NoError(t, err)

	assert.EqualValues(t, 3, len(model.Root.Children()))
}

func TestModel_refreshRemovesLocalItemsDeletedRemotely(t *testing.T) {
	client, ng := setup()

	model := newModel(client, ng.RootNode, ng)

	for i := 0; i < 3; i++ {
		_, err := client.CreateNode(graph.NodeInfo{
			ParentId: graph.RootNodeId,
			Name:     fmt.Sprint(i),
		})
		assert.NoError(t, err)
	}

	assert.NoError(t, model.init())

	assert.EqualValues(t, 3, len(model.Root.Children()))

	children := model.graph.NodeWithId(ng.RootNode.Id).Children()
	assert.NoError(t, client.RemoveNode(children[0].Id))

	assert.NoError(t, model.Refresh())
	assert.EqualValues(t, 2, len(model.Root.Children()))
}

func TestModel_findNodeByNameReturnsCorrectNode(t *testing.T) {
	client, ng := setup()

	model := newModel(client, ng.RootNode, ng)

	for i := 0; i < 3; i++ {
		_, err := client.CreateNode(graph.NodeInfo{
			ParentId: graph.RootNodeId,
			Name:     fmt.Sprint(i),
		})
		assert.NoError(t, err)
	}

	assert.NoError(t, model.init())

	node := model.FindNodeByName("1")
	assert.Equal(t, ng.RootNode.Id, node.Parent().Id)
	assert.EqualValues(t, 0, node.Size())
}

var (
	server *httptest.Server
)

func setup() (apiclient.ApiClient, *graph.NodeGraph) {
	graph := graph.TestInit()
	server = httptest.NewServer(api.NewApi(graph))
	client := apiclient.ApiClient{server.URL, api.JsonEncoding}

	return client, graph
}
