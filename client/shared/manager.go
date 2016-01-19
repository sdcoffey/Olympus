package shared

import (
	"github.com/google/cayley"
	"github.com/sdcoffey/olympus/client/apiclient"
	"github.com/sdcoffey/olympus/fs"
)

type OManager struct {
	Model  *OModel
	api    apiclient.ApiClient
	handle *cayley.Handle
}

func NewManager(client apiclient.ApiClient, dbConnection *cayley.Handle) *OManager {
	manager := new(OManager)
	manager.api = client
	manager.handle = dbConnection
	manager.Model = newModel(client, fs.RootNodeId)
	fs.Init(manager.handle)

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
