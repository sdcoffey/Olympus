package api

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"errors"
	"fmt"

	"os"

	"github.com/sdcoffey/olympus/Godeps/_workspace/src/github.com/stretchr/testify/assert"
	"github.com/sdcoffey/olympus/graph"
)

var (
	server    *httptest.Server
	serverUrl string
	client    *http.Client
)

func init() {
	client = http.DefaultClient
}

func setup() *graph.NodeGraph {
	nodeGraph := graph.TestInit()
	server = httptest.NewServer(NewApi(nodeGraph))
	serverUrl = server.URL

	return nodeGraph
}

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

func TestListNodes_returnsErrorIfFileNotExist(t *testing.T) {
	setup()

	req := request("GET", "/ls/not-an-id", nil)
	resp, err := client.Do(req)
	assert.NoError(t, err)

	assert.EqualValues(t, http.StatusNotFound, resp.StatusCode)
	assert.Equal(t, "Node with id: not-an-id does not exist\n", msg(resp))
}

func TestLsFiles_returnsFilesForValidParent(t *testing.T) {
	nodeGraph := setup()

	req := request("GET", "/ls/"+graph.RootNodeId, nil)
	nodeGraph.CreateDirectory(nodeGraph.RootNode, "child")

	resp, err := client.Do(req)
	assert.Nil(t, err)
	assert.EqualValues(t, http.StatusOK, resp.StatusCode)

	var files []graph.NodeInfo
	decode(resp, &files)

	assert.Len(t, files, 1)

	file := files[0]
	assert.Equal(t, "child", file.Name)
}

func TestLsFiles_returnsNFilesWhenLimitProvided(t *testing.T) {
	nodeGraph := setup()
	req := request("GET", "/ls/"+graph.RootNodeId+"?limit=1", nil)
	nodeGraph.CreateDirectory(nodeGraph.RootNode, "child1")
	nodeGraph.CreateDirectory(nodeGraph.RootNode, "child2")

	resp, err := client.Do(req)
	assert.NoError(t, err)
	assert.EqualValues(t, http.StatusOK, resp.StatusCode)

	var files []graph.NodeInfo
	decode(resp, &files)

	assert.Len(t, files, 1)

	file := files[0]
	assert.Equal(t, "child1", file.Name)
}

func TestLsFiles_startsWithNFileWhenWatermarkProvided(t *testing.T) {
	nodeGraph := setup()
	req := request("GET", "/ls/"+graph.RootNodeId+"?watermark=1", nil)
	nodeGraph.CreateDirectory(nodeGraph.RootNode, "child1")
	nodeGraph.CreateDirectory(nodeGraph.RootNode, "child2")

	resp, err := client.Do(req)
	assert.NoError(t, err)
	assert.EqualValues(t, http.StatusOK, resp.StatusCode)

	var files []graph.NodeInfo
	decode(resp, &files)

	assert.Len(t, files, 1)

	file := files[0]
	assert.Equal(t, "child2", file.Name)
}

func TestRmFile_returnsErrorIfFileNotExist(t *testing.T) {
	setup()

	req := request("DELETE", "/rm/child", nil)

	resp, err := client.Do(req)
	assert.NoError(t, err)
	assert.EqualValues(t, http.StatusNotFound, resp.StatusCode)

	assert.Equal(t, "Node with id: child does not exist\n", msg(resp))
}

func TestRmFile_removesFileSuccessfully(t *testing.T) {
	nodeGraph := setup()

	node, _ := nodeGraph.CreateDirectory(nodeGraph.RootNode, "child")
	req := request("DELETE", "/rm/"+node.Id, nil)

	resp, err := client.Do(req)
	assert.NoError(t, err)
	assert.EqualValues(t, http.StatusOK, resp.StatusCode)

	assert.EqualValues(t, 0, len(nodeGraph.RootNode.Children()))
}

func TestMvFile_returnsErrorIfFileNotExist(t *testing.T) {
	nodeGraph := setup()

	req := request("PATCH", "/mv/not-an-id/"+nodeGraph.RootNode.Id, nil)

	resp, err := client.Do(req)
	assert.NoError(t, err)
	assert.EqualValues(t, http.StatusNotFound, resp.StatusCode)
	assert.Equal(t, "Node with id: not-an-id does not exist\n", msg(resp))
}

