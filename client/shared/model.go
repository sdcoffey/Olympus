package shared

import (
	"errors"
	"fmt"
	"github.com/google/cayley"
	"github.com/sdcoffey/olympus/client/apiclient"
	"github.com/sdcoffey/olympus/fs"
)

type Model struct {
	root     *fs.OFile
	client   apiclient.ApiClient
	children []*fs.OFile
}

func NewModel(client apiclient.ApiClient, db *cayley.Handle) *Model {
	model := new(Model)
	model.client = client
	fs.Init(db)

	return model
}

func (model *Model) Init() error {
	if root, err := fs.RootNode(); err != nil {
		return err
	} else {
		model.root = root
	}

	if err := model.Refresh(); err != nil {
		return err
	} else {
		return nil
	}
}

func (model *Model) MoveToNode(id string) error {
	var file *fs.OFile
	file = fs.FileWithId(id)

	if !file.Exists() {
		return errors.New(fmt.Sprintf("File with id: %s not found", id))
	} else if !file.IsDir() {
		return errors.New(fmt.Sprint("Cannot move to non-directory"))
	} else {
		model.root = file
		if err := model.Refresh(); err != nil {
			return err
		} else {
			model.children = file.Children()
			return nil
		}
	}
}

func (model *Model) Root() *fs.OFile {
	return model.root
}

func (model *Model) Refresh() error {
	fileSet := make(map[string]string)
	for _, fod := range model.children {
		fileSet[fod.Id] = ""
	}

	if fileInfos, err := model.client.Ls(model.root.Id); err != nil {
		return err
	} else {
		model.children = make([]*fs.OFile, len(fileInfos))
		for idx, fi := range fileInfos {
			file := fs.FileWithFileInfo(fi)
			file.Save()
			if _, ok := fileSet[fi.Id]; ok {
				delete(fileSet, fi.Id)
			}

			model.children[idx] = file
		}
	}

	for id, _ := range fileSet {
		file := fs.FileWithId(id)
		fs.Rm(file)
	}

	return nil
}

func (model *Model) Count() int {
	return len(model.root.Children())
}

func (model *Model) At(index int) *fs.OFile {
	if index < 0 || index >= len(model.children) {
		return nil
	} else {
		return model.children[index]
	}
}

func (model *Model) FindFileByName(name string) *fs.OFile {
	if name == ".." {
		return model.root.Parent()
	} else if file := fs.FileWithName(model.root.Id, name); file == nil {
		model.Refresh()
		file = fs.FileWithName(model.root.Id, name)
		return file
	} else {
		return file
	}
}

func (model *Model) CreateDirectory(name string) error {
	if _, err := model.client.Mkdir(model.root.Id, name); err != nil {
		return err
	} else if err := model.Refresh(); err != nil {
		return err
	} else {
		return nil
	}
}
