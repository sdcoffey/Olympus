package shared

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/sdcoffey/olympus/client/apiclient"
	"github.com/sdcoffey/olympus/fs"
)

type OModel struct {
	Root *fs.OFile
	api  apiclient.OlympusClient
}

func newModel(api apiclient.OlympusClient, RootId string) *OModel {
	model := new(OModel)
	model.Root = fs.FileWithId(RootId)
	model.api = api

	return model
}

func (model *OModel) Init() error {
	if !model.Root.Exists() {
		return errors.New(fmt.Sprintf("Root with id: %s does not exist", model.Root.Id))
	}

	if err := model.Refresh(); err != nil {
		return err
	} else {
		return nil
	}
}

func (model *OModel) Refresh() error {
	fileSet := make(map[string]bool)
	for _, fileOnDisk := range model.Root.Children() {
		fileSet[fileOnDisk.Id] = true
	}

	if fileInfos, err := model.api.ListFiles(model.Root.Id); err != nil {
		return err
	} else {
		var err error
		for i := 0; i < len(fileInfos) && err == nil; i++ {
			fi := fileInfos[i]
			file := fs.FileWithFileInfo(fi)
			err = file.Save()
			if _, ok := fileSet[fi.Id]; ok {
				delete(fileSet, fi.Id)
			}
		}

		if err != nil {
			return err
		}
	}

	for id, _ := range fileSet {
		staleId := fs.FileWithId(id)
		fs.Rm(staleId)
	}

	return nil
}

func (model *OModel) FindFileByName(name string) *fs.OFile {
	if file := fs.FileWithName(model.Root.Id, name); file == nil {
		model.Refresh()
		file = fs.FileWithName(model.Root.Id, name)
		return file
	} else {
		return file
	}
}

func (model *OModel) FindFileByPath(path string) *fs.OFile {
	path = model.absPath(path)

	return nil
}

// Takes a path relative to this model, e.g. "../path/to/file, and constructs a absolute path
// from it => /data/one/path/to/file
func (model *OModel) absPath(path string) string {
	if len(path) == 0 {
		return path
	} else if string(path[0]) == "/" {
		return path
	}

	leaves := filepath.SplitList(path)
	curnode := model.Root
	path = "/"
	for _, leaf := range leaves {
		if leaf == ".." && curnode.Parent() != nil {
			curnode = curnode.Parent()
		} else {
			curnode := fs.FileWithName(curnode.Id, leaf)
			path = filepath.Join(path, curnode.Name())
		}
	}
	return path
}
