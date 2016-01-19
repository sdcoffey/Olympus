package shared

import (
	"errors"
	"fmt"
	"github.com/sdcoffey/olympus/client/apiclient"
	"github.com/sdcoffey/olympus/fs"
)

type OModel struct {
	Root     *fs.OFile
	api      apiclient.ApiClient
}

func newModel(api apiclient.ApiClient, RootId string) *OModel {
	model := new(OModel)
	model.Root = fs.FileWithId(RootId)
	model.api = api

	return model
}

func (model *OModel) Init() error {
	if !model.Root.Exists() {
		return errors.New(fmt.Sprintf("Root with id: %s does not exist", model.Root))
	}

	if err := model.Refresh(); err != nil {
		return err
	} else {
		return nil
	}
}

func (model *OModel) Refresh() error {
	fileSet := make(map[string]string)
	for _, fileOnDisk := range model.Root.Children() {
		fileSet[fileOnDisk.Id] = ""
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
