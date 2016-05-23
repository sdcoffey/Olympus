package api

import (
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/sdcoffey/olympus/Godeps/_workspace/src/code.google.com/p/go-uuid/uuid"
	"github.com/sdcoffey/olympus/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/sdcoffey/olympus/env"
	"github.com/sdcoffey/olympus/graph"
	"github.com/sdcoffey/olympus/util"
)

type Encoder interface {
	Encode(interface{}) error
}

type Decoder interface {
	Decode(interface{}) error
}

type OlympusApi struct {
	http.Handler
	graph *graph.NodeGraph
}

func NewApi(ng *graph.NodeGraph) OlympusApi {
	r := mux.NewRouter()
	v1Router := r.PathPrefix("/v1").Subrouter()

	restApi := OlympusApi{r, ng}

	v1Router.HandleFunc("/ls/{parentId}", restApi.ListNodes).Methods("GET")
	v1Router.HandleFunc("/ls/{nodeId}/blocks", restApi.Blocks).Methods("GET")
	v1Router.HandleFunc("/dd/{nodeId}/{offset}", restApi.WriteBlock).Methods("POST")
	v1Router.HandleFunc("/mv/{nodeId}/{newParentId}", restApi.MoveNode).Methods("PATCH")
	v1Router.HandleFunc("/rm/{nodeId}", restApi.RemoveNode).Methods("DELETE")
	v1Router.HandleFunc("/cr/{parentId}", restApi.CreateNode).Methods("POST")
	v1Router.HandleFunc("/update/{nodeId}", restApi.UpdateNode).Methods("PATCH")
	v1Router.HandleFunc("/cat/{nodeId}/{offset}", restApi.ReadBlock).Methods("GET")
	v1Router.HandleFunc("/dl/{nodeId}", restApi.DownloadFile).Methods("GET")

	blockServer := http.FileServer(http.Dir(env.EnvPath(env.DataPath)))
	r.Handle("/block/{blockId}", http.StripPrefix("/block/", blockServer))

	return restApi
}

func encoderFromHeader(w io.Writer, header http.Header) Encoder {
	if header.Get("Accept") == "application/gob" {
		return gob.NewEncoder(w)
	} else {
		return json.NewEncoder(w)
	}
}

func decoderFromHeader(body io.Reader, header http.Header) Decoder {
	if header.Get("Content-Type") == "application/gob" {
		return gob.NewDecoder(body)
	} else {
		return json.NewDecoder(body)
	}
}

// GET v1/dl/{nodeId}
func (restApi OlympusApi) DownloadFile(writer http.ResponseWriter, req *http.Request) {
	node := restApi.graph.NodeWithId(paramFromRequest("nodeId", req))
	if !node.Exists() {
		writeNodeNotFoundError(node.Id, writer)
		return
	}

	writer.Header().Add("Content-Type", node.Type())
	http.ServeContent(writer, req, node.Name(), node.MTime(), node.ReadSeeker())
}

func (restApi OlympusApi) listNodes(parentNode *graph.Node, watermark, limit int) []graph.NodeInfo {
	minI := func(lhs, rhs int) int {
		if lhs > rhs {
			return lhs
		} else {
			return rhs
		}
	}

	children := parentNode.Children()
	var start, end int

	if watermark > 0 && watermark < len(children) {
		start = watermark
	} else {
		start = 0
	}

	if limit > 0 {
		end = minI(start+limit, len(children)) - 1
	} else {
		end = len(children)
	}

	children = children[start:end]
	response := make([]graph.NodeInfo, len(children))

	for idx, child := range children {
		response[idx] = child.NodeInfo()
	}

	return response
}

