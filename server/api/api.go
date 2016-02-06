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

	"github.com/gorilla/mux"
	"github.com/sdcoffey/olympus/env"
	"github.com/sdcoffey/olympus/fs"
)

type Encoder interface {
	Encode(interface{}) error
}

type Decoder interface {
	Decode(interface{}) error
}

type OlympusApi struct {
	http.Handler
	fs *fs.Filesystem
}

func NewApi(fs *fs.Filesystem) OlympusApi {
	r := mux.NewRouter()
	v1Router := r.PathPrefix("/v1").Subrouter()

	restApi := OlympusApi{r, fs}

	v1Router.HandleFunc("/ls/{parentId}", restApi.LsFiles).Methods("GET")
	v1Router.HandleFunc("/ls/{fileId}/blocks", restApi.Blocks).Methods("GET")
	v1Router.HandleFunc("/mv/{fileId}/{newParentId}", restApi.MvFile).Methods("PATCH")
	v1Router.HandleFunc("/rm/{fileId}", restApi.Rm).Methods("DELETE")
	v1Router.HandleFunc("/mkdir/{parentId}/{name}", restApi.MkDir).Methods("POST")
	v1Router.HandleFunc("/cr/{parentId}", restApi.Cr).Methods("POST")
	v1Router.HandleFunc("/update/{fileId}", restApi.Update).Methods("PATCH")
	v1Router.HandleFunc("/dd/{fileId}/{offset}", restApi.WriteBlock).Methods("POST")
	v1Router.HandleFunc("/cat/{fileId}/{offset}", restApi.ReadBlock).Methods("GET")

	fileServer := http.FileServer(http.Dir(env.EnvPath(env.DataPath)))
	v1Router.Handle("/block/{blockId}", http.StripPrefix("/v1/block/", fileServer))

	return api
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
func (restApi OlympusApi) LsFiles(writer http.ResponseWriter, req *http.Request) {
	file := fs.FileWithId(paramFromRequest("parentId", req), restApi.fs)
	if !file.Exists() {
		writeFileNotFoundError(file, writer)
		return
	}

	children := file.Children()
	response := make([]fs.FileInfo, len(children))

	for idx, child := range children {
		response[idx] = child.FileInfo()
	}

	encoder := encoderFromHeader(writer, req.Header)
	writer.WriteHeader(http.StatusOK)
	encoder.Encode(response)
}

// DELETE /v1/rm/{fileId}
func (restApi OlympusApi) RmFile(writer http.ResponseWriter, req *http.Request) {
	file := fs.FileWithId(paramFromRequest("fileId", req), restApi.fs)
	if !file.Exists() {
		writeFileNotFoundError(file, writer)
		return
	}

	err := fs.Rm(file)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
	} else {
		writer.WriteHeader(http.StatusOK)
	}
}

// PATCH /mv/{fileId}/{newParentId}?rename={newName}
func (restApi OlympusApi) MvFile(writer http.ResponseWriter, req *http.Request) {
	file := fs.FileWithId(paramFromRequest("fileId", req), restApi.fs)
	if !file.Exists() {
		writeFileNotFoundError(file, writer)
		return
	}

	newParent := fs.FileWithId(paramFromRequest("newParentId", req), restApi.fs)
	if !newParent.Exists() {
		writeFileNotFoundError(newParent, writer)
		return
	}

	newName := req.URL.Query().Get("rename")
	if newName == "" {
		newName = file.Name()
	}

	err := restApi.fs.MoveObject(file, newName, newParent.Id)
	if err != nil {
		http.Error(writer, err.Error(), 500)
	} else {
		writer.WriteHeader(200)
	}
}

// POST v1/mkdir/{parentId}/{fileName}
func MkDir(writer http.ResponseWriter, req *http.Request) {
	parent := fs.FileWithId(paramFromRequest("parentId", req))
	if !parent.Exists() {
		writeFileNotFoundError(parent, writer)
		return
	}

	name := paramFromRequest("name", req)
	newFolder, err := parent.MkDir(name)
	if err != nil {
		http.Error(writer, err.Error(), 400)
	} else {
		encoder := encoderFromHeader(writer, req.Header)
		writer.WriteHeader(200)
		encoder.Encode(newFolder.Id)
	}
}

