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

	"code.google.com/p/go-uuid/uuid"
	"github.com/gorilla/mux"
	"github.com/sdcoffey/olympus/env"
	"github.com/sdcoffey/olympus/graph"
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
	v1Router.HandleFunc("/mv/{nodeId}/{newParentId}", restApi.MoveNode).Methods("PATCH")
	v1Router.HandleFunc("/rm/{nodeId}", restApi.RemoveNode).Methods("DELETE")
	v1Router.HandleFunc("/cr/{parentId}", restApi.CreateNode).Methods("POST")
	v1Router.HandleFunc("/update/{nodeId}", restApi.UpdateNode).Methods("PATCH")
	v1Router.HandleFunc("/dd/{nodeId}/{offset}", restApi.WriteBlock).Methods("POST")
	v1Router.HandleFunc("/cat/{nodeId}/{offset}", restApi.ReadBlock).Methods("GET")

	fileServer := http.FileServer(http.Dir(env.EnvPath(env.DataPath)))
	v1Router.Handle("/block/{blockId}", http.StripPrefix("/v1/block/", fileServer))

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

// GET v1/ls/{parentId}
func (restApi OlympusApi) ListNodes(writer http.ResponseWriter, req *http.Request) {
	node := restApi.graph.NodeWithId(paramFromRequest("parentId", req))
	if !node.Exists() {
		writeNodeNotFoundError(node, writer)
		return
	}

	children := node.Children()
	response := make([]graph.NodeInfo, len(children))

	for idx, child := range children {
		response[idx] = child.NodeInfo()
	}

	encoder := encoderFromHeader(writer, req.Header)
	writer.WriteHeader(http.StatusOK)
	encoder.Encode(response)
}

// DELETE /v1/rm/{nodeId}
func (restApi OlympusApi) RemoveNode(writer http.ResponseWriter, req *http.Request) {
	node := restApi.graph.NodeWithId(paramFromRequest("nodeId", req))
	if !node.Exists() {
		writeNodeNotFoundError(node, writer)
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
		writeNodeNotFoundError(node, writer)
		return
	}

	newParent := restApi.graph.NodeWithId(paramFromRequest("newParentId", req))
	if !newParent.Exists() {
		writeNodeNotFoundError(newParent, writer)
		return
	}

	newName := req.URL.Query().Get("rename")
	if newName == "" {
		newName = node.Name()
	}

	err := restApi.graph.MoveNode(node, newName, newParent.Id)
	if err != nil {
		http.Error(writer, err.Error(), 500)
	} else {
		writer.WriteHeader(200)
	}
}

// POST v1/cr/{parentId}
// body -> {nodeInfo}
// returns -> {nodeInfo}
func (restApi OlympusApi) CreateNode(writer http.ResponseWriter, req *http.Request) {
	parent := restApi.graph.NodeWithId(paramFromRequest("parentId", req))
	if !parent.Exists() {
		writeNodeNotFoundError(parent, writer)
		return
	}

	var nodeInfo graph.NodeInfo
	defer req.Body.Close()
	decoder := decoderFromHeader(req.Body, req.Header)

	if err := decoder.Decode(&nodeInfo); err != nil {
		writer.WriteHeader(400)
		writer.Write([]byte(err.Error()))
		return
	} else if node := restApi.graph.NodeWithName(parent.Id, nodeInfo.Name); node != nil && node.Exists() {
		http.Error(writer, fmt.Sprintf("Node exists, call /v1/touch/%s/to update this object", node.Id), 400)
	} else {
		nodeInfo.ParentId = parent.Id
		nodeInfo.MTime = time.Now()
		newNode := restApi.graph.NodeWithNodeInfo(nodeInfo)
		newNode.Id = uuid.New()
		if err := newNode.Save(); err != nil {
			http.Error(writer, err.Error(), 400)
		} else {
			encoder := encoderFromHeader(writer, req.Header)
			writer.WriteHeader(200)
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
		writeNodeNotFoundError(node, writer)
		return
	}

	var nodeInfo graph.NodeInfo
	defer req.Body.Close()
	decoder := decoderFromHeader(req.Body, req.Header)
	err := decoder.Decode(&nodeInfo)

	if err != nil {
		http.Error(writer, err.Error(), 400)
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
		func() error {
			return node.Resize(nodeInfo.Size)
		},
	}

	for i := 0; i < len(changes) && err == nil; i++ {
		err = changes[i]()
	}

	if err != nil {
		http.Error(writer, err.Error(), 500)
	} else {
		writer.WriteHeader(200)
	}
}

// GET v1/hasBlocks/{nodeId}/blocks
// returns -> [BlockInfo] (hashes we don't have)
func (restApi OlympusApi) Blocks(writer http.ResponseWriter, req *http.Request) {
	node := restApi.graph.NodeWithId(paramFromRequest("nodeId", req))
	if !node.Exists() {
		writeNodeNotFoundError(node, writer)
		return
	} else if node.IsDir() {
		http.Error(writer, fmt.Sprintf("Node with id: %s is a directory", node.Id), 400)
		return
	}

	blocks := node.Blocks()

	writer.WriteHeader(200)
	encoder := encoderFromHeader(writer, req.Header)
	encoder.Encode(blocks)
}

// POST v1/dd/{nodeId}/{offset}
func (restApi OlympusApi) WriteBlock(writer http.ResponseWriter, req *http.Request) {
	node := restApi.graph.NodeWithId(paramFromRequest("nodeId", req))
	if !node.Exists() {
		writeNodeNotFoundError(node, writer)
		return
	}

	defer req.Body.Close()
	var data []byte
	var err error
	if data, err = ioutil.ReadAll(req.Body); err != nil {
		http.Error(writer, err.Error(), 400)
		return
	}

	offsetString := paramFromRequest("offset", req)
	if offset, err := strconv.ParseInt(offsetString, 10, 64); err != nil {
		http.Error(writer, fmt.Sprintf("Invalid offset parameter: %s", offsetString), 400)
	} else if err := node.WriteData(data, offset); err != nil {
		http.Error(writer, err.Error(), 400)
	}
}

// GET cat/{nodeId}/{offset}
func (restApi OlympusApi) ReadBlock(writer http.ResponseWriter, req *http.Request) {
	node := restApi.graph.NodeWithId(paramFromRequest("nodeId", req))
	if !node.Exists() {
		writeNodeNotFoundError(node, writer)
		return
	} else if node.IsDir() {
		http.Error(writer, fmt.Sprintf("Requested node id %s is a directory", node.Id), 400)
		return
	}

	offsetString := paramFromRequest("offset", req)
	if offset, err := strconv.ParseInt(offsetString, 10, 64); err != nil {
		http.Error(writer, fmt.Sprintf("Invalid offset parameter: %s", offsetString), 400)
	} else {
		block := node.BlockWithOffset(offset)

		if block == "" {
			http.Error(writer, fmt.Sprintf("Block with id: %s does not belong to node with id: %s", block, node.Id), 404)
		} else {
			http.Redirect(writer, req, "/v1/block/"+string(block), http.StatusFound)
		}
	}
}

func paramFromRequest(key string, req *http.Request) string {
	vars := mux.Vars(req)
	return vars[key]
}

func writeNodeNotFoundError(node *graph.Node, writer http.ResponseWriter) {
	writer.WriteHeader(400)
	writer.Write([]byte(fmt.Sprintf("Node with id: %s does not exist", node.Id)))
}