func TestMvFile_returnsErrorIfNewParentNotExist(t *testing.T) {
	nodeGraph := setup()

	node, _ := nodeGraph.CreateDirectory(nodeGraph.RootNode, "child")
	req := request("PATCH", "/mv/"+node.Id+"/not-a-parent", nil)

	resp, err := client.Do(req)
	assert.NoError(t, err)
	assert.EqualValues(t, http.StatusNotFound, resp.StatusCode)
	assert.Equal(t, "Node with id: not-a-parent does not exist\n", msg(resp))
}

func TestMvFile_movesFileSuccessfully(t *testing.T) {
	nodeGraph := setup()

	node, _ := nodeGraph.CreateDirectory(nodeGraph.RootNode, "child")
	node2, _ := nodeGraph.CreateDirectory(nodeGraph.RootNode, "child2")

	req := request("PATCH", "/mv/"+node.Id+"/"+node2.Id, nil)

	resp, err := client.Do(req)
	assert.NoError(t, err)
	assert.EqualValues(t, http.StatusOK, resp.StatusCode)

	assert.EqualValues(t, 1, len(nodeGraph.RootNode.Children()))
	assert.EqualValues(t, 1, len(node2.Children()))
}

func TestMvFile_renameInPlaceWorksSuccessfully(t *testing.T) {
	nodeGraph := setup()

	node, _ := nodeGraph.CreateDirectory(nodeGraph.RootNode, "child")

	url := fmt.Sprintf("/mv/%s/%s?rename=%s", node.Id, node.Parent().Id, "renamedChild")
	req := request("PATCH", url, nil)

	resp, err := client.Do(req)

	assert.NoError(t, err)
	assert.EqualValues(t, http.StatusOK, resp.StatusCode)

	assert.Equal(t, "renamedChild", nodeGraph.RootNode.Children()[0].Name())
}

func TestMvFile_renameAndMoveWorksSuccessfully(t *testing.T) {
	nodeGraph := setup()

	node, _ := nodeGraph.CreateDirectory(nodeGraph.RootNode, "child")
	node2, _ := nodeGraph.CreateDirectory(nodeGraph.RootNode, "child2")

	url := fmt.Sprintf("/mv/%s/%s?rename=%s", node.Id, node2.Id, "renamedChild")
	req := request("PATCH", url, nil)

	resp, err := client.Do(req)

	assert.NoError(t, err)
	assert.EqualValues(t, http.StatusOK, resp.StatusCode)

	assert.EqualValues(t, 1, len(nodeGraph.RootNode.Children()))
	assert.Equal(t, node2.Id, nodeGraph.RootNode.Children()[0].Id)

	assert.Equal(t, 1, len(node2.Children()))
	assert.Equal(t, "renamedChild", node2.Children()[0].Name())
}

func TestCreateNode_returnsErrorForMissingParent(t *testing.T) {
	setup()

	ni := graph.NodeInfo{
		Name: "Node",
		Mode: 755,
		Size: 1024,
		Type: "application/text",
	}

	url := fmt.Sprintf("/cr/%s", "not-a-parent")
	req := request("POST", url, encode(ni))

	resp, err := client.Do(req)
	assert.NoError(t, err)

	assert.EqualValues(t, http.StatusNotFound, resp.StatusCode)
}

func TestCreateNode_ignoresParentIdInBody(t *testing.T) {
	nodeGraph := setup()

	ni := graph.NodeInfo{
		ParentId: "abcd",
		Name:     "Node",
		Mode:     755,
		Size:     1024,
		Type:     "application/text",
	}

	url := fmt.Sprintf("/cr/%s", nodeGraph.RootNode.Id)
	req := request("POST", url, encode(ni))

	resp, err := client.Do(req)
	assert.NoError(t, err)

	assert.EqualValues(t, http.StatusCreated, resp.StatusCode)

	assert.EqualValues(t, 1, len(nodeGraph.RootNode.Children()))
	assert.Equal(t, "Node", nodeGraph.RootNode.Children()[0].Name())
}

func TestCreateNode_ignoresMTimeInBody(t *testing.T) {
	nodeGraph := setup()

	ni := graph.NodeInfo{
		Name:  "Node",
		Mode:  755,
		Size:  1024,
		MTime: time.Now().Add(-time.Hour * 10),
		Type:  "application/text",
	}

	url := fmt.Sprintf("/cr/%s", nodeGraph.RootNode.Id)
	req := request("POST", url, encode(ni))

	resp, err := client.Do(req)
	assert.NoError(t, err)

	assert.EqualValues(t, http.StatusCreated, resp.StatusCode)

	assert.EqualValues(t, 1, len(nodeGraph.RootNode.Children()))
	mTime := nodeGraph.RootNode.Children()[0].MTime()
	assert.True(t, mTime.Sub(time.Now()) < time.Second)
}

