package shared

import (
	"os"
	"time"

	"github.com/google/cayley"
	"github.com/sdcoffey/olympus/client/apiclient"
	"github.com/sdcoffey/olympus/graph"
)

type Manager struct {
	api   apiclient.OlympusClient
	graph *graph.NodeGraph
}

func NewManager(client apiclient.OlympusClient, handle *cayley.Handle) *Manager {
	manager := new(Manager)
	if ng, err := graph.NewGraph(handle); err != nil {
		panic(err)
	} else {
		manager.graph = ng
	}
	manager.api = client
	return manager
}

func (manager *Manager) Model(nodeId string) (*Model, error) {
	model := newModel(manager.api, manager.graph.NodeWithId(nodeId), manager.graph)
	if err := model.init(); err != nil {
		return nil, err
	}
	return model, nil
}

func (manager *Manager) CreateDirectory(parentId string, name string) error {
	info := graph.NodeInfo{
		ParentId: parentId,
		Name:     name,
		MTime:    time.Now(),
		Mode:     uint32(os.ModeDir) | 700,
	}
	_, err := manager.api.CreateNode(info)
	return err
}

func (manager *Manager) RemoveNode(nodeId string) error {
	if err := manager.api.RemoveNode(nodeId); err != nil {
		return err
	}

	return nil
}

func (manager *Manager) MoveNode(nodeId, newParentId, newName string) error {
	if err := manager.api.MoveNode(nodeId, newParentId, newName); err != nil {
		return err
	}

	return nil
}
