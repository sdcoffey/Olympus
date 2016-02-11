package shared

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
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

func (manager *Manager) UploadFile(parentId, localPath string) (*graph.Node, error) {
	if fi, err := os.Stat(localPath); err != nil {
		return nil, err
	} else if fi.IsDir() {
		return nil, errors.New("Cannot upload a directory")
	} else {
		nodeInfo := graph.NodeInfo{
			Name:     filepath.Base(fi.Name()),
			Size:     fi.Size(),
			Mode:     0700,
			MTime:    time.Now(),
			ParentId: parentId,
		}
		if newNode, err := manager.api.CreateNode(nodeInfo); err != nil {
			return nil, err
		} else if localFile, err := os.Open(localPath); err != nil {
			return nil, err
		} else {
			defer localFile.Close()
			var i int64
			for i = 0; i < fi.Size(); i += graph.BLOCK_SIZE {
				buf := make([]byte, min(fi.Size()-i, graph.BLOCK_SIZE))
				if _, err = localFile.ReadAt(buf, i); err != nil {
					return nil, err
				}
				rd := bytes.NewBuffer(buf)
				if err := manager.api.SendBlock(newNode.Id, i, rd); err != nil {
					return nil, err
				}
			}
			return manager.graph.NodeWithNodeInfo(newNode), err
		}
	}
}

func min(a, b int64) int64 {
	if a < b {
		return a
	} else {
		return b
	}
}
