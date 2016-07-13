package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/sdcoffey/olympus/Godeps/_workspace/src/github.com/stretchr/testify/assert"
	. "github.com/sdcoffey/olympus/Godeps/_workspace/src/gopkg.in/check.v1"
	"github.com/sdcoffey/olympus/graph"
	"github.com/sdcoffey/olympus/graph/testutils"
	"github.com/sdcoffey/olympus/server/api"
)

func (suite *ApiTestSuite) TestListNodes_returns404IfFileNotExist(t *C) {
	req := suite.request(api.ListNodes.Build("not-a-node"), nil)
	resp, err := suite.client.Do(req)
	assert.NoError(t, err)

	assert.EqualValues(t, http.StatusNotFound, resp.StatusCode)
	assert.Contains(t, msg(resp), "Node with id: not-a-node does not exist")
}

func (suite *ApiTestSuite) TestListNodes_returnsFilesForValidParent(t *C) {
	endpoint := api.ListNodes.Build(graph.RootNodeId)
	req := suite.request(endpoint, nil)
	suite.ng.CreateDirectory(suite.ng.RootNode.Id, "child")

	resp, err := suite.client.Do(req)
	assert.Nil(t, err)
	assert.EqualValues(t, http.StatusOK, resp.StatusCode)

	var files []graph.NodeInfo
	decode(resp, &files)

	assert.Len(t, files, 1)

	file := files[0]
	assert.Equal(t, "child", file.Name)
}

func (suite *ApiTestSuite) TestListNodes_returnsNFilesWhenLimitProvided(t *C) {
	req := suite.request(api.ListNodes.Build(graph.RootNodeId).Query("limit", "1"), nil)
	suite.ng.CreateDirectory(suite.ng.RootNode.Id, "child1")
	suite.ng.CreateDirectory(suite.ng.RootNode.Id, "child2")

	resp, err := suite.client.Do(req)
	assert.NoError(t, err)
	assert.EqualValues(t, http.StatusOK, resp.StatusCode)

	var files []graph.NodeInfo
	decode(resp, &files)

	assert.Len(t, files, 1)

	file := files[0]
	assert.Equal(t, "child1", file.Name)
}

func (suite *ApiTestSuite) TestListNodes_startsWithNFileWhenWatermarkProvided(t *C) {
	endpoint := api.ListNodes.Build(graph.RootNodeId).Query("watermark", "1")
	req := suite.request(endpoint, nil)
	suite.ng.CreateDirectory(suite.ng.RootNode.Id, "child1")
	suite.ng.CreateDirectory(suite.ng.RootNode.Id, "child2")

	resp, err := suite.client.Do(req)
	assert.NoError(t, err)
	assert.EqualValues(t, http.StatusOK, resp.StatusCode)

	var files []graph.NodeInfo
	decode(resp, &files)

	assert.Len(t, files, 1)

	file := files[0]
	assert.Equal(t, "child2", file.Name)
}

func (suite *ApiTestSuite) TestRmFile_returnsErrorIfFileNotExist(t *C) {
	endpoint := api.RemoveNode.Build("child")
	req := suite.request(endpoint, nil)

	resp, err := suite.client.Do(req)
	assert.NoError(t, err)
	assert.EqualValues(t, http.StatusNotFound, resp.StatusCode)

	assert.Equal(t, "Node with id: child does not exist\n", msg(resp))
}

func (suite *ApiTestSuite) TestRmFile_removesFileSuccessfully(t *C) {
	node, _ := suite.ng.CreateDirectory(suite.ng.RootNode.Id, "child")
	endpoint := api.RemoveNode.Build(node.Id)
	req := suite.request(endpoint, nil)

	resp, err := suite.client.Do(req)
	assert.NoError(t, err)
	assert.EqualValues(t, http.StatusOK, resp.StatusCode)

	assert.EqualValues(t, 0, len(suite.ng.RootNode.Children()))
}

