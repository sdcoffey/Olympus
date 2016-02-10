package api

import (
	"encoding/gob"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/cayley"
	"github.com/sdcoffey/olympus/graph"
	"github.com/stretchr/testify/assert"
)

func TestEncoderFromHeader_returnsCorrectEncoder(t *testing.T) {
	header := http.Header(make(map[string][]string))
	header.Add("Accept", "application/gob")

	encoder := encoderFromHeader(nil, header)
	assert.IsType(t, &gob.Encoder{}, encoder)
}

func TestEncoderFromHeader_returnsJsonEcoderByDefault(t *testing.T) {
	header := http.Header(make(map[string][]string))

	encoder := encoderFromHeader(nil, header)
	assert.IsType(t, &json.Encoder{}, encoder)
}

func TestDecoderFromHeader_returnsCorrectDecoder(t *testing.T) {
	header := http.Header(make(map[string][]string))
	header.Add("Content-Type", "application/gob")

	decoder := decoderFromHeader(nil, header)
	assert.IsType(t, &gob.Decoder{}, decoder)
}

func TestDecoderFromHeader_returnsJsonDecoderByDefault(t *testing.T) {
	header := http.Header(make(map[string][]string))

	decoder := decoderFromHeader(nil, header)
	assert.IsType(t, &json.Decoder{}, decoder)
}

var (
	server    *httptest.Server
	serverUrl string
	client    *http.Client
	nodeGraph *graph.NodeGraph
)

func init() {
	if handle, err := cayley.NewMemoryGraph(); err != nil {
		panic(err)
	} else if nodeGraph, err = graph.NewGraph(handle); err != nil {
		panic(err)
	}

	server = httptest.NewServer(NewApi(nodeGraph))
	serverUrl = server.URL
	client = http.DefaultClient
}

func endpointFmt(method string) string {
	return serverUrl + "/v1" + method
}

func TestListNodes_returnsErrorIfFileNotExist(t *testing.T) {
	req := request("GET", "/ls/not-an-id", nil)
	resp, err := client.Do(req)
	assert.NoError(t, err)

	assert.EqualValues(t, 400, resp.StatusCode)
	assert.Equal(t, "Node with id: not-an-id does not exist", msg(resp))
}

func TestLsFiles_returnsFilesForValidParent(t *testing.T) {
	cleanFs()

	req := request("GET", "/ls/"+graph.RootNodeId, nil)
	nodeGraph.CreateDirectory(nodeGraph.RootNode, "child")

	resp, err := client.Do(req)
	assert.Nil(t, err)
	assert.EqualValues(t, 200, resp.StatusCode)

	decoder := json.NewDecoder(resp.Body)
	var files []graph.NodeInfo
	decoder.Decode(&files)
	assert.Len(t, files, 1)

	file := files[0]
	assert.Equal(t, "child", file.Name)
}

func TestRmFile_returnsErrorIfFileNotExist(t *testing.T) {
	cleanFs()

	req := request("DELETE", "/rm/child", nil)

	resp, err := client.Do(req)
	assert.NoError(t, err)
	assert.EqualValues(t, 400, resp.StatusCode)

	assert.Equal(t, "Node with id: child does not exist", msg(resp))
}

func TestRmFile_removesFileSuccessfully(t *testing.T) {
	cleanFs()

	node, _ := nodeGraph.CreateDirectory(nodeGraph.RootNode, "child")
	req := request("DELETE", "/rm/"+node.Id, nil)

	resp, err := client.Do(req)
	assert.NoError(t, err)
	assert.EqualValues(t, 200, resp.StatusCode)

	assert.EqualValues(t, 0, len(nodeGraph.RootNode.Children()))
}

func TestMvFile_returnsErrorIfFileNotExist(t *testing.T) {
	cleanFs()

	req := request("PATCH", "/mv/not-an-id/"+nodeGraph.RootNode.Id, nil)

	resp, err := client.Do(req)
	assert.NoError(t, err)
	assert.EqualValues(t, 400, resp.StatusCode)
	assert.Equal(t, "Node with id: not-an-id does not exist", msg(resp))
}

func TestMvFile_returnsErrorIfNewParentNotExist(t *testing.T) {
	cleanFs()

	node, _ := nodeGraph.CreateDirectory(nodeGraph.RootNode, "child")
	req := request("PATCH", "/mv/"+node.Id+"/not-a-parent", nil)

	resp, err := client.Do(req)
	assert.NoError(t, err)
	assert.EqualValues(t, 400, resp.StatusCode)
	assert.Equal(t, "Node with id: not-a-parent does not exist", msg(resp))
}

func TestMvFile_movesFileSuccessfully(t *testing.T) {
	cleanFs()

	node, _ := nodeGraph.CreateDirectory(nodeGraph.RootNode, "child")
	node2, _ := nodeGraph.CreateDirectory(nodeGraph.RootNode, "child2")

	req := request("PATCH", "/mv/"+node.Id+"/"+node2.Id, nil)

	resp, err := client.Do(req)
	assert.Nil(t, err)
	assert.EqualValues(t, 200, resp.StatusCode)

	assert.EqualValues(t, 1, len(nodeGraph.RootNode.Children()))
	assert.EqualValues(t, 1, len(node2.Children()))

	//todo: moved file has different attrs - WHY
}

func TestMvFile_renameInPlaceWorksSuccessfully(t *testing.T) {

}

func TestMvFile_renameAndMoveWorksSuccessfully(t *testing.T) {

}

func request(method, path string, body io.Reader) *http.Request {
	req, _ := http.NewRequest(method, endpointFmt(path), body)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}

func msg(resp *http.Response) string {
	defer resp.Body.Close()
	dat, _ := ioutil.ReadAll(resp.Body)
	return string(dat)
}

func cleanFs() {
	for _, child := range nodeGraph.RootNode.Children() {
		nodeGraph.RemoveNode(child)
	}
}