// GET v1/ls/{parentId}?watermark=<int>&limit=<int>
func (restApi OlympusApi) ListNodes(writer http.ResponseWriter, req *http.Request) {
	parentNode := restApi.graph.NodeWithId(paramFromRequest("parentId", req))
	if !parentNode.Exists() {
		writeNodeNotFoundError(parentNode.Id, writer)
		return
	}

	watermark, limit := -1, -1

	watermarkVals := req.URL.Query()["watermark"]
	limitVals := req.URL.Query()["limit"]

	if len(watermarkVals) > 0 {
		w, _ := strconv.ParseInt(watermarkVals[0], 10, 64)
		watermark = int(w)
	}
	if len(limitVals) > 0 {
		l, _ := strconv.ParseInt(limitVals[0], 10, 64)
		limit = int(l)
	}

	encoder := encoderFromHeader(writer, req.Header)
	writer.WriteHeader(http.StatusOK)
	encoder.Encode(restApi.listNodes(parentNode, watermark, limit))
}

// DELETE /v1/rm/{nodeId}
func (restApi OlympusApi) RemoveNode(writer http.ResponseWriter, req *http.Request) {
	node := restApi.graph.NodeWithId(paramFromRequest("nodeId", req))
	if !node.Exists() {
		writeNodeNotFoundError(node.Id, writer)
		return
	}

	err := restApi.graph.RemoveNode(node)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
	} else {
		writer.WriteHeader(http.StatusOK)
	}
}

// PATCH /mv/{nodeId}/{newParentId}?rename={newName}
func (restApi OlympusApi) MoveNode(writer http.ResponseWriter, req *http.Request) {
	node := restApi.graph.NodeWithId(paramFromRequest("nodeId", req))
	if !node.Exists() {
		writeNodeNotFoundError(node.Id, writer)
		return
	}

	newParent := restApi.graph.NodeWithId(paramFromRequest("newParentId", req))
	if !newParent.Exists() {
		writeNodeNotFoundError(newParent.Id, writer)
		return
	}

	newName := req.URL.Query().Get("rename")
	if newName == "" {
		newName = node.Name()
	}

	err := restApi.graph.MoveNode(node, newName, newParent.Id)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
	} else {
		writer.WriteHeader(http.StatusOK)
	}
}

// POST v1/cr/{parentId}
// body -> {nodeInfo}
// returns -> {nodeInfo}
func (restApi OlympusApi) CreateNode(writer http.ResponseWriter, req *http.Request) {
	parent := restApi.graph.NodeWithId(paramFromRequest("parentId", req))
	if !parent.Exists() {
		writeNodeNotFoundError(parent.Id, writer)
		return
	}

	var nodeInfo graph.NodeInfo
	defer req.Body.Close()
	decoder := decoderFromHeader(req.Body, req.Header)

	if err := decoder.Decode(&nodeInfo); err != nil {
		writer.WriteHeader(http.StatusBadRequest)
		writer.Write([]byte(err.Error()))
		return
	} else if node := restApi.graph.NodeWithName(parent.Id, nodeInfo.Name); node != nil && node.Exists() {
		http.Error(writer, fmt.Sprintf("Node exists, call /v1/touch/%s/ to update this object", node.Id),
			http.StatusBadRequest)
	} else {
		nodeInfo.ParentId = parent.Id
		nodeInfo.MTime = time.Now()

		if nodeInfo.Type == "" {
			nodeInfo.Type = util.MimeType(nodeInfo.Name)
		}

		newNode := restApi.graph.NodeWithNodeInfo(nodeInfo)
		newNode.Id = uuid.New()
		if err := newNode.Save(); err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
		} else {
			encoder := encoderFromHeader(writer, req.Header)
			writer.WriteHeader(http.StatusCreated)
			nodeInfo = newNode.NodeInfo()
			encoder.Encode(nodeInfo)
		}
	}
}

