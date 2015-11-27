package api

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/sdcoffey/olympus/fs"
	"net/http"
	"os"
	"time"
)

type FileInfo struct {
	Id    string
	Name  string
	Size  int64
	MTime time.Time
	Attr  int64
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

	err := fs.Mv(file, newName, newParent.Id)
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
	newFolder, err := fs.MkDir(parent.Id, name)
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
	err := json.Unmarshal(req.Body, &fileInfo)
	if err != nil {
		writer.WriteHeader(400)
		writer.Write([]byte(err.Error()))
		return
	} else if file.Exists() {
		same := true
		if fileInfo.Size != file.Size() {
			same = false
		} else if fileInfo.MTime != file.ModTime() {
			same = false
		} else if fileInfo.Attr != file.Mode() {
			same = false
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
	err := json.Unmarshal(req.Body, &fileInfo)
	if err != nil {
		writer.WriteHeader(400)
		writer.Write([]byte(err.Error()))
		return
	}

	changes := make([]func() error, 0)
	if fileInfo.Name != file.Name() {
		changes = append(changes, func() error {
			return fs.Mv(file, fileInfo.Name, file.Parent().Id)
		})
	}
	if fileInfo.Attr != file.Mode() {
		changes = append(changes, func() error {
			return fs.Chmod(file, os.FileMode(fileInfo.Attr))
		})
	}
	if !fileInfo.MTime.Equal(file.ModTime()) {
		changes = append(changes, func() error {
			fs
		})
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
