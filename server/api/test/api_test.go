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

	. "github.com/sdcoffey/olympus/checkers"
	"github.com/sdcoffey/olympus/graph"
	"github.com/sdcoffey/olympus/graph/testutils"
	"github.com/sdcoffey/olympus/server/api"
	. "gopkg.in/check.v1"
)

func (suite *ApiTestSuite) TestListNodes_returns404IfFileNotExist(t *C) {
	req := suite.request(api.ListNodes.Build("not-a-node"), nil)
	resp, err := suite.client.Do(req)
	t.Check(err, IsNil)

	t.Check(resp.StatusCode, Equals, http.StatusNotFound)
	t.Check(msg(resp), Matches, "Node with id: not-a-node does not exist\n")
}

func (suite *ApiTestSuite) TestListNodes_returnsFilesForValidParent(t *C) {
	endpoint := api.ListNodes.Build(graph.RootNodeId)
	req := suite.request(endpoint, nil)
	suite.ng.NewNode("child", graph.RootNodeId, os.ModeDir)

	resp, err := suite.client.Do(req)
	t.Check(err, IsNil)
	t.Check(resp.StatusCode, Equals, http.StatusOK)

	var files []graph.NodeInfo
	decode(resp, &files)

	t.Check(files, HasLen, 1)

	file := files[0]
	t.Check(file.Name, Equals, "child")
}

func (suite *ApiTestSuite) TestListNodes_returnsNFilesWhenLimitProvided(t *C) {
	req := suite.request(api.ListNodes.Build(graph.RootNodeId).Query("limit", "1"), nil)
	suite.ng.NewNode("child1", graph.RootNodeId, os.ModeDir)
	suite.ng.NewNode("child2", graph.RootNodeId, os.ModeDir)

	resp, err := suite.client.Do(req)
	t.Check(err, IsNil)
	t.Check(resp.StatusCode, Equals, http.StatusOK)

	var files []graph.NodeInfo
	decode(resp, &files)

	t.Check(files, HasLen, 1)

	file := files[0]
	t.Check(file.Name, Equals, "child1")
}

func (suite *ApiTestSuite) TestListNodes_startsWithNFileWhenWatermarkProvided(t *C) {
	endpoint := api.ListNodes.Build(graph.RootNodeId).Query("watermark", "1")
	req := suite.request(endpoint, nil)
	suite.ng.NewNode("child1", graph.RootNodeId, os.ModeDir)
	suite.ng.NewNode("child2", graph.RootNodeId, os.ModeDir)

	resp, err := suite.client.Do(req)
	t.Check(err, IsNil)
	t.Check(resp.StatusCode, Equals, http.StatusOK)

	var files []graph.NodeInfo
	decode(resp, &files)

	t.Check(files, HasLen, 1)

	file := files[0]
	t.Check(file.Name, Equals, "child2")
}

func (suite *ApiTestSuite) TestRmFile_returnsErrorIfFileNotExist(t *C) {
	endpoint := api.RemoveNode.Build("child")
	req := suite.request(endpoint, nil)

	resp, err := suite.client.Do(req)
	t.Check(err, IsNil)
	t.Check(resp.StatusCode, Equals, http.StatusNotFound)

	t.Check(msg(resp), Matches, "Node with id: child does not exist\n")
}

func (suite *ApiTestSuite) TestRmFile_removesFileSuccessfully(t *C) {
	nodeId, err := suite.createNode(graph.RootNodeId, graph.NodeInfo{
		Name: "child.txt",
		Mode: 0755,
	})
	endpoint := api.RemoveNode.Build(nodeId)
	req := suite.request(endpoint, nil)

	resp, err := suite.client.Do(req)
	t.Check(err, IsNil)
	t.Check(resp.StatusCode, Equals, http.StatusOK)

	t.Check(suite.ng.RootNode.Children(), HasLen, 0)
}