func TestCreateNode_ignoresSizeInBody(t *testing.T) {
	nodeGraph := setup()

	ni := graph.NodeInfo{
		Name: "Node",
		Mode: 755,
		Size: 1024,
		Type: "application/text",
	}

	url := fmt.Sprintf("/cr/%s", nodeGraph.RootNode.Id)
	req := request("POST", url, encode(ni))

	resp, err := client.Do(req)
	assert.NoError(t, err)

	assert.EqualValues(t, http.StatusCreated, resp.StatusCode)
	assert.EqualValues(t, 0, nodeGraph.RootNode.Children()[0].Size())
}

func TestCreateNode_getsTypeFromExtension(t *testing.T) {
	nodeGraph := setup()

	ni := graph.NodeInfo{
		Name: "graph.txt",
		Mode: 755,
		Size: 1024,
	}

	url := fmt.Sprintf("/cr/%s", nodeGraph.RootNode.Id)
	req := request("POST", url, encode(ni))

	resp, err := client.Do(req)
	assert.NoError(t, err)

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	assert.Equal(t, "text/plain", nodeGraph.RootNode.Children()[0].Type())
}

func TestCreateNode_usesTypeInBodyIfProvided(t *testing.T) {
	nodeGraph := setup()

	ni := graph.NodeInfo{
		Name: "graph.txt",
		Mode: 755,
		Size: 1024,
		Type: "application/json",
	}

	url := fmt.Sprintf("/cr/%s", nodeGraph.RootNode.Id)
	req := request("POST", url, encode(ni))

	resp, err := client.Do(req)
	assert.NoError(t, err)

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	assert.Equal(t, "application/json", nodeGraph.RootNode.Children()[0].Type())
}

func TestCreateNode_returns400ForJunkData(t *testing.T) {
	nodeGraph := setup()

	ni := "not real data"

	url := fmt.Sprintf("/cr/%s", nodeGraph.RootNode.Id)
	req := request("POST", url, encode(ni))

	resp, err := client.Do(req)
	assert.NoError(t, err)

	assert.EqualValues(t, http.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, msg(resp), "cannot unmarshal")
}

func TestCreateNode_returns400WhenNodeExists(t *testing.T) {
	nodeGraph := setup()

	nodeGraph.CreateDirectory(nodeGraph.RootNode, "child")

	ni := graph.NodeInfo{
		Name: "child",
	}

	url := fmt.Sprintf("/cr/%s", nodeGraph.RootNode.Id)
	req := request("POST", url, encode(ni))

	resp, err := client.Do(req)
	assert.NoError(t, err)

	assert.EqualValues(t, http.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, msg(resp), "Node exists")
}

func TestCreateNode_createsNodeSuccessfully(t *testing.T) {
	nodeGraph := setup()

	ni := graph.NodeInfo{
		Name: "thing.txt",
		Size: 1024,
		Mode: 0755,
	}

	url := fmt.Sprintf("/cr/%s", nodeGraph.RootNode.Id)
	req := request("POST", url, encode(ni))

	resp, err := client.Do(req)
	assert.NoError(t, err)

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	assert.EqualValues(t, 1, len(nodeGraph.RootNode.Children()))
	node := nodeGraph.RootNode.Children()[0]

	assert.NotEmpty(t, node.Id)
	assert.Equal(t, nodeGraph.RootNode.Id, node.Parent().Id)
	assert.Equal(t, "thing.txt", node.Name())
	assert.EqualValues(t, 0755, node.Mode().Perm())
	assert.Equal(t, "text/plain", node.Type())
	assert.True(t, node.MTime().Sub(time.Now()) < time.Second)
}

func TestUpdateNode_updatesNode(t *testing.T) {
	nodeGraph := setup()

	ni := graph.NodeInfo{
		Name: "thing.txt",
		Mode: 0755,
	}

	id, err := createNode(nodeGraph.RootNode.Id, ni)
	assert.NoError(t, err)

	ni.Name = "abcd.ghi"
	ni.Mode = 0700

	url := fmt.Sprintf("/update/%s", id)
	req := request("PATCH", url, encode(ni))

	resp, err := client.Do(req)
	assert.NoError(t, err)

	assert.EqualValues(t, http.StatusOK, resp.StatusCode)

	changedNode := nodeGraph.RootNode.Children()[0]
	assert.Equal(t, ni.Name, changedNode.Name())
	assert.EqualValues(t, ni.Mode, changedNode.Mode())
	assert.EqualValues(t, ni.Size, changedNode.Size())
}