func (suite *ApiTestSuite) TestCreateNode_returnsErrorForMissingParent(t *C) {
	ni := graph.NodeInfo{
		Name: "Node",
		Mode: 755,
		Size: 1024,
		Type: "application/text",
	}

	endpoint := api.CreateNode.Build("not-a-parent")
	req := suite.request(endpoint, encode(ni))

	resp, err := suite.client.Do(req)
	assert.NoError(t, err)

	assert.EqualValues(t, http.StatusNotFound, resp.StatusCode)
}

func (suite *ApiTestSuite) TestCreateNode_ignoresParentIdInBody(t *C) {
	ni := graph.NodeInfo{
		ParentId: "abcd",
		Name:     "Node",
		Mode:     755,
		Size:     1024,
		Type:     "application/text",
	}

	endpoint := api.CreateNode.Build(graph.RootNodeId)
	req := suite.request(endpoint, encode(ni))

	resp, err := suite.client.Do(req)
	assert.NoError(t, err)

	assert.EqualValues(t, http.StatusCreated, resp.StatusCode)

	assert.EqualValues(t, 1, len(suite.ng.RootNode.Children()))
	assert.Equal(t, "Node", suite.ng.RootNode.Children()[0].Name())
}

func (suite *ApiTestSuite) TestCreateNode_ignoresMTimeInBody(t *C) {
	ni := graph.NodeInfo{
		Name:  "Node",
		Mode:  755,
		Size:  1024,
		MTime: time.Now().Add(-time.Hour * 10),
		Type:  "application/text",
	}

	endpoint := api.CreateNode.Build(graph.RootNodeId)
	req := suite.request(endpoint, encode(ni))

	resp, err := suite.client.Do(req)
	assert.NoError(t, err)

	assert.EqualValues(t, http.StatusCreated, resp.StatusCode)

	assert.EqualValues(t, 1, len(suite.ng.RootNode.Children()))
	mTime := suite.ng.RootNode.Children()[0].MTime()
	assert.True(t, mTime.Sub(time.Now()) < time.Second)
}

func (suite *ApiTestSuite) TestCreateNode_ignoresSizeInBody(t *C) {
	ni := graph.NodeInfo{
		Name: "Node",
		Mode: 755,
		Size: 1024,
		Type: "application/text",
	}

	endpoint := api.CreateNode.Build(graph.RootNodeId)
	req := suite.request(endpoint, encode(ni))

	resp, err := suite.client.Do(req)
	assert.NoError(t, err)

	assert.EqualValues(t, http.StatusCreated, resp.StatusCode)
	assert.EqualValues(t, 0, suite.ng.RootNode.Children()[0].Size())
}

func (suite *ApiTestSuite) TestCreateNode_getsTypeFromExtension(t *C) {
	ni := graph.NodeInfo{
		Name: "graph.txt",
		Mode: 755,
		Size: 1024,
	}

	endpoint := api.CreateNode.Build(graph.RootNodeId)
	req := suite.request(endpoint, encode(ni))

	resp, err := suite.client.Do(req)
	assert.NoError(t, err)

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	assert.Equal(t, "text/plain", suite.ng.RootNode.Children()[0].Type())
}

func (suite *ApiTestSuite) TestCreateNode_usesTypeInBodyIfProvided(t *C) {
	ni := graph.NodeInfo{
		Name: "graph.txt",
		Mode: 755,
		Size: 1024,
		Type: "application/json",
	}

	endpoint := api.CreateNode.Build(graph.RootNodeId)
	req := suite.request(endpoint, encode(ni))

	resp, err := suite.client.Do(req)
	assert.NoError(t, err)

	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Equal(t, "application/json", suite.ng.RootNode.Children()[0].Type())
}

func (suite *ApiTestSuite) TestCreateNode_returns400ForJunkData(t *C) {
	ni := "not real data"

	endpoint := api.CreateNode.Build(graph.RootNodeId)
	req := suite.request(endpoint, encode(ni))

	resp, err := suite.client.Do(req)
	assert.NoError(t, err)

	assert.EqualValues(t, http.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, msg(resp), "cannot unmarshal")
}