func (suite *ApiTestSuite) TestCreateNode_returnsErrorForMissingParent(t *C) {
	ni := graph.NodeInfo{
		Name: "Node",
		Mode: 755,
		Type: "application/text",
	}

	endpoint := api.CreateNode.Build("not-a-parent")
	req := suite.request(endpoint, encode(ni))

	resp, err := suite.client.Do(req)
	t.Check(err, IsNil)
	t.Check(resp.StatusCode, Equals, http.StatusNotFound)
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
	t.Check(err, IsNil)
	t.Check(resp.StatusCode, Equals, http.StatusCreated)

	t.Assert(suite.ng.RootNode.Children(), HasLen, 1)
	t.Check(suite.ng.RootNode.Children()[0].Name(), Equals, "Node")
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
	t.Check(err, IsNil)
	t.Check(resp.StatusCode, Equals, http.StatusCreated)

	t.Assert(suite.ng.RootNode.Children(), HasLen, 1)
	mTime := suite.ng.RootNode.Children()[0].MTime()
	t.Check(mTime.Sub(time.Now()) < time.Second, IsTrue)
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
	t.Check(err, IsNil)
	t.Check(resp.StatusCode, Equals, http.StatusCreated)

	t.Check(suite.ng.RootNode.Children()[0].Size(), Equals, int64(0))
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
	t.Check(err, IsNil)
	t.Check(resp.StatusCode, Equals, http.StatusCreated)

	t.Check(suite.ng.RootNode.Children()[0].Type(), Equals, "text/plain")
}

func (suite *ApiTestSuite) TestCreateNode_returns400ForJunkData(t *C) {
	ni := "not real data"

	endpoint := api.CreateNode.Build(graph.RootNodeId)
	req := suite.request(endpoint, encode(ni))

	resp, err := suite.client.Do(req)
	t.Check(err, IsNil)
	t.Check(resp.StatusCode, Equals, http.StatusBadRequest)

	t.Check(msg(resp), Contains, "cannot unmarshal")
}

func (suite *ApiTestSuite) TestCreateNode_returns400WhenNodeExists(t *C) {
	suite.ng.NewNode("child", graph.RootNodeId, os.ModeDir)

	ni := graph.NodeInfo{
		Name: "child",
	}

	endpoint := api.CreateNode.Build(graph.RootNodeId)
	req := suite.request(endpoint, encode(ni))

	resp, err := suite.client.Do(req)
	t.Check(err, IsNil)
	t.Check(resp.StatusCode, Equals, http.StatusBadRequest)

	t.Check(msg(resp), Contains, "Node exists")
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
	t.Check(err, IsNil)
	t.Check(resp.StatusCode, Equals, http.StatusCreated)

	t.Assert(suite.ng.RootNode.Children(), HasLen, 1)
	node := suite.ng.RootNode.Children()[0]

	t.Check(node.Id, Not(Equals), "")
	t.Check(node.Parent().Id, Equals, graph.RootNodeId)
	t.Check(node.Name(), Equals, "thing.txt")
	t.Check(node.Mode(), Equals, os.FileMode(0755))
	t.Check(node.Type(), Equals, "text/plain")
	t.Check(node.MTime().Sub(time.Now()) < time.Second, IsTrue)
}

func (suite *ApiTestSuite) TestUpdateNode_updatesNode(t *C) {
	ni := graph.NodeInfo{
		Name: "thing.txt",
		Mode: 0755,
	}

	id, err := suite.createNode(suite.ng.RootNode.Id, ni)
	t.Check(err, IsNil)

	ni.Name = "abcd.ghi"
	ni.Mode = 0700

	endpoint := api.UpdateNode.Build(id)
	req := suite.request(endpoint, encode(ni))

	resp, err := suite.client.Do(req)
	t.Check(err, IsNil)
	t.Check(resp.StatusCode, Equals, http.StatusOK)

	changedNode := suite.ng.RootNode.Children()[0]
	t.Check(changedNode.Name(), Equals, ni.Name)
	t.Check(changedNode.Mode(), Equals, ni.Mode)
	t.Check(changedNode.Size(), Equals, ni.Size)
}

