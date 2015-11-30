package api

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/sdcoffey/olympus/fs"
	"net/http"
	"time"
)

type Child struct {
	Id    string
	Name  string
	Size  int64
	MTime time.Time
	Attr  int64
}

type LsResponse struct {
	Name     string
	ParentId string
	Children []Child
}

// /ls/{parentId}
func LsFiles(writer http.ResponseWriter, req *http.Request) {
	file := fs.FileWithId(paramFromRequest("parentId", req))
	if !file.Exists() {
		writeFileNotFoundError(file, writer)
		return
	}

	response := LsResponse{Name: file.Name(), ParentId: file.Id}
	children := file.Children()
	responseChildren := make([]Child, len(children))

	for idx, child := range children {
		responseChildren[idx] = Child{
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

// /rm/{fileId}
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

// /mv/{fileId}/{newParentId}
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

	err := fs.Mv(file, file.Name(), newParent.Id)
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

	vars := mux.Vars(req)
	newName := vars["name"]
	newFolder, err := fs.MkDir(parent.Id, newName)
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

func Cr(writer http.ResponseWriter, req *http.Request) {
}

func paramFromRequest(key string, req *http.Request) string {
	vars := mux.Vars(req)
	return vars[key]
}

func writeFileNotFoundError(file *fs.OFile, writer http.ResponseWriter) {
	writer.WriteHeader(400)
	writer.Write([]byte(fmt.Sprintf("File with id: %s does not exist", file.Id)))
}