func TestUpdateNode_ignoresNewSize(t *testing.T) {
	nodeGraph := setup()

	ni := graph.NodeInfo{
		Name: "abc.txt",
		Mode: 0755,
	}

	id, err := createNode(nodeGraph.RootNode.Id, ni)
	assert.NoError(t, err)

	ni.Size = 1024

	url := fmt.Sprintf("/update/%s", id)
	req := request("PATCH", url, encode(ni))

	resp, err := client.Do(req)
	assert.NoError(t, err)

	assert.EqualValues(t, http.StatusOK, resp.StatusCode)

	changedNode := nodeGraph.NodeWithId(id)
	assert.EqualValues(t, 0, changedNode.Size())
}

func TestUpdateNode_returns404ForMissingNode(t *testing.T) {
	setup()

	ni := graph.NodeInfo{
		Name: "thing.txt",
		Mode: 0755,
		Size: 1024,
	}

	url := "/update/not-an-id"
	req := request("PATCH", url, encode(ni))

	resp, err := client.Do(req)
	assert.NoError(t, err)

	assert.EqualValues(t, http.StatusNotFound, resp.StatusCode)
}

func TestBlocks_listsBlocksForNode(t *testing.T) {
	nodeGraph := setup()

	ni := graph.NodeInfo{
		Name: "thing.txt",
		Mode: 0755,
	}

	id, err := createNode(nodeGraph.RootNode.Id, ni)
	assert.NoError(t, err)

	hash, err := writeBlock(graph.BLOCK_SIZE, 0, id)
	assert.NoError(t, err)

	url := fmt.Sprintf("/ls/%s/blocks", id)
	req := request("GET", url, nil)

	resp, err := client.Do(req)
	assert.NoError(t, err)

	assert.EqualValues(t, http.StatusOK, resp.StatusCode)

	var blocks []graph.BlockInfo
	decode(resp, &blocks)

	assert.Len(t, blocks, 1)
	assert.EqualValues(t, 0, blocks[0].Offset)
	assert.Equal(t, hash, blocks[0].Hash)
}

func TestBlocks_returns404ForMissingNode(t *testing.T) {
	setup()

	url := fmt.Sprintf("/ls/%s/blocks", "abcd")
	req := request("GET", url, nil)

	resp, err := client.Do(req)
	assert.NoError(t, err)
	assert.EqualValues(t, http.StatusNotFound, resp.StatusCode)
}

func TestBlocks_returns400ForDir(t *testing.T) {
	nodeGraph := setup()

	url := fmt.Sprintf("/ls/%s/blocks", nodeGraph.RootNode.Id)
	req := request("GET", url, nil)

	resp, err := client.Do(req)
	assert.NoError(t, err)
	assert.EqualValues(t, http.StatusBadRequest, resp.StatusCode)
}

func TestWriteBlock_returns404ForMissingNode(t *testing.T) {
	setup()

	url := "/dd/node/0"
	_, data := fileData(graph.MEGABYTE)
	req := request("POST", url, data)

	resp, err := client.Do(req)
	assert.NoError(t, err)
	assert.EqualValues(t, http.StatusNotFound, resp.StatusCode)
}

func TestWriteBlock_returns201OnSuccess(t *testing.T) {
	nodeGraph := setup()

	nodeInfo := graph.NodeInfo{
		Name: "child.txt",
		Mode: 0755,
	}

	id, err := createNodeWithSize(graph.RootNodeId, nodeInfo, graph.MEGABYTE)
	assert.NoError(t, err)

	node := nodeGraph.NodeWithId(id)
	path := graph.LocationOnDisk(node.Blocks()[0].Hash)
	assert.NotEmpty(t, path)

	fi, err := os.Stat(path)
	assert.NoError(t, err)
	assert.EqualValues(t, graph.MEGABYTE, fi.Size())
}

func TestWriteBlock_returns400ForMismatchedHashes(t *testing.T) {
	setup()

	nodeInfo := graph.NodeInfo{
		Name: "child.txt",
		Mode: 0755,
	}

	id, err := createNode(graph.RootNodeId, nodeInfo)
	assert.NoError(t, err)

	url := fmt.Sprintf("/dd/%s/0", id)
	_, dat := fileData(graph.MEGABYTE)
	req := request("POST", url, dat)
	req.Header.Add("Content-Hash", "bad hash")

	resp, err := client.Do(req)
	assert.NoError(t, err)
	assert.EqualValues(t, http.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, msg(resp), "does not match")
}