func (suite *ApiTestSuite) TestUpdateNode_ignoresNewSize(t *C) {
	ni := graph.NodeInfo{
		ParentId: graph.RootNodeId,
		Name:     "abc.txt",
		Mode:     0755,
	}

	id, err := suite.createNode(graph.RootNodeId, ni)
	t.Check(err, IsNil)

	ni.Size = 1024

	endpoint := api.UpdateNode.Build(id)
	req := suite.request(endpoint, encode(ni))

	resp, err := suite.client.Do(req)
	t.Check(err, IsNil)
	t.Check(resp.StatusCode, Equals, http.StatusOK)

	changedNode := suite.ng.NodeWithId(id)
	t.Check(changedNode.Size(), Equals, int64(0))
}

func (suite *ApiTestSuite) TestUpdateNode_returns404ForMissingNode(t *C) {
	ni := graph.NodeInfo{
		Name: "thing.txt",
		Mode: 0755,
		Size: 1024,
	}

	req := suite.request(api.UpdateNode.Build("not-an-id"), encode(ni))

	resp, err := suite.client.Do(req)
	t.Check(err, IsNil)
	t.Check(resp.StatusCode, Equals, http.StatusNotFound)
}

func (suite *ApiTestSuite) TestUpdateNode_movesFileSuccessfully(t *C) {
	folder, err := suite.ng.NewNode("folder 1", graph.RootNodeId, os.ModeDir)
	t.Check(err, IsNil)

	nodeInfo := graph.NodeInfo{
		Name: "file.txt",
		Mode: 0755,
	}
	id, err := suite.createNodeWithSize(suite.ng.RootNode.Id, nodeInfo, 1024)
	t.Check(err, IsNil)

	nodeInfo.ParentId = folder.Id

	req := suite.request(api.UpdateNode.Build(id), encode(nodeInfo))
	resp, err := suite.client.Do(req)
	t.Check(err, IsNil)
	t.Check(msg(resp), Equals, "")

	t.Check(folder.Children(), HasLen, 1)
	t.Check(folder.Children()[0].Name(), Equals, "file.txt")
}

func (suite *ApiTestSuite) TestUpdateNode_renameAndMoveWorksSuccessfully(t *C) {
	folder, err := suite.ng.NewNode("folder 1", graph.RootNodeId, os.ModeDir)
	t.Check(err, IsNil)

	nodeInfo := graph.NodeInfo{
		Name: "file.txt",
		Mode: 0755,
	}
	id, err := suite.createNodeWithSize(suite.ng.RootNode.Id, nodeInfo, 1024)
	t.Check(err, IsNil)

	nodeInfo.Name = "file.pdf"
	nodeInfo.ParentId = folder.Id

	req := suite.request(api.UpdateNode.Build(id), encode(nodeInfo))
	resp, err := suite.client.Do(req)
	t.Check(err, IsNil)
	t.Check(msg(resp), Equals, "")

	t.Check(folder.Children(), HasLen, 1)
	t.Check(folder.Children()[0].Name(), Equals, "file.pdf")
}

func (suite *ApiTestSuite) TestBlocks_listsBlocksForNode(t *C) {
	ni := graph.NodeInfo{
		Name: "thing.txt",
		Mode: 0755,
	}

	id, err := suite.createNode(graph.RootNodeId, ni)
	t.Check(err, IsNil)

	hash, err := suite.writeBlock(graph.BLOCK_SIZE, 0, id)
	t.Check(err, IsNil)

	req := suite.request(api.ListBlocks.Build(id), nil)

	resp, err := suite.client.Do(req)
	t.Check(err, IsNil)

	t.Check(resp.StatusCode, Equals, http.StatusOK)

	var blocks []graph.BlockInfo
	decode(resp, &blocks)

	t.Assert(blocks, HasLen, 1)
	t.Check(blocks[0].Offset, Equals, int64(0))
	t.Check(blocks[0].Hash, Equals, hash)
}

