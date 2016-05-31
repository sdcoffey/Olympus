package apiclient

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"encoding/xml"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/sdcoffey/olympus/Godeps/_workspace/src/github.com/pborman/uuid"
	"github.com/sdcoffey/olympus/Godeps/_workspace/src/github.com/stretchr/testify/assert"
	"github.com/sdcoffey/olympus/graph"
	"github.com/sdcoffey/olympus/server/api"
)

var (
	server *httptest.Server
)

func setup() (string, *graph.NodeGraph) {
	graph := graph.TestInit()
	server = httptest.NewServer(api.NewApi(graph))

	return server.URL, graph
}

func TestApiClient_TestEncoder_returnsCorrectEncoder(t *testing.T) {
	client := ApiClient{"", api.JsonEncoding}
	assert.IsType(t, &json.Encoder{}, client.encoder(nil))

	client = ApiClient{"", api.GobEncoding}
	assert.IsType(t, &gob.Encoder{}, client.encoder(nil))

	client = ApiClient{"", api.XmlEncoding}
	assert.IsType(t, &xml.Encoder{}, client.encoder(nil))
}

func TestApiClient_TestDecoder_returnsCorrectEncoder(t *testing.T) {
	client := ApiClient{"", api.JsonEncoding}
	assert.IsType(t, &json.Decoder{}, client.decoder(nil))

	client = ApiClient{"", api.GobEncoding}
	assert.IsType(t, &gob.Decoder{}, client.decoder(nil))

	client = ApiClient{"", api.XmlEncoding}
	assert.IsType(t, &xml.Decoder{}, client.decoder(nil))
}

func TestApiClient_ListNodes_returnsEmptyListWhenNoNodes(t *testing.T) {
	address, _ := setup()
	client := ApiClient{address, api.JsonEncoding}

	nodes, err := client.ListNodes(graph.RootNodeId)
	assert.NoError(t, err)
	assert.Len(t, nodes, 0)
}

func TestApiClient_ListNodes_returnsNodes(t *testing.T) {
	address, ng := setup()

	client := ApiClient{address, api.JsonEncoding}

	_, err := ng.CreateDirectory(ng.RootNode, "child1")
	assert.NoError(t, err)

	_, err = ng.CreateDirectory(ng.RootNode, "child2")
	assert.NoError(t, err)

	nodes, err := client.ListNodes(graph.RootNodeId)
	assert.NoError(t, err)

	assert.Len(t, nodes, 2)

	assert.Equal(t, "child1", nodes[0].Name)
	assert.True(t, nodes[0].Mode&uint32(os.ModeDir) > 0)
	assert.Equal(t, graph.RootNodeId, nodes[0].ParentId)

	assert.Equal(t, "child2", nodes[1].Name)
	assert.True(t, nodes[1].Mode&uint32(os.ModeDir) > 0)
	assert.Equal(t, graph.RootNodeId, nodes[1].ParentId)
}

func TestApiClient_ListBlocks_listsBlocks(t *testing.T) {
	address, ng := setup()

	client := ApiClient{address, api.JsonEncoding}
	node := ng.NodeWithNodeInfo(graph.NodeInfo{
		Id:       uuid.New(),
		ParentId: graph.RootNodeId,
		Name:     "thing.txt",
		Mode:     0755,
	})
	assert.NoError(t, node.save())

	dat1 := graph.RandDat(graph.MEGABYTE)
	hash1 := graph.Hash(dat1)
	dat2 := graph.RandDat(graph.MEGABYTE)
	hash2 := graph.Hash(dat2)

	assert.NoError(t, node.WriteData(dat1, 0))
	assert.NoError(t, node.WriteData(dat2, graph.MEGABYTE))

	blocks, err := client.ListBlocks(node.Id)
	assert.NoError(t, err)
	assert.Len(t, blocks, 2)
	assert.Equal(t, hash1, blocks[0].Hash)
	assert.EqualValues(t, 0, blocks[0].Offset)
	assert.Equal(t, hash2, blocks[1].Hash)
	assert.EqualValues(t, graph.MEGABYTE, blocks[1].Offset)
}

func TestApiClient_WriteBlock_writesBlock(t *testing.T) {
	address, ng := setup()

	client := ApiClient{address, api.JsonEncoding}
	node := ng.NodeWithNodeInfo(graph.NodeInfo{
		Id:       uuid.New(),
		ParentId: graph.RootNodeId,
		Name:     "thing.txt",
		Mode:     0755,
	})
	assert.NoError(t, node.save())

	dat := graph.RandDat(graph.MEGABYTE)
	buf := bytes.NewBuffer(dat)

	err := client.WriteBlock(node.Id, 0, graph.Hash(dat), buf)
	assert.NoError(t, err)

	blocks := node.Blocks()
	assert.Len(t, blocks, 1)
	assert.Equal(t, graph.Hash(dat), blocks[0].Hash)
	assert.EqualValues(t, 0, blocks[0].Offset)
}