func (suite *ApiTestSuite) TestCreateNode_returns400WhenNodeExists(t *C) {
	suite.ng.CreateDirectory(suite.ng.RootNode.Id, "child")

	ni := graph.NodeInfo{
		Name: "child",
	}

	endpoint := api.CreateNode.Build(graph.RootNodeId)
	req := suite.request(endpoint, encode(ni))

	resp, err := suite.client.Do(req)
	assert.NoError(t, err)

	assert.EqualValues(t, http.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, msg(resp), "Node exists")
}

func (suite *ApiTestSuite) TestCreateNode_createsNodeSuccessfully(t *C) {
	ni := graph.NodeInfo{
		Name: "thing.txt",
		Size: 1024,
		Mode: 0755,
	}

	endpoint := api.CreateNode.Build(graph.RootNodeId)
	req := suite.request(endpoint, encode(ni))

	resp, err := suite.client.Do(req)
	assert.NoError(t, err)

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	assert.EqualValues(t, 1, len(suite.ng.RootNode.Children()))
	node := suite.ng.RootNode.Children()[0]

	assert.NotEmpty(t, node.Id)
	assert.Equal(t, suite.ng.RootNode.Id, node.Parent().Id)
	assert.Equal(t, "thing.txt", node.Name())
	assert.EqualValues(t, 0755, node.Mode().Perm())
	assert.Equal(t, "text/plain", node.Type())
	assert.True(t, node.MTime().Sub(time.Now()) < time.Second)
}

func (suite *ApiTestSuite) TestUpdateNode_updatesNode(t *C) {
	ni := graph.NodeInfo{
		Name: "thing.txt",
		Mode: 0755,
	}

	id, err := suite.createNode(suite.ng.RootNode.Id, ni)
	assert.NoError(t, err)

	ni.Name = "abcd.ghi"
	ni.Mode = 0700

	endpoint := api.UpdateNode.Build(id)
	req := suite.request(endpoint, encode(ni))

	resp, err := suite.client.Do(req)
	assert.NoError(t, err)

	assert.EqualValues(t, http.StatusOK, resp.StatusCode)

	changedNode := suite.ng.RootNode.Children()[0]
	assert.Equal(t, ni.Name, changedNode.Name())
	assert.EqualValues(t, ni.Mode, changedNode.Mode())
	assert.EqualValues(t, ni.Size, changedNode.Size())
}

func (suite *ApiTestSuite) TestUpdateNode_ignoresNewSize(t *C) {
	ni := graph.NodeInfo{
		Name: "abc.txt",
		Mode: 0755,
	}

	id, err := suite.createNode(suite.ng.RootNode.Id, ni)
	assert.NoError(t, err)

	ni.Size = 1024

	endpoint := api.UpdateNode.Build(id)
	req := suite.request(endpoint, encode(ni))

	resp, err := suite.client.Do(req)
	assert.NoError(t, err)

	assert.EqualValues(t, http.StatusOK, resp.StatusCode)

	changedNode := suite.ng.NodeWithId(id)
	assert.EqualValues(t, 0, changedNode.Size())
}

func (suite *ApiTestSuite) TestUpdateNode_returns404ForMissingNode(t *C) {
	ni := graph.NodeInfo{
		Name: "thing.txt",
		Mode: 0755,
		Size: 1024,
	}

	req := suite.request(api.UpdateNode.Build("not-an-id"), encode(ni))

	resp, err := suite.client.Do(req)
	assert.NoError(t, err)

	assert.EqualValues(t, http.StatusNotFound, resp.StatusCode)
}

func (suite *ApiTestSuite) TestUpdateNode_movesFileSuccessfully(t *C) {
	folder, err := suite.ng.CreateDirectory(suite.ng.RootNode.Id, "folder 1")
	assert.NoError(t, err)

	nodeInfo := graph.NodeInfo{
		Name: "file.txt",
		Mode: 0755,
	}
	id, err := suite.createNodeWithSize(suite.ng.RootNode.Id, nodeInfo, 1024)
	assert.NoError(t, err)

	nodeInfo.ParentId = folder.Id

	req := suite.request(api.UpdateNode.Build(id), encode(nodeInfo))
	resp, err := suite.client.Do(req)
	assert.NoError(t, err)
	assert.Empty(t, msg(resp))

	assert.Len(t, folder.Children(), 1)
	assert.Equal(t, "file.txt", folder.Children()[0].Name())
}