func (suite *ApiTestSuite) TestBlocks_returns404ForMissingNode(t *C) {
	req := suite.request(api.ListBlocks.Build("abcd"), nil)

	resp, err := suite.client.Do(req)
	t.Check(err, IsNil)
	t.Check(resp.StatusCode, Equals, http.StatusNotFound)
}

func (suite *ApiTestSuite) TestBlocks_returns400ForDir(t *C) {
	req := suite.request(api.ListBlocks.Build(graph.RootNodeId), nil)

	resp, err := suite.client.Do(req)
	t.Check(err, IsNil)
	t.Check(resp.StatusCode, Equals, http.StatusBadRequest)
}

func (suite *ApiTestSuite) TestWriteBlock_returns404ForMissingNode(t *C) {
	_, data := fileData(graph.MEGABYTE)
	req := suite.request(api.WriteBlock.Build("node", 0), data)

	resp, err := suite.client.Do(req)
	t.Check(err, IsNil)
	t.Check(resp.StatusCode, Equals, http.StatusNotFound)
}

func (suite *ApiTestSuite) TestWriteBlock_returns201OnSuccess(t *C) {
	nodeInfo := graph.NodeInfo{
		Name: "child.txt",
		Mode: 0755,
	}

	id, err := suite.createNodeWithSize(graph.RootNodeId, nodeInfo, graph.MEGABYTE)
	t.Check(err, IsNil)

	node := suite.ng.NodeWithId(id)
	path := graph.LocationOnDisk(node.Blocks()[0].Hash)
	t.Check(path, Not(Equals), "")

	fi, err := os.Stat(path)
	t.Check(err, IsNil)
	t.Check(fi.Size(), Equals, int64(graph.MEGABYTE))
}

func (suite *ApiTestSuite) TestWriteBlock_returns400ForMismatchedHashes(t *C) {
	nodeInfo := graph.NodeInfo{
		Name: "child.txt",
		Mode: 0755,
	}

	id, err := suite.createNode(graph.RootNodeId, nodeInfo)
	t.Check(err, IsNil)

	_, dat := fileData(graph.MEGABYTE)

	req := suite.request(api.WriteBlock.Build(id, 0), dat)
	req.Header.Add("Content-Hash", "bad hash")

	resp, err := suite.client.Do(req)
	t.Check(err, IsNil)
	t.Check(resp.StatusCode, Equals, http.StatusBadRequest)
	t.Check(msg(resp), Contains, "does not match")
}

func (suite *ApiTestSuite) TestWriteBlock_returns400ForInvalidOffset(t *C) {
	nodeInfo := graph.NodeInfo{
		Name: "child.txt",
		Mode: 0755,
	}

	id, err := suite.createNode(graph.RootNodeId, nodeInfo)
	t.Check(err, IsNil)

	offset := 12
	hash, dat := fileData(graph.MEGABYTE)
	req := suite.request(api.WriteBlock.Build(id, offset), dat)
	req.Header.Add("Content-Hash", hash)

	resp, err := suite.client.Do(req)
	t.Check(err, IsNil)
	t.Check(resp.StatusCode, Equals, http.StatusBadRequest)
	t.Check(msg(resp), Contains, fmt.Sprintf("%d is not a valid offset", offset))
}

func (suite *ApiTestSuite) TestWriteBlock_returns400OnNoData(t *C) {
	nodeInfo := graph.NodeInfo{
		Name: "child.txt",
		Mode: 0755,
	}

	id, err := suite.createNode(graph.RootNodeId, nodeInfo)
	t.Check(err, IsNil)

	offset := 12
	req := suite.request(api.WriteBlock.Build(id, offset), nil)
	resp, err := suite.client.Do(req)

	t.Check(err, IsNil)
	t.Check(resp.StatusCode, Equals, http.StatusBadRequest)
}