func TestApiClient_RemoveNode_removesNode(t *testing.T) {
	address, ng := setup()

	client := ApiClient{address, api.JsonEncoding}
	node := ng.NodeWithNodeInfo(graph.NodeInfo{
		Id:       uuid.New(),
		ParentId: graph.RootNodeId,
		Name:     "thing.txt",
		Mode:     0755,
	})
	assert.NoError(t, node.save())

	err := client.RemoveNode(node.Id)
	assert.NoError(t, err)

	assert.Len(t, ng.RootNode.Children(), 0)
}

func TestApiClient_CreateNode_createsNode(t *testing.T) {
	address, ng := setup()

	client := ApiClient{address, api.JsonEncoding}
	nodeInfo := graph.NodeInfo{
		ParentId: graph.RootNodeId,
		Name:     "file.txt",
		Mode:     755,
	}
	returnedNodeInfo, err := client.CreateNode(nodeInfo)
	assert.NoError(t, err)

	createdNode := ng.NodeWithId(returnedNodeInfo.Id)
	assert.Equal(t, nodeInfo.Name, createdNode.Name())
	assert.EqualValues(t, int(755), int(createdNode.Mode()))
	assert.True(t, time.Now().Sub(createdNode.MTime()) < time.Second)
}

func TestApiClient_UpdateNode_updatesNode(t *testing.T) {
	address, ng := setup()

	client := ApiClient{address, api.JsonEncoding}
	node := ng.NodeWithNodeInfo(graph.NodeInfo{
		Id:       uuid.New(),
		ParentId: graph.RootNodeId,
		Name:     "thing.txt",
		Mode:     0755,
	})
	assert.NoError(t, node.save())

	info := node.NodeInfo()
	info.Mode = 700
	info.Name = "file.dat"

	err := client.UpdateNode(info)
	assert.NoError(t, err)

	assert.Equal(t, "file.dat", node.Name())
	assert.EqualValues(t, 700, int(node.Mode()))
}

func TestApiCLient_request_buildsCorrectUrl(t *testing.T) {
	address, _ := setup()

	client := ApiClient{address, api.JsonEncoding}

	request, err := client.request(api.CreateNode, "abcd")
	assert.NoError(t, err)

	assert.Equal(t, address+"/v1/node/abcd", request.URL.String())
}

func TestApiClient_request_setsCorrectVerb(t *testing.T) {
	address, _ := setup()

	client := ApiClient{address, api.JsonEncoding}

	request, err := client.request(api.ListNodes, "abcd")
	assert.NoError(t, err)
	assert.Equal(t, "GET", request.Method)

	request, err = client.request(api.WriteBlock, "abcd", 0)
	assert.NoError(t, err)
	assert.Equal(t, "PUT", request.Method)

	request, err = client.request(api.RemoveNode, "abcd")
	assert.NoError(t, err)
	assert.Equal(t, "DELETE", request.Method)

	request, err = client.request(api.CreateNode, "abcd")
	assert.NoError(t, err)
	assert.Equal(t, "POST", request.Method)

	request, err = client.request(api.UpdateNode, "abcd")
	assert.NoError(t, err)
	assert.Equal(t, "PATCH", request.Method)
}

func TestApiClient_request_setsCorrectHeaders(t *testing.T) {
	address, _ := setup()

	client := ApiClient{address, api.JsonEncoding}

	request, err := client.request(api.ListNodes, "abcd")
	assert.NoError(t, err)
	assert.Equal(t, string(api.JsonEncoding), request.Header.Get("Accept"))

	request, err = client.request(api.CreateNode, "abcd")
	assert.NoError(t, err)
	assert.Equal(t, string(api.JsonEncoding), request.Header.Get("Accept"))
	assert.Equal(t, string(api.JsonEncoding), request.Header.Get("Content-Type"))

	request, err = client.request(api.UpdateNode, "abcd")
	assert.NoError(t, err)
	assert.Equal(t, string(api.JsonEncoding), request.Header.Get("Accept"))
	assert.Equal(t, string(api.JsonEncoding), request.Header.Get("Content-Type"))

	request, err = client.request(api.WriteBlock, "abcd", 0)
	assert.NoError(t, err)
	assert.Equal(t, string(api.JsonEncoding), request.Header.Get("Accept"))
	assert.Equal(t, string(api.JsonEncoding), request.Header.Get("Content-Type"))
}

func TestApiClient_do_readsErrorFROmBodyOnNonOkStatus(t *testing.T) {
	address, _ := setup()

	client := ApiClient{address, api.JsonEncoding}
	_, err := client.ListNodes("not-found")
	assert.Contains(t, err.Error(), "Node with id: not-found does not exist")
}