func (suite *ApiTestSuite) TestUpdateNode_renameAndMoveWorksSuccessfully(t *C) {
	folder, err := suite.ng.CreateDirectory(suite.ng.RootNode.Id, "folder 1")
	assert.NoError(t, err)

	nodeInfo := graph.NodeInfo{
		Name: "file.txt",
		Mode: 0755,
	}
	id, err := suite.createNodeWithSize(suite.ng.RootNode.Id, nodeInfo, 1024)
	assert.NoError(t, err)

	nodeInfo.Name = "file.pdf"
	nodeInfo.ParentId = folder.Id

	req := suite.request(api.UpdateNode.Build(id), encode(nodeInfo))
	resp, err := suite.client.Do(req)
	assert.NoError(t, err)
	assert.Empty(t, msg(resp))

	assert.Len(t, folder.Children(), 1)
	assert.Equal(t, "file.pdf", folder.Children()[0].Name())
}

func (suite *ApiTestSuite) TestBlocks_listsBlocksForNode(t *C) {
	ni := graph.NodeInfo{
		Name: "thing.txt",
		Mode: 0755,
	}

	id, err := suite.createNode(suite.ng.RootNode.Id, ni)
	assert.NoError(t, err)

	hash, err := suite.writeBlock(graph.BLOCK_SIZE, 0, id)
	assert.NoError(t, err)

	req := suite.request(api.ListBlocks.Build(id), nil)

	resp, err := suite.client.Do(req)
	assert.NoError(t, err)

	assert.EqualValues(t, http.StatusOK, resp.StatusCode)

	var blocks []graph.BlockInfo
	decode(resp, &blocks)

	assert.Len(t, blocks, 1)
	assert.EqualValues(t, 0, blocks[0].Offset)
	assert.Equal(t, hash, blocks[0].Hash)
}

func (suite *ApiTestSuite) TestBlocks_returns404ForMissingNode(t *C) {
	req := suite.request(api.ListBlocks.Build("abcd"), nil)

	resp, err := suite.client.Do(req)
	assert.NoError(t, err)
	assert.EqualValues(t, http.StatusNotFound, resp.StatusCode)
}

func (suite *ApiTestSuite) TestBlocks_returns400ForDir(t *C) {
	req := suite.request(api.ListBlocks.Build(graph.RootNodeId), nil)

	resp, err := suite.client.Do(req)
	assert.NoError(t, err)
	assert.EqualValues(t, http.StatusBadRequest, resp.StatusCode)
}

func (suite *ApiTestSuite) TestWriteBlock_returns404ForMissingNode(t *C) {
	_, data := fileData(graph.MEGABYTE)
	req := suite.request(api.WriteBlock.Build("node", 0), data)

	resp, err := suite.client.Do(req)
	assert.NoError(t, err)
	assert.EqualValues(t, http.StatusNotFound, resp.StatusCode)
}

func (suite *ApiTestSuite) TestWriteBlock_returns201OnSuccess(t *C) {
	nodeInfo := graph.NodeInfo{
		Name: "child.txt",
		Mode: 0755,
	}

	id, err := suite.createNodeWithSize(graph.RootNodeId, nodeInfo, graph.MEGABYTE)
	assert.NoError(t, err)

	node := suite.ng.NodeWithId(id)
	path := graph.LocationOnDisk(node.Blocks()[0].Hash)
	assert.NotEmpty(t, path)

	fi, err := os.Stat(path)
	assert.NoError(t, err)
	assert.EqualValues(t, graph.MEGABYTE, fi.Size())
}

func (suite *ApiTestSuite) TestWriteBlock_returns400ForMismatchedHashes(t *C) {
	nodeInfo := graph.NodeInfo{
		Name: "child.txt",
		Mode: 0755,
	}

	id, err := suite.createNode(graph.RootNodeId, nodeInfo)
	assert.NoError(t, err)

	_, dat := fileData(graph.MEGABYTE)

	req := suite.request(api.WriteBlock.Build(id, 0), dat)
	req.Header.Add("Content-Hash", "bad hash")

	resp, err := suite.client.Do(req)
	assert.NoError(t, err)
	assert.EqualValues(t, http.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, msg(resp), "does not match")
}