func TestWriteBlock_returns400ForInvalidOffset(t *testing.T) {
	setup()

	nodeInfo := graph.NodeInfo{
		Name: "child.txt",
		Mode: 0755,
	}

	id, err := createNode(graph.RootNodeId, nodeInfo)
	assert.NoError(t, err)

	offset := 12
	url := fmt.Sprintf("/dd/%s/%d", id, offset)
	hash, dat := fileData(graph.MEGABYTE)
	req := request("POST", url, dat)
	req.Header.Add("Content-Hash", hash)

	resp, err := client.Do(req)
	assert.NoError(t, err)
	assert.EqualValues(t, http.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, msg(resp), fmt.Sprintf("%d is not a valid offset", offset))
}

func TestWriteBlock_returns400OnNoData(t *testing.T) {
	setup()

	nodeInfo := graph.NodeInfo{
		Name: "child.txt",
		Mode: 0755,
	}

	id, err := createNode(graph.RootNodeId, nodeInfo)
	assert.NoError(t, err)

	offset := 12
	url := fmt.Sprintf("/dd/%s/%d", id, offset)
	req := request("POST", url, nil)
	resp, err := client.Do(req)

	assert.NoError(t, err)
	assert.EqualValues(t, http.StatusBadRequest, resp.StatusCode)
}

func TestWriteBlock_returns400ForJunkOffest(t *testing.T) {
	setup()

	nodeInfo := graph.NodeInfo{
		Name: "child.txt",
		Mode: 0755,
	}

	id, err := createNode(graph.RootNodeId, nodeInfo)
	assert.NoError(t, err)

	offset := "junk-stuff"
	url := fmt.Sprintf("/dd/%s/%s", id, offset)
	hash, dat := fileData(graph.MEGABYTE)
	req := request("POST", url, dat)
	req.Header.Add("Content-Hash", hash)

	resp, err := client.Do(req)

	assert.NoError(t, err)
	assert.EqualValues(t, http.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, msg(resp), "Invalid offset parameter: "+offset)
}

func TestReadBlock_returns404ForMissingNode(t *testing.T) {
	setup()

	req := request("GET", "/cat/not-a-node/0", nil)
	resp, err := client.Do(req)
	assert.NoError(t, err)

	assert.EqualValues(t, http.StatusNotFound, resp.StatusCode)
}

func TestReadBlock_returns400ForDir(t *testing.T) {
	setup()

	id, err := createNode(graph.RootNodeId, graph.NodeInfo{
		Name: "child",
		Mode: 755 | uint32(os.ModeDir),
	})
	assert.NoError(t, err)

	url := fmt.Sprintf("/cat/%s/0", id)
	req := request("GET", url, nil)

	resp, err := client.Do(req)
	assert.NoError(t, err)

	assert.EqualValues(t, http.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, msg(resp), fmt.Sprintf("Requested node id %s is a directory", id))
}

func TestReadBlock_returns404ForMisalignedOffset(t *testing.T) {
	setup()

	id, err := createNodeWithSize(graph.RootNodeId, graph.NodeInfo{
		Name: "child",
		Mode: 755,
	}, graph.BLOCK_SIZE)
	assert.NoError(t, err)

	offset := 1
	url := fmt.Sprintf("/cat/%s/%d", id, offset)
	req := request("GET", url, nil)

	resp, err := client.Do(req)
	assert.NoError(t, err)

	assert.EqualValues(t, http.StatusNotFound, resp.StatusCode)
	assert.Contains(t, msg(resp), fmt.Sprintf("Block at offset %d not found", offset))
}

func TestReadBlock_returns400ForJunkOffset(t *testing.T) {
	setup()

	id, err := createNodeWithSize(graph.RootNodeId, graph.NodeInfo{
		Name: "child",
		Mode: 755,
	}, graph.BLOCK_SIZE)
	assert.NoError(t, err)

	offset := "junk"
	url := fmt.Sprintf("/cat/%s/%s", id, offset)
	req := request("GET", url, nil)

	resp, err := client.Do(req)
	assert.NoError(t, err)

	assert.EqualValues(t, http.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, msg(resp), fmt.Sprintf("Invalid offset parameter: %s", offset))
}

