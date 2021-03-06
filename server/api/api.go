package api

import (
	"encoding/gob"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"

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

	v1Router.HandleFunc(ListNodes.Template(), restApi.ListNodes).Methods(ListNodes.Verb)
	v1Router.HandleFunc(ListBlocks.Template(), restApi.Blocks).Methods(ListBlocks.Verb)
	v1Router.HandleFunc(WriteBlock.Template(), restApi.WriteBlock).Methods(WriteBlock.Verb)
	v1Router.HandleFunc(RemoveNode.Template(), restApi.RemoveNode).Methods(RemoveNode.Verb)
	v1Router.HandleFunc(CreateNode.Template(), restApi.CreateNode).Methods(CreateNode.Verb)
	v1Router.HandleFunc(UpdateNode.Template(), restApi.UpdateNode).Methods(UpdateNode.Verb)
	v1Router.HandleFunc(ReadBlock.Template(), restApi.ReadBlock).Methods(ReadBlock.Verb)
	v1Router.HandleFunc(DownloadNode.Template(), restApi.DownloadFile).Methods(ReadBlock.Verb)

	blockServer := http.FileServer(http.Dir(env.EnvPath(env.DataPath)))
	r.Handle("/block/{blockId}", http.StripPrefix("/block/", blockServer))

	return restApi
}

func encoderFromHeader(w io.Writer, header http.Header) Encoder {
	switch header.Get("Accept") {
	case string(GobEncoding):
		return gob.NewEncoder(w)
	case string(XmlEncoding):
		return xml.NewEncoder(w)
	default:
		return json.NewEncoder(w)
	}
}

func decoderFromHeader(body io.Reader, header http.Header) Decoder {
	switch header.Get("Content-Type") {
	case string(GobEncoding):
		return gob.NewDecoder(body)
	case string(XmlEncoding):
		return xml.NewDecoder(body)
	default:
		return json.NewDecoder(body)
	}
}

