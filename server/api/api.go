package api

import (
	"encoding/gob"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/sdcoffey/olympus/fs"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type Encoder interface {
	Encode(interface{}) error
}

type Decoder interface {
	Decode(interface{}) error
}

func Router() http.Handler {
	r := mux.NewRouter()
	v1Router := r.PathPrefix("/v1").Subrouter()
	v1Router.HandleFunc("/ls/{parentId}", LsFiles).Methods("GET")
	v1Router.HandleFunc("/mv/{fileId}/{newParentId}", MvFile).Methods("PATCH")
	v1Router.HandleFunc("/rm/{fileId}", RmFile).Methods("DELETE")
	v1Router.HandleFunc("/mkdir/{parentId}/{name}", MkDir).Methods("POST")
	v1Router.HandleFunc("/cr/{parentId}/{name}", Cr).Methods("POST")
	v1Router.HandleFunc("/update/{fileId}", Update).Methods("PATCH")
	v1Router.HandleFunc("/hasBlocks/{fileId}", HasBlocks).Methods("GET")
	v1Router.HandleFunc("/dd/{fileId}/{blockHash}/{offset}", WriteBlock).Methods("POST")

	return r
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
func LsFiles(writer http.ResponseWriter, req *http.Request) {
	file := fs.FileWithId(paramFromRequest("parentId", req))
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
func RmFile(writer http.ResponseWriter, req *http.Request) {
	file := fs.FileWithId(paramFromRequest("fileId", req))
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
func MvFile(writer http.ResponseWriter, req *http.Request) {
	file := fs.FileWithId(paramFromRequest("fileId", req))
	if !file.Exists() {
		writeFileNotFoundError(file, writer)
		return
	}

	newParent := fs.FileWithId(paramFromRequest("newParentId", req))
	if !newParent.Exists() {
		writeFileNotFoundError(newParent, writer)
		return
	}

	newName := req.URL.Query().Get("rename")
	if newName == "" {
		newName = file.Name()
	}

	err := file.Mv(newName, newParent.Id)
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

// POST v1/cr/{parentId}/{name}
// body -> {FileInfo}
// returns -> {FileInfo}
func Cr(writer http.ResponseWriter, req *http.Request) {
	parent := fs.FileWithId(paramFromRequest("parentId", req))
	if !parent.Exists() {
		writeFileNotFoundError(parent, writer)
		return
	}

	file := fs.FileWithName(parent.Id, paramFromRequest("name", req))

	var fileInfo fs.FileInfo
	defer req.Body.Close()
	decoder := decoderFromHeader(req.Body, req.Header)

	if err := decoder.Decode(&fileInfo); err != nil {
		writer.WriteHeader(400)
		writer.Write([]byte(err.Error()))
		return
	} else if file != nil && file.Exists() {
		http.Error(writer, fmt.Sprint("File exists, call /v1/touch/%s/to update this object", file.Id), 400)
	} else {
		if newFile, err := fs.MkFile(fileInfo.Name, parent.Id, fileInfo.Size, fileInfo.MTime); err != nil {
			http.Error(writer, err.Error(), 400)
		} else {
			encoder := encoderFromHeader(writer, req.Header)
			writer.WriteHeader(200)
			fileInfo.Id = newFile.Id
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

// GET v1/hasBlocks/{fileId}?blocks=hash1,hash2
// returns -> {[]string} (hashes we don't have)
func HasBlocks(writer http.ResponseWriter, req *http.Request) {
	file := fs.FileWithId(paramFromRequest("fileId", req))
	if !file.Exists() {
		writeFileNotFoundError(file, writer)
		return
	}

	questionableBlocks := req.URL.Query().Get("blocks")
	questionableBlockList := strings.Split(questionableBlocks, ",")

	neededBlocks := make([]string, 0, len(questionableBlockList))
	for _, questionableBlock := range questionableBlockList {
		block := fs.BlockWithHash(questionableBlock)
		if !block.IsOnDisk() {
			neededBlocks = append(neededBlocks, block.Hash)
		}
	}

	writer.WriteHeader(200)
	encoder := encoderFromHeader(writer, req.Header)
	encoder.Encode(&neededBlocks)
}

// POST v1/dd/{fileId}/{blockHash}/{offset}
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

	hash := paramFromRequest("blockHash", req)
	if offset, err := strconv.ParseInt(paramFromRequest("offset", req), 10, 64); err != nil {
		http.Error(writer, err.Error(), 400)
	} else {
		block := file.BlockWithOffset(offset)
		if block.Hash != hash {
			file.RemoveBlock(block)
			block = fs.BlockWithHash(hash)
		}
		if !block.IsOnDisk() {
			if _, err = block.Write(data); err != nil {
				http.Error(writer, err.Error(), 400)
			}
		}
		if err = file.AddBlock(block, offset); err != nil {
			http.Error(writer, err.Error(), 400)
		} else {
			writer.WriteHeader(200)
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
