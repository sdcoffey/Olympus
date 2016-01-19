package api

import (
	"encoding/gob"
	"encoding/json"
	"errors"
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

// v1/ls/{parentId}
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
	writer.WriteHeader(200)
	encoder.Encode(response)
}

// v1/rm/{fileId}
func RmFile(writer http.ResponseWriter, req *http.Request) {
	file := fs.FileWithId(paramFromRequest("fileId", req))
	if !file.Exists() {
		writeFileNotFoundError(file, writer)
		return
	}

	err := fs.Rm(file)
	if err != nil {
		writeStatusErr(500, err, writer)
	} else {
		writer.WriteHeader(200)
	}
}

// /mv/{fileId}/{newParentId}?rename={newName}
func MvFile(writer http.ResponseWriter, req *http.Request) {
	file := fs.FileWithId(paramFromRequest("fileId", req))
	if !file.Exists() {
		writeFileNotFoundError(file, writer)
		return
	}

	newParent := fs.FileWithId(paramFromRequest("newParentId", req))
	if !newParent.Exists() {
		writeFileNotFoundError(file, writer)
	}

	newName := req.URL.Query().Get("rename")
	if newName == "" {
		newName = file.Name()
	}

	err := file.Mv(newName, newParent.Id)
	if err != nil {
		writeStatusErr(500, err, writer)
	} else {
		writer.WriteHeader(200)
	}
}

// v1/mkdir/{parentId}/{fileName}
func MkDir(writer http.ResponseWriter, req *http.Request) {
	parent := fs.FileWithId(paramFromRequest("parentId", req))
	if !parent.Exists() {
		writeFileNotFoundError(parent, writer)
		return
	}

	name := paramFromRequest("name", req)
	newFolder, err := parent.MkDir(name)
	if err != nil {
		writeError(err, writer)
	} else {
		encoder := encoderFromHeader(writer, req.Header)
		writer.WriteHeader(200)
		encoder.Encode(newFolder.Id)
	}
}

// v1/cr/{parentId}/{name}
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
		writeError(errors.New("File exists, call /v1/touch/"+file.Id+" to update this object"), writer)
	} else {
		if newFile, err := fs.MkFile(fileInfo.Name, parent.Id, fileInfo.Size, fileInfo.MTime); err != nil {
			writeError(err, writer)
		} else {
			encoder := encoderFromHeader(writer, req.Header)
			writer.WriteHeader(200)
			fileInfo.Id = newFile.Id
			encoder.Encode(fileInfo)
		}
	}
}

// v1/update/{fileId}
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
		writeError(err, writer)
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
		writeStatusErr(500, err, writer)
	} else {
		writer.WriteHeader(200)
	}
}

// v1/hasBlocks/{fileId}?blocks=hash1,hash2
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

// v1/dd/{fileId}/{blockHash}/{offset}
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
		writeError(err, writer)
		return
	}

	hash := paramFromRequest("blockHash", req)
	if offset, err := strconv.ParseInt(paramFromRequest("offset", req), 10, 64); err != nil {
		writeError(err, writer)
	} else {
		block := file.BlockWithOffset(offset)
		if block.Hash != hash {
			file.RemoveBlock(block)
			block = fs.BlockWithHash(hash)
		}
		if !block.IsOnDisk() {
			if _, err = block.Write(data); err != nil {
				writeError(err, writer)
			}
		}
		if err = file.AddBlock(block, offset); err != nil {
			writeError(err, writer)
		} else {
			writer.WriteHeader(200)
		}
	}
}

func paramFromRequest(key string, req *http.Request) string {
	vars := mux.Vars(req)
	return vars[key]
}

func writeStatusErr(statusCode int, err error, writer http.ResponseWriter) {
	writer.WriteHeader(statusCode)
	writer.Write([]byte(err.Error()))
}

func writeError(err error, writer http.ResponseWriter) {
	writeStatusErr(400, err, writer)
}

func writeFileNotFoundError(file *fs.OFile, writer http.ResponseWriter) {
	writer.WriteHeader(400)
	writer.Write([]byte(fmt.Sprintf("File with id: %s does not exist", file.Id)))
}