func TestReadBlock_returns404WhenBlockDoesntExist(t *testing.T) {
	setup()

	id, err := createNode(graph.RootNodeId, graph.NodeInfo{
		Name: "child.txt",
		Mode: 755,
	})
	assert.NoError(t, err)

	offset := 0
	url := fmt.Sprintf("/cat/%s/%d", id, offset)
	req := request("GET", url, nil)

	resp, err := client.Do(req)
	assert.NoError(t, err)

	assert.EqualValues(t, http.StatusNotFound, resp.StatusCode)
	assert.Contains(t, msg(resp), fmt.Sprintf("Block at offset %d not found", offset))
}

func TestReadBlock_redirectsWhenBlockFound(t *testing.T) {
	setup()

	id, err := createNodeWithSize(graph.RootNodeId, graph.NodeInfo{
		Name: "child",
		Mode: 755,
	}, graph.MEGABYTE)
	assert.NoError(t, err)

	offset := 0
	url := fmt.Sprintf("/cat/%s/%d", id, offset)
	req := request("GET", url, nil)

	resp, err := client.Do(req)
	assert.NoError(t, err)

	assert.EqualValues(t, http.StatusOK, resp.StatusCode)
	assert.EqualValues(t, graph.MEGABYTE, resp.ContentLength)
}

// Helpers

func createNode(parentId string, nodeInfo graph.NodeInfo) (string, error) {
	url := fmt.Sprintf("/cr/%s", parentId)
	req := request("POST", url, encode(nodeInfo))
	if resp, err := client.Do(req); err != nil {
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

func createNodeWithSize(parentId string, nodeInfo graph.NodeInfo, size int) (string, error) {
	if nodeInfo.Mode&uint32(os.ModeDir) > 0 {
		return "", errors.New("Trying to create directory with size")
	}

	if id, err := createNode(parentId, nodeInfo); err != nil {
		return "", err
	} else {
		uploadBlock := func(size, offset int) error {
			url := fmt.Sprintf("/dd/%s/%d", id, offset)
			hash, dat := fileData(graph.MEGABYTE)
			req := request("POST", url, dat)
			req.Header.Add("Content-Hash", hash)

			if resp, err := client.Do(req); err != nil {
				return err
			} else if resp.StatusCode != http.StatusCreated {
				return errors.New(msg(resp))
			}

			return nil
		}

		var err error
		for i := 0; i < size && err == nil; i += graph.BLOCK_SIZE {
			uploadSize := graph.BLOCK_SIZE
			if size-i < graph.BLOCK_SIZE {
				uploadSize = size - i
			}
			err = uploadBlock(uploadSize, i)
		}

		return id, err
	}
}

func writeBlock(size, offset int, id string) (string, error) {
	hash, data := fileData(size)

	url := fmt.Sprintf("/dd/%s/%d", id, offset)
	req := request("POST", url, data)
	req.Header.Add("Content-Hash", hash)
	if resp, err := client.Do(req); err != nil {
		return "", err
	} else if resp.StatusCode != http.StatusCreated {
		return "", errors.New(msg(resp))
	}

	return hash, nil
}

func Test_createNode(t *testing.T) {
	nodeGraph := setup()

	info := graph.NodeInfo{
		Name:     "thing.txt",
		ParentId: nodeGraph.RootNode.Id,
		Mode:     0755,
	}

	id, err := createNode(nodeGraph.RootNode.Id, info)
	assert.NoError(t, err)

	node := nodeGraph.NodeWithName(nodeGraph.RootNode.Id, "thing.txt")
	assert.Equal(t, node.Id, id)

	assert.Equal(t, nodeGraph.RootNode.Id, node.Parent().Id)
	assert.Equal(t, "thing.txt", node.Name())
	assert.EqualValues(t, 0755, node.Mode().Perm())
	assert.Equal(t, "text/plain", node.Type())
	assert.True(t, node.MTime().Sub(time.Now()) < time.Second)
}

func request(method, path string, body io.Reader) *http.Request {
	req, _ := http.NewRequest(method, endpointFmt(path), body)
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
	dat := graph.RandDat(size)
	return graph.Hash(dat), bytes.NewBuffer(dat)
}

func msg(resp *http.Response) string {
	defer resp.Body.Close()
	dat, _ := ioutil.ReadAll(resp.Body)
	return string(dat)

}

func endpointFmt(method string) string {
	return serverUrl + "/v1" + method
}