// GET v1/node/{nodeId}/stream
func (restApi OlympusApi) DownloadFile(writer http.ResponseWriter, req *http.Request) {
	node := restApi.graph.NodeWithId(paramFromRequest("nodeId", req))
	if !node.Exists() {
		writeNodeNotFoundError(node.Id, req, writer)
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

// GET v1/node/{parentId}?watermark=<int>&limit=<int>
func (restApi OlympusApi) ListNodes(writer http.ResponseWriter, req *http.Request) {
	parentNode := restApi.graph.NodeWithId(paramFromRequest("parentId", req))
	if !parentNode.Exists() {
		writeNodeNotFoundError(parentNode.Id, req, writer)
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

	dataResponse(restApi.listNodes(parentNode, watermark, limit), http.StatusOK, req, writer)
}

// DELETE /v1/node/{nodeId}
func (restApi OlympusApi) RemoveNode(writer http.ResponseWriter, req *http.Request) {
	node := restApi.graph.NodeWithId(paramFromRequest("nodeId", req))
	if !node.Exists() {
		writeNodeNotFoundError(node.Id, req, writer)
		return
	}

	err := restApi.graph.RemoveNode(node)
	if err != nil {
		errorResponse(ApiError{INTERNAL, err.Error()}, http.StatusInternalServerError, req, writer)
	} else {
		writer.WriteHeader(http.StatusOK)
	}
}

// POST v1/node/{parentId}
// body -> {nodeInfo}
// returns -> {nodeInfo}
func (restApi OlympusApi) CreateNode(writer http.ResponseWriter, req *http.Request) {
	parent := restApi.graph.NodeWithId(paramFromRequest("parentId", req))
	if !parent.Exists() {
		writeNodeNotFoundError(parent.Id, req, writer)
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
		errorResponse(ApiError{NODE_EXISTS, node.Id}, http.StatusBadRequest, req, writer)
	} else {
		if newNode, err := restApi.graph.NewNode(nodeInfo.Name, parent.Id, nodeInfo.Mode); err != nil {
			errorResponse(ApiError{INVALID_PARAM, err.Error()}, http.StatusBadRequest, req, writer)
		} else {
			dataResponse(newNode.NodeInfo(), http.StatusCreated, req, writer)
		}
	}
}

// PATCH v1/node/{nodeId}
// body -> {nodeInfo}
func (restApi OlympusApi) UpdateNode(writer http.ResponseWriter, req *http.Request) {
	node := restApi.graph.NodeWithId(paramFromRequest("nodeId", req))
	if !node.Exists() {
		writeNodeNotFoundError(node.Id, req, writer)
		return
	}

	var nodeInfo graph.NodeInfo
	defer req.Body.Close()
	decoder := decoderFromHeader(req.Body, req.Header)
	err := decoder.Decode(&nodeInfo)

	if err != nil {
		errorResponse(ApiError{INVALID_PARAM, err.Error()}, http.StatusBadRequest, req, writer)
		return
	}

	if err = node.Update(nodeInfo); err != nil {
		errorResponse(ApiError{INTERNAL, err.Error()}, http.StatusInternalServerError, req, writer)
	} else {
		writer.WriteHeader(http.StatusOK)
	}
}

// GET v1/node/{nodeId}/blocks
// returns -> [BlockInfo] (hashes associated with this file)
func (restApi OlympusApi) Blocks(writer http.ResponseWriter, req *http.Request) {
	node := restApi.graph.NodeWithId(paramFromRequest("nodeId", req))
	if !node.Exists() {
		writeNodeNotFoundError(node.Id, req, writer)
		return
	} else if node.IsDir() {
		errorResponse(ApiError{IS_DIRECTORY, node.Id}, http.StatusBadRequest, req, writer)
		return
	}

	blocks := node.Blocks()

	dataResponse(blocks, http.StatusOK, req, writer)
}

// PUT v1/node/{nodeId}/{offset}
func (restApi OlympusApi) WriteBlock(writer http.ResponseWriter, req *http.Request) {
	node := restApi.graph.NodeWithId(paramFromRequest("nodeId", req))
	if !node.Exists() {
		writeNodeNotFoundError(node.Id, req, writer)
		return
	}

	defer req.Body.Close()
	var data []byte
	var err error
	if data, err = ioutil.ReadAll(req.Body); err != nil {
		errorResponse(ApiError{INVALID_PARAM, err.Error()}, http.StatusBadRequest, req, writer)
		return
	}

	headerHash := req.Header.Get("Content-Hash")
	dataHash := graph.Hash(data)

	if dataHash != headerHash {
		errorResponse(ApiError{INCONGRUOUS_HASH, ""}, http.StatusBadRequest, req, writer)
		return
	}

	offsetString := paramFromRequest("offset", req)
	if offset, err := strconv.ParseInt(offsetString, 10, 64); err != nil {
		errorResponse(ApiError{INVALID_PARAM, fmt.Sprintf("Offset parameter: %s", offsetString)}, http.StatusBadRequest, req, writer)
	} else if err := node.WriteData(data, offset); err != nil {
		errorResponse(ApiError{INVALID_PARAM, err.Error()}, http.StatusBadRequest, req, writer)
	} else {
		writer.WriteHeader(http.StatusCreated)
	}
}

// GET v1/node/{nodeId}/{offset}
func (restApi OlympusApi) ReadBlock(writer http.ResponseWriter, req *http.Request) {
	node := restApi.graph.NodeWithId(paramFromRequest("nodeId", req))
	if !node.Exists() {
		writeNodeNotFoundError(node.Id, req, writer)
		return
	} else if node.IsDir() {
		errorResponse(ApiError{IS_DIRECTORY, node.Id}, http.StatusBadRequest, req, writer)
		return
	}

	offsetString := paramFromRequest("offset", req)
	if offset, err := strconv.ParseInt(offsetString, 10, 64); err != nil {
		errorResponse(ApiError{INVALID_PARAM, fmt.Sprintf("Offset parameter: %s", offsetString)}, http.StatusBadRequest, req, writer)
	} else {
		block := node.BlockWithOffset(offset)

		if block == "" {
			errorResponse(ApiError{NO_SUCH_BLOCK, offsetString}, http.StatusNotFound, req, writer)
		} else {
			http.Redirect(writer, req, "/block/"+block, http.StatusFound)
		}
	}
}

func dataResponse(data interface{}, statusCode int, req *http.Request, writer http.ResponseWriter) {
	encoder := encoderFromHeader(writer, req.Header)
	writer.WriteHeader(statusCode)
	resp := NewDataResponse(data)
	encoder.Encode(resp)
}

func errorResponse(err ApiError, statusCode int, req *http.Request, writer http.ResponseWriter) {
	encoder := encoderFromHeader(writer, req.Header)
	writer.WriteHeader(statusCode)
	resp := NewErrorResponse(&err)
	encoder.Encode(resp)
}

func paramFromRequest(key string, req *http.Request) string {
	vars := mux.Vars(req)
	return vars[key]
}

func writeNodeNotFoundError(id string, req *http.Request, writer http.ResponseWriter) {
	errorResponse(ApiError{NO_SUCH_NODE, id}, http.StatusNotFound, req, writer)
}