func (suite *ApiTestSuite) TestWriteBlock_returns400ForInvalidOffset(t *C) {
	nodeInfo := graph.NodeInfo{
		Name: "child.txt",
		Mode: 0755,
	}

	id, err := suite.createNode(graph.RootNodeId, nodeInfo)
	assert.NoError(t, err)

	offset := 12
	hash, dat := fileData(graph.MEGABYTE)
	req := suite.request(api.WriteBlock.Build(id, offset), dat)
	req.Header.Add("Content-Hash", hash)

	resp, err := suite.client.Do(req)
	assert.NoError(t, err)
	assert.EqualValues(t, http.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, msg(resp), fmt.Sprintf("%d is not a valid offset", offset))
}

func (suite *ApiTestSuite) TestWriteBlock_returns400OnNoData(t *C) {
	nodeInfo := graph.NodeInfo{
		Name: "child.txt",
		Mode: 0755,
	}

	id, err := suite.createNode(graph.RootNodeId, nodeInfo)
	assert.NoError(t, err)

	offset := 12
	req := suite.request(api.WriteBlock.Build(id, offset), nil)
	resp, err := suite.client.Do(req)

	assert.NoError(t, err)
	assert.EqualValues(t, http.StatusBadRequest, resp.StatusCode)
}

func (suite *ApiTestSuite) TestWriteBlock_returns400ForJunkOffest(t *C) {
	nodeInfo := graph.NodeInfo{
		Name: "child.txt",
		Mode: 0755,
	}

	id, err := suite.createNode(graph.RootNodeId, nodeInfo)
	assert.NoError(t, err)

	hash, dat := fileData(graph.MEGABYTE)

	offset := "junk-stuff"
	req := suite.request(api.WriteBlock.Build(id, offset), dat)
	req.Header.Add("Content-Hash", hash)

	resp, err := suite.client.Do(req)

	assert.NoError(t, err)
	assert.EqualValues(t, http.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, msg(resp), "Invalid offset parameter: "+offset)
}

func (suite *ApiTestSuite) TestReadBlock_returns404ForMissingNode(t *C) {
	req := suite.request(api.ReadBlock.Build("abcd", 0), nil)
	resp, err := suite.client.Do(req)
	assert.NoError(t, err)

	assert.EqualValues(t, http.StatusNotFound, resp.StatusCode)
}

func (suite *ApiTestSuite) TestReadBlock_returns400ForDir(t *C) {
	id, err := suite.createNode(graph.RootNodeId, graph.NodeInfo{
		Name: "child",
		Mode: os.FileMode(755 | uint32(os.ModeDir)),
	})
	assert.NoError(t, err)

	req := suite.request(api.ReadBlock.Build(id, 0), nil)

	resp, err := suite.client.Do(req)
	assert.NoError(t, err)

	assert.EqualValues(t, http.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, msg(resp), fmt.Sprintf("Requested node id %s is a directory", id))
}

func (suite *ApiTestSuite) TestReadBlock_returns404ForMisalignedOffset(t *C) {
	id, err := suite.createNodeWithSize(graph.RootNodeId, graph.NodeInfo{
		Name: "child",
		Mode: 755,
	}, graph.BLOCK_SIZE)
	assert.NoError(t, err)

	offset := 1
	req := suite.request(api.ReadBlock.Build(id, offset), nil)

	resp, err := suite.client.Do(req)
	assert.NoError(t, err)

	assert.EqualValues(t, http.StatusNotFound, resp.StatusCode)
	assert.Contains(t, msg(resp), fmt.Sprintf("Block at offset %d not found", offset))
}

func (suite *ApiTestSuite) TestReadBlock_returns400ForJunkOffset(t *C) {
	id, err := suite.createNodeWithSize(graph.RootNodeId, graph.NodeInfo{
		Name: "child",
		Mode: 755,
	}, graph.BLOCK_SIZE)
	assert.NoError(t, err)

	offset := "junk"
	req := suite.request(api.ReadBlock.Build(id, offset), nil)

	resp, err := suite.client.Do(req)
	assert.NoError(t, err)

	assert.EqualValues(t, http.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, msg(resp), fmt.Sprintf("Invalid offset parameter: %s", offset))
}

