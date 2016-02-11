package apiclient

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/sdcoffey/olympus/graph"
)

type OlympusClient interface {
	ListNodes(parentId string) ([]graph.NodeInfo, error)
	MoveNode(nodeid, newParentId, newName string) error
	RemoveNode(nodeId string) error
	CreateNode(info graph.NodeInfo) (graph.NodeInfo, error)
	UpdateNode(info graph.NodeInfo) error
	HasBlocks(nodeId string, blocks []string) ([]string, error)
	SendBlock(nodeId string, offset int64, data io.Reader) error
}

type ApiClient struct {
	Address string
}

func (client ApiClient) url(method string) string {
	return fmt.Sprint(client.Address, "/v1/", method)
}

func (client ApiClient) ListNodes(parentId string) ([]graph.NodeInfo, error) {
	url := client.url(fmt.Sprintf("ls/%s", parentId))
	if request, err := http.NewRequest("GET", url, nil); err != nil {
		return make([]graph.NodeInfo, 0), err
	} else {
		request.Header.Add("Accept", "application/gob")
		if resp, err := http.DefaultClient.Do(request); err != nil {
			return make([]graph.NodeInfo, 0), err
		} else {
			defer resp.Body.Close()
			var infos []graph.NodeInfo
			decoder := gob.NewDecoder(resp.Body)
			err = decoder.Decode(&infos)

			return infos, err
		}
	}
}

func (client ApiClient) MoveNode(nodeId, newParentId, newName string) error {
	url := client.url(fmt.Sprintf("mv/%s/%s", nodeId, newParentId))
	if request, err := http.NewRequest("PATCH", url, nil); err != nil {
		return err
	} else {
		if newName != "" {
			request.URL.Query().Add("rename", newName)
		}
		_, err := http.DefaultClient.Do(request)
		return err
	}
}

func (client ApiClient) RemoveNode(nodeId string) error {
	url := client.url(fmt.Sprintf("rm/%s", nodeId))
	if request, err := http.NewRequest("DELETE", url, nil); err != nil {
		return err
	} else {
		_, err := http.DefaultClient.Do(request)
		return err
	}
}

func (client ApiClient) CreateNode(info graph.NodeInfo) (graph.NodeInfo, error) {
	url := client.url(fmt.Sprintf("cr/%s", info.ParentId))
	body := new(bytes.Buffer)
	encoder := gob.NewEncoder(body)
	if err := encoder.Encode(info); err != nil {
		return graph.NodeInfo{}, err
	} else if request, err := http.NewRequest("POST", url, body); err != nil {
		return graph.NodeInfo{}, err
	} else {
		request.Header.Add("Accept", "application/gob")
		request.Header.Add("Content-Type", "application/gob")
		if resp, err := http.DefaultClient.Do(request); err != nil {
			return graph.NodeInfo{}, err
		} else {
			defer resp.Body.Close()
			var fi graph.NodeInfo
			decoder := gob.NewDecoder(resp.Body)
			if err = decoder.Decode(&fi); err != nil {
				return graph.NodeInfo{}, err
			} else {
				return fi, nil
			}
		}
	}
}

func (client ApiClient) UpdateNode(nodeInfo graph.NodeInfo) error {
	url := client.url(fmt.Sprintf("update/%s", nodeInfo.Id))
	body := new(bytes.Buffer)
	encoder := gob.NewEncoder(body)
	if err := encoder.Encode(nodeInfo); err != nil {
		return err
	} else if request, err := http.NewRequest("PATCH", url, body); err != nil {
		return err
	} else {
		request.Header.Add("Content-Type", "application/gob")
		_, err := http.DefaultClient.Do(request)
		return err
	}
}

func (client ApiClient) HasBlocks(nodeId string, blocks []string) ([]string, error) {
	url := client.url(fmt.Sprintf("hasBlocks/%s", nodeId))
	hashList := strings.Join(blocks, ",")

	if request, err := http.NewRequest("GET", url, nil); err != nil {
		return []string{}, err
	} else {
		request.Header.Add("Accept", "application/gob")
		request.URL.Query().Add("blocks", hashList)

		if resp, err := http.DefaultClient.Do(request); err != nil {
			return []string{}, err
		} else {
			defer resp.Body.Close()
			var neededHashes []string
			decoder := gob.NewDecoder(resp.Body)
			if err = decoder.Decode(&neededHashes); err != nil {
				return neededHashes, err
			} else {
				return neededHashes, nil
			}
		}
	}
}

func (client ApiClient) SendBlock(nodeId string, offset int64, data io.Reader) error {
	url := client.url(fmt.Sprintf("dd/%s/%d", nodeId, offset))

	if request, err := http.NewRequest("POST", url, data); err != nil {
		return err
	} else {
		_, err := http.DefaultClient.Do(request)
		return err
	}
}