func (suite *ApiTestSuite) TestWriteBlock_returns400ForJunkOffest(t *C) {
	nodeInfo := graph.NodeInfo{
		Name: "child.txt",
		Mode: 0755,
	}

	id, err := suite.createNode(graph.RootNodeId, nodeInfo)
	t.Check(err, IsNil)

	hash, dat := fileData(graph.MEGABYTE)

	offset := "junk-stuff"
	req := suite.request(api.WriteBlock.Build(id, offset), dat)
	req.Header.Add("Content-Hash", hash)

	resp, err := suite.client.Do(req)

	t.Check(err, IsNil)
	t.Check(resp.StatusCode, Equals, http.StatusBadRequest)
	t.Check(msg(resp), Contains, "Invalid offset parameter")
}

func (suite *ApiTestSuite) TestReadBlock_returns404ForMissingNode(t *C) {
	req := suite.request(api.ReadBlock.Build("abcd", 0), nil)
	resp, err := suite.client.Do(req)
	t.Check(err, IsNil)

	t.Check(resp.StatusCode, Equals, http.StatusNotFound)
}

func (suite *ApiTestSuite) TestReadBlock_returns400ForDir(t *C) {
	id, err := suite.createNode(graph.RootNodeId, graph.NodeInfo{
		Name: "child",
		Mode: os.FileMode(755 | uint32(os.ModeDir)),
	})
	t.Check(err, IsNil)

	req := suite.request(api.ReadBlock.Build(id, 0), nil)

	resp, err := suite.client.Do(req)
	t.Check(err, IsNil)

	t.Check(resp.StatusCode, Equals, http.StatusBadRequest)
	t.Check(msg(resp), Contains, fmt.Sprintf("Requested node id %s is a directory", id))
}

func (suite *ApiTestSuite) TestReadBlock_returns404ForMisalignedOffset(t *C) {
	id, err := suite.createNodeWithSize(graph.RootNodeId, graph.NodeInfo{
		Name: "child",
		Mode: 755,
	}, graph.BLOCK_SIZE)
	t.Check(err, IsNil)

	offset := 1
	req := suite.request(api.ReadBlock.Build(id, offset), nil)

	resp, err := suite.client.Do(req)
	t.Check(err, IsNil)

	t.Check(resp.StatusCode, Equals, http.StatusNotFound)
	t.Check(msg(resp), Contains, fmt.Sprintf("Block at offset %d not found", offset))
}

func (suite *ApiTestSuite) TestReadBlock_returns400ForJunkOffset(t *C) {
	id, err := suite.createNodeWithSize(graph.RootNodeId, graph.NodeInfo{
		Name: "child",
		Mode: 755,
	}, graph.BLOCK_SIZE)
	t.Check(err, IsNil)

	offset := "junk"
	req := suite.request(api.ReadBlock.Build(id, offset), nil)

	resp, err := suite.client.Do(req)
	t.Check(err, IsNil)

	t.Check(resp.StatusCode, Equals, http.StatusBadRequest)
	t.Check(msg(resp), Contains, fmt.Sprintf("Invalid offset parameter: %s", offset))
}

func (suite *ApiTestSuite) TestReadBlock_returns404WhenBlockDoesntExist(t *C) {
	id, err := suite.createNode(graph.RootNodeId, graph.NodeInfo{
		Name: "child.txt",
		Mode: 755,
	})
	t.Check(err, IsNil)

	offset := 0
	req := suite.request(api.ReadBlock.Build(id, offset), nil)

	resp, err := suite.client.Do(req)
	t.Check(err, IsNil)

	t.Check(resp.StatusCode, Equals, http.StatusNotFound)
	t.Check(msg(resp), Contains, fmt.Sprintf("Block at offset %d not found", offset))
}

func (suite *ApiTestSuite) TestReadBlock_redirectsWhenBlockFound(t *C) {
	id, err := suite.createNodeWithSize(graph.RootNodeId, graph.NodeInfo{
		Name: "child",
		Mode: 755,
	}, graph.MEGABYTE)
	t.Check(err, IsNil)

	offset := 0
	req := suite.request(api.ReadBlock.Build(id, offset), nil)

	resp, err := suite.client.Do(req)
	t.Check(err, IsNil)

	t.Check(resp.StatusCode, Equals, http.StatusOK)
	t.Check(resp.ContentLength, Equals, int64(graph.MEGABYTE))
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
