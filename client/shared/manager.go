package shared

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"fmt"

	"github.com/cayleygraph/cayley"
	"github.com/sdcoffey/olympus/client/apiclient"
	"github.com/sdcoffey/olympus/graph"
	"github.com/sdcoffey/olympus/util"
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
		Mode:     os.ModeDir | os.FileMode(0700),
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
	nodeInfo := graph.NodeInfo{
		Id:       nodeId,
		ParentId: newParentId,
		Name:     newName,
	}
	return manager.api.UpdateNode(nodeInfo)
}

type ProgressCallback func(total, current int64)

func (manager *Manager) UploadFile(parentId, localPath string, callback ProgressCallback) (*graph.Node, error) {
	errorFmt := func(err error) error {
		return fmt.Errorf("Error uploading file: %s", err.Error())
	}

	if fi, err := os.Stat(localPath); err != nil {
		return nil, errorFmt(err)
	} else if fi.IsDir() {
		return nil, errors.New("Cannot upload a directory")
	} else {
		nodeInfo := graph.NodeInfo{
			Name:     filepath.Base(fi.Name()),
			Size:     fi.Size(),
			Mode:     0700,
			MTime:    time.Now(),
			ParentId: parentId,
			Type:     util.MimeType(fi.Name()),
		}
		if newNode, err := manager.api.CreateNode(nodeInfo); err != nil {
			return nil, errorFmt(err)
		} else if localFile, err := os.Open(localPath); err != nil {
			return nil, errorFmt(err)
		} else {
			defer localFile.Close()

			errChan := make(chan error)
			uploadChan := make(chan heap, 5)
			defer close(uploadChan)

			var wg sync.WaitGroup
			numBlocks := int(fi.Size() / graph.BLOCK_SIZE)
			if fi.Size()%graph.BLOCK_SIZE > 0 {
				numBlocks++
			}

			wg.Add(numBlocks)
			var uploadedBytes int64
			for i := 0; i < 5; i++ { // TODO: min(numblocks, 5)
				go func() {
					for h := range uploadChan {
						payloadSize := int64(len(h.data))

						rd := bytes.NewBuffer(h.data)
						hash := graph.Hash(h.data)
						if err = manager.api.WriteBlock(newNode.Id, h.offset, hash, rd); err != nil {
							errChan <- err
						}
						uploadedBytes += payloadSize
						callback(fi.Size(), uploadedBytes)
						wg.Done()
					}
				}()
			}

			errChecker := func() error {
				select {
				case err := <-errChan:
					return errorFmt(err)
				default:
					return nil
				}
			}

			var offset int64
			for offset = 0; offset < fi.Size(); offset += graph.BLOCK_SIZE {
				buf := make([]byte, min(fi.Size()-offset, graph.BLOCK_SIZE))
				if _, err = localFile.ReadAt(buf, offset); err != nil {
					return nil, errorFmt(err)
				}
				uploadChan <- heap{offset, buf}

				if err := errChecker(); err != nil {
					return nil, errorFmt(err)
				}
			}

			if err := errChecker(); err != nil {
				return nil, errorFmt(err)
			}
			wg.Wait()

			return manager.graph.NodeWithNodeInfo(newNode), err
		}
	}
}

type heap struct {
	offset int64
	data   []byte
}

func min(a, b int64) int64 {
	if a < b {
		return a
	} else {
		return b
	}
}
