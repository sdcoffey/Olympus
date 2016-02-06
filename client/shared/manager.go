package shared

import (
	"github.com/google/cayley"
	"github.com/sdcoffey/olympus/client/apiclient"
	"github.com/sdcoffey/olympus/fs"
)

type OManager struct {
	Model  *OModel
	api    apiclient.OlympusClient
	handle *cayley.Handle
}

func NewManager(client apiclient.OlympusClient, dbConnection *cayley.Handle) *OManager {
	manager := new(OManager)
	manager.api = client
	manager.handle = dbConnection
	manager.Model = newModel(client, fs.RootNodeId)
	if err := fs.Init(manager.handle); err != nil {
		panic(err)
	}

	return manager
}

func (manager *OManager) ChangeDirectory(fileId string) error {
	model := newModel(manager.api, fileId)
	if err := model.Init(); err != nil {
		return err
	}
	manager.Model = model
	return nil
}

func (manager *OManager) Init() error {
	return manager.Model.Init()
}

func (manager *OManager) CreateDirectory(parentId string, name string) error {
	_, err := manager.api.CreateDirectory(parentId, name)
	return err
}

func (manager *OManager) RemoveFile(id string) error {
	if err := manager.api.RemoveFile(id); err != nil {
		return err
	} else if err = manager.Model.Refresh(); err != nil {
		return err
	}

	return nil
}

func (manager *OManager) MoveFile(fileId, newParentId, newName string) error {
	if err := manager.api.MoveFile(fileId, newParentId, newName); err != nil {
		return err
	} else if err = manager.Model.Refresh(); err != nil {
		return err
	}

	return nil
}