// PATCH v1/update/{nodeId}
// body -> {nodeInfo}
func (restApi OlympusApi) UpdateNode(writer http.ResponseWriter, req *http.Request) {
	node := restApi.graph.NodeWithId(paramFromRequest("nodeId", req))
	if !node.Exists() {
		writeNodeNotFoundError(node.Id, writer)
		return
	}

	var nodeInfo graph.NodeInfo
	defer req.Body.Close()
	decoder := decoderFromHeader(req.Body, req.Header)
	err := decoder.Decode(&nodeInfo)

	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	changes := []func() error{
		func() error {
			return restApi.graph.MoveNode(node, nodeInfo.Name, node.Parent().Id)
		},
		func() error {
			return node.Chmod(os.FileMode(nodeInfo.Mode))
		},
		func() error {
			return node.Touch(nodeInfo.MTime)
		},
	}

	for i := 0; i < len(changes) && err == nil; i++ {
		err = changes[i]()
	}

	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
	} else {
		writer.WriteHeader(http.StatusOK)
	}
}

// GET v1/ls/{nodeId}/blocks
// returns -> [BlockInfo] (hashes we don't have)
func (restApi OlympusApi) Blocks(writer http.ResponseWriter, req *http.Request) {
	node := restApi.graph.NodeWithId(paramFromRequest("nodeId", req))
	if !node.Exists() {
		writeNodeNotFoundError(node.Id, writer)
		return
	} else if node.IsDir() {
		http.Error(writer, fmt.Sprintf("Node with id: %s is a directory", node.Id), http.StatusBadRequest)
		return
	}

	blocks := node.Blocks()

	writer.WriteHeader(http.StatusOK)
	encoder := encoderFromHeader(writer, req.Header)
	encoder.Encode(blocks)
}

// POST v1/dd/{nodeId}/{offset}
func (restApi OlympusApi) WriteBlock(writer http.ResponseWriter, req *http.Request) {
	node := restApi.graph.NodeWithId(paramFromRequest("nodeId", req))
	if !node.Exists() {
		writeNodeNotFoundError(node.Id, writer)
		return
	}

	defer req.Body.Close()
	var data []byte
	var err error
	if data, err = ioutil.ReadAll(req.Body); err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	headerHash := req.Header.Get("Content-Hash")
	dataHash := graph.Hash(data)

	if dataHash != headerHash {
		http.Error(writer, "Hash in header does not match data's hash", http.StatusBadRequest)
		return
	}

	offsetString := paramFromRequest("offset", req)
	if offset, err := strconv.ParseInt(offsetString, 10, 64); err != nil {
		http.Error(writer, fmt.Sprintf("Invalid offset parameter: %s", offsetString), http.StatusBadRequest)
	} else if err := node.WriteData(data, offset); err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
	} else {
		writer.WriteHeader(http.StatusCreated)
	}
}

// GET cat/{nodeId}/{offset}
func (restApi OlympusApi) ReadBlock(writer http.ResponseWriter, req *http.Request) {
	node := restApi.graph.NodeWithId(paramFromRequest("nodeId", req))
	if !node.Exists() {
		writeNodeNotFoundError(node.Id, writer)
		return
	} else if node.IsDir() {
		http.Error(writer, fmt.Sprintf("Requested node id %s is a directory", node.Id), http.StatusBadRequest)
		return
	}

	offsetString := paramFromRequest("offset", req)
	if offset, err := strconv.ParseInt(offsetString, 10, 64); err != nil {
		http.Error(writer, fmt.Sprintf("Invalid offset parameter: %s", offsetString), http.StatusBadRequest)
	} else {
		block := node.BlockWithOffset(offset)

		if block == "" {
			http.Error(writer, fmt.Sprintf("Block at offset %d not found", offset), http.StatusNotFound)
		} else {
			http.Redirect(writer, req, "/block/"+block, http.StatusFound)
		}
	}
}

func paramFromRequest(key string, req *http.Request) string {
	vars := mux.Vars(req)
	return vars[key]
}

func writeNodeNotFoundError(id string, writer http.ResponseWriter) {
	http.Error(writer, fmt.Sprintf("Node with id: %s does not exist", id), http.StatusNotFound)
}