// POST v1/cr/{parentId}
// body -> {FileInfo}
// returns -> {FileInfo}
func (restApi OlympusApi) Cr(writer http.ResponseWriter, req *http.Request) {
	parent := fs.FileWithId(paramFromRequest("parentId", req), restApi.fs)
	if !parent.Exists() {
		writeFileNotFoundError(parent, writer)
		return
	}

	var fileInfo fs.FileInfo
	defer req.Body.Close()
	decoder := decoderFromHeader(req.Body, req.Header)

	if err := decoder.Decode(&fileInfo); err != nil {
		writer.WriteHeader(400)
		writer.Write([]byte(err.Error()))
		return
	} else if file := restApi.fs.FileWithName(parent.Id, fileInfo.Name); file != nil && file.Exists() {
		http.Error(writer, fmt.Sprintf("File exists, call /v1/touch/%s/to update this object", file.Id), 400)
	} else {
		fileInfo.ParentId = parent.Id
		newFile := fs.FileWithFileInfo(fileInfo)
		if err := newFile.Save(); err != nil {
			http.Error(writer, err.Error(), 400)
		} else {
			encoder := encoderFromHeader(writer, req.Header)
			writer.WriteHeader(200)
			fileInfo = newFile.FileInfo()
			encoder.Encode(fileInfo)
		}
	}
}

// PATCH v1/update/{fileId}
// body -> {FileInfo}
func Update(writer http.ResponseWriter, req *http.Request) {
	file := fs.FileWithId(paramFromRequest("fileId", req))
	if !file.Exists() {
		writeFileNotFoundError(file, writer)
		return
	}

	var fileInfo fs.FileInfo
	defer req.Body.Close()
	decoder := decoderFromHeader(req.Body, req.Header)
	err := decoder.Decode(&fileInfo)

	if err != nil {
		http.Error(writer, err.Error(), 400)
		return
	}

	changes := []func() error{
		func() error {
			return file.Mv(fileInfo.Name, file.Parent().Id)
		},
		func() error {
			return file.Chmod(os.FileMode(fileInfo.Attr))
		},
		func() error {
			return file.Touch(fileInfo.MTime)
		},
		func() error {
			return file.Resize(fileInfo.Size)
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

// GET v1/hasBlocks/{fileId}/blocks
// returns -> [BlockInfo] (hashes we don't have)
func Blocks(writer http.ResponseWriter, req *http.Request) {
	file := fs.FileWithId(paramFromRequest("fileId", req))
	if !file.Exists() {
		writeFileNotFoundError(file, writer)
		return
	} else if file.IsDir() {
		http.Error(writer, fmt.Sprintf("File with id: %s is a directory", file.Id), 400)
		return
	}

	blocks := file.Blocks()

	writer.WriteHeader(200)
	encoder := encoderFromHeader(writer, req.Header)
	encoder.Encode(blocks)
}

// POST v1/dd/{fileId}/{offset}
func WriteBlock(writer http.ResponseWriter, req *http.Request) {
	file := fs.FileWithId(paramFromRequest("fileId", req))
	if !file.Exists() {
		writeFileNotFoundError(file, writer)
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
	} else if err := file.WriteData(data, offset); err != nil {
		http.Error(writer, err.Error(), 400)
	}
}

// GET cat/{fileId}/{offset}
func ReadBlock(writer http.ResponseWriter, req *http.Request) {
	file := fs.FileWithId(paramFromRequest("fileId", req))
	if !file.Exists() {
		writeFileNotFoundError(file, writer)
		return
	} else if file.IsDir() {
		http.Error(writer, fmt.Sprintf("Requested file id %s is a directory", file.Id), 400)
		return
	}

	offsetString := paramFromRequest("offset", req)
	if offset, err := strconv.ParseInt(offsetString, 10, 64); err != nil {
		http.Error(writer, fmt.Sprintf("Invalid offset parameter: %s", offsetString), 400)
	} else {
		block := file.BlockWithOffset(offset)

		if block == "" {
			http.Error(writer, fmt.Sprintf("Block with id: %s does not belong to file with id: %s", block, file.Id), 404)
		} else {
			http.Redirect(writer, req, "/v1/block/"+string(block), http.StatusFound)
		}
	}
}

func paramFromRequest(key string, req *http.Request) string {
	vars := mux.Vars(req)
	return vars[key]
}

func writeFileNotFoundError(file *fs.OFile, writer http.ResponseWriter) {
	writer.WriteHeader(400)
	writer.Write([]byte(fmt.Sprintf("File with id: %s does not exist", file.Id)))
}