func (suite *ApiTestSuite) TestReadBlock_returns404WhenBlockDoesntExist(t *C) {
	id, err := suite.createNode(graph.RootNodeId, graph.NodeInfo{
		Name: "child.txt",
		Mode: 755,
	})
	assert.NoError(t, err)

	offset := 0
	req := suite.request(api.ReadBlock.Build(id, offset), nil)

	resp, err := suite.client.Do(req)
	assert.NoError(t, err)

	assert.EqualValues(t, http.StatusNotFound, resp.StatusCode)
	assert.Contains(t, msg(resp), fmt.Sprintf("Block at offset %d not found", offset))
}

func (suite *ApiTestSuite) TestReadBlock_redirectsWhenBlockFound(t *C) {
	id, err := suite.createNodeWithSize(graph.RootNodeId, graph.NodeInfo{
		Name: "child",
		Mode: 755,
	}, graph.MEGABYTE)
	assert.NoError(t, err)

	offset := 0
	req := suite.request(api.ReadBlock.Build(id, offset), nil)

	resp, err := suite.client.Do(req)
	assert.NoError(t, err)

	assert.EqualValues(t, http.StatusOK, resp.StatusCode)
	assert.EqualValues(t, graph.MEGABYTE, resp.ContentLength)
}

// Helpers
func (suite *ApiTestSuite) createNode(parentId string, nodeInfo graph.NodeInfo) (string, error) {
	req := suite.request(api.CreateNode.Build(parentId), encode(nodeInfo))
	if resp, err := suite.client.Do(req); err != nil {
		return "", err
	} else if resp.StatusCode != http.StatusCreated {
		return "", errors.New(msg(resp))
	} else {
		defer resp.Body.Close()
		decoder := json.NewDecoder(resp.Body)
		var created graph.NodeInfo
		decoder.Decode(&created)
		return created.Id, nil
	}
}

func (suite *ApiTestSuite) createNodeWithSize(parentId string, nodeInfo graph.NodeInfo, size int) (string, error) {
	if nodeInfo.Mode&os.ModeDir > 0 {
		return "", errors.New("Trying to create directory with size")
	}

	if id, err := suite.createNode(parentId, nodeInfo); err != nil {
		return "", err
	} else {
		var err error
		for i := 0; i < size && err == nil; i += graph.BLOCK_SIZE {
			uploadSize := graph.BLOCK_SIZE
			if size-i < graph.BLOCK_SIZE {
				uploadSize = size - i
			}
			_, err = suite.writeBlock(uploadSize, i, id)
		}

		return id, err
	}
}

func (suite *ApiTestSuite) writeBlock(size, offset int, id string) (string, error) {
	hash, data := fileData(size)

	req := suite.request(api.WriteBlock.Build(id, offset), data)
	req.Header.Add("Content-Hash", hash)
	if resp, err := suite.client.Do(req); err != nil {
		return "", err
	} else if resp.StatusCode != http.StatusCreated {
		return "", errors.New(msg(resp))
	}

	return hash, nil
}

func (suite *ApiTestSuite) request(endpoint api.Endpoint, body io.Reader) *http.Request {
	req, _ := http.NewRequest(endpoint.Verb, endpointFmt(suite.server.URL, endpoint.String()), body)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}

func encode(v interface{}) io.Reader {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.Encode(&v)

	return &buf
}

func decode(resp *http.Response, v interface{}) {
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	decoder.Decode(v)
}

func fileData(size int) (string, io.Reader) {
	dat := testutils.RandDat(size)
	return graph.Hash(dat), bytes.NewBuffer(dat)
}

func msg(resp *http.Response) string {
	defer resp.Body.Close()
	dat, _ := ioutil.ReadAll(resp.Body)
	return string(dat)

}

func endpointFmt(baseUrl, method string) string {
	return baseUrl + "/v1" + method
}
