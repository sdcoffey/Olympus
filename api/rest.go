package api

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/sdcoffey/olympus/fs"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type FileInfo struct {
	Id    string
	Name  string
	Size  int64
	MTime time.Time
	Attr  int64
}

type BlockInfo struct {
	Hash   string
	Offset int64
}

type LsResponse struct {
	Name     string
	ParentId string
	Children []FileInfo
}

// v1/ls/{parentId}
func LsFiles(writer http.ResponseWriter, req *http.Request) {
	file := fs.FileWithId(paramFromRequest("parentId", req))
	if !file.Exists() {
		writeFileNotFoundError(file, writer)
		return
	}

	response := LsResponse{Name: file.Name(), ParentId: file.Id}
	children := file.Children()
	responseChildren := make([]FileInfo, len(children))

	for idx, child := range children {
		responseChildren[idx] = FileInfo{
			Id:    child.Id,
			Name:  child.Name(),
			Size:  child.Size(),
			MTime: child.ModTime(),
			Attr:  int64(child.Mode()),
		}
	}
	response.Children = responseChildren
	serialized, err := json.Marshal(&response)
	if err != nil {
		writer.WriteHeader(500)
		return
	}

	writer.WriteHeader(200)
	writer.Write(serialized)
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
		writer.WriteHeader(500)
		writer.Write([]byte(err.Error()))
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
		writer.WriteHeader(500)
		writer.Write([]byte(err.Error()))
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
		writer.WriteHeader(400)
		writer.Write([]byte(err.Error()))
	} else {
		type response struct {
			Id string
		}

		bytes, err := json.Marshal(&response{newFolder.Id})
		if err != nil {
			writer.WriteHeader(400)
			writer.Write([]byte(err.Error()))
		} else {
			writer.WriteHeader(200)
			writer.Write(bytes)
		}
	}
}

// v1/cr/{parentId}/{name}
// body -> {FileInfo}
func Cr(writer http.ResponseWriter, req *http.Request) {
	parent := fs.FileWithId(paramFromRequest("parentId", req))
	if !parent.Exists() {
		writeFileNotFoundError(parent, writer)
		return
	}

	file := fs.FileWithName(parent.Id, paramFromRequest("name", req))

	var fileInfo FileInfo
	defer req.Body.Close()
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&fileInfo)

	if err != nil {
		writer.WriteHeader(400)
		writer.Write([]byte(err.Error()))
		return
	} else if file != nil && file.Exists() {
		writer.WriteHeader(400)
		writer.Write([]byte(fmt.Sprint("File exists, call /v1/touch/", file.Id, " to update this object")))
	} else {
		if newFile, err := fs.MkFile(fileInfo.Name, parent.Id, fileInfo.Size, fileInfo.MTime); err != nil {
			writer.WriteHeader(400)
			writer.Write([]byte(err.Error()))
		} else {
			writer.WriteHeader(200)
			fileInfo.Id = newFile.Id
			body, _ := json.Marshal(&fileInfo)
			writer.Write(body)
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

	var fileInfo FileInfo
	defer req.Body.Close()
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&fileInfo)

	if err != nil {
		writer.WriteHeader(400)
		writer.Write([]byte(err.Error()))
		return
	}

	changes := make([]func() error, 3)
	changes[0] = func() error {
		return file.Mv(fileInfo.Name, file.Parent().Id)
	}
	changes[1] = func() error {
		return file.Chmod(os.FileMode(fileInfo.Attr))
	}
	changes[2] = func() error {
		return file.Touch(fileInfo.MTime)
	}

	//todo :resize

	for i := 0; i < len(changes) && err == nil; i++ {
		err = changes[i]()
	}

	if err != nil {
		writer.WriteHeader(400)
		writer.Write([]byte(err.Error()))
	} else {
		writer.WriteHeader(200)
	}
}

// v1/hasBlocks/{fileId}?blocks=hash1,hash2
// body {[]string} (hashes we don't have)
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

	body, _ := json.Marshal(&neededBlocks)
	writer.WriteHeader(200)
	writer.Write(body)
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

func writeError(err error, writer http.ResponseWriter) {
	writer.WriteHeader(400)
	writer.Write([]byte(err.Error()))
}

func writeFileNotFoundError(file *fs.OFile, writer http.ResponseWriter) {
	writer.WriteHeader(400)
	writer.Write([]byte(fmt.Sprintf("File with id: %s does not exist", file.Id)))
}
