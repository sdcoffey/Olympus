package apiclient

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/sdcoffey/olympus/fs"
)

type OlympusClient interface {
	ListFiles(parentId string) ([]fs.FileInfo, error)
	CreateDirectory(parentId, name string) (string, error)
	MoveFile(fileid, newParentId, newName string) error
	RemoveFile(fileId string) error
	CreateFile(info fs.FileInfo) (fs.FileInfo, error)
	UpdateFile(info fs.FileInfo) error
	HasBlocks(fileId string, blocks []string) ([]string, error)
	SendBlock(fileId string, block fs.BlockInfo, data io.Reader) error
}

type ApiClient struct {
	Address string
}

func (client ApiClient) url(method string) string {
	return fmt.Sprint(client.Address, "/v1/", method)
}

func (client ApiClient) ListFiles(parentId string) ([]fs.FileInfo, error) {
	url := client.url(fmt.Sprintf("ls/%s", parentId))
	if request, err := http.NewRequest("GET", url, nil); err != nil {
		return make([]fs.FileInfo, 0), err
	} else {
		request.Header.Add("Accept", "application/gob")
		if resp, err := http.DefaultClient.Do(request); err != nil {
			return make([]fs.FileInfo, 0), err
		} else {
			defer resp.Body.Close()
			var infos []fs.FileInfo
			decoder := gob.NewDecoder(resp.Body)
			err = decoder.Decode(&infos)

			return infos, err
		}
	}
}

func (client ApiClient) CreateDirectory(parentId, name string) (string, error) {
	url := client.url(fmt.Sprintf("mkdir/%s/%s", parentId, name))
	if request, err := http.NewRequest("POST", url, nil); err != nil {
		return "", err
	} else {
		request.Header.Add("Accept", "application/gob")
		if resp, err := http.DefaultClient.Do(request); err != nil {
			return "", err
		} else {
			var id string
			defer resp.Body.Close()
			decoder := gob.NewDecoder(resp.Body)
			if decoder.Decode(&decoder); resp.StatusCode != 200 {
				return "", errors.New(string(id))
			} else {
				return string(id), nil
			}
		}
	}
}

func (client ApiClient) MoveFile(fileId, newParentId, newName string) error {
	url := client.url(fmt.Sprintf("mv/%s/%s", fileId, newParentId))
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

func (client ApiClient) RemoveFile(fileId string) error {
	url := client.url(fmt.Sprintf("rm/%s", fileId))
	if request, err := http.NewRequest("DELETE", url, nil); err != nil {
		return err
	} else {
		_, err := http.DefaultClient.Do(request)
		return err
	}
}

func (client ApiClient) CreateFile(info fs.FileInfo) (fs.FileInfo, error) {
	url := client.url(fmt.Sprintf("cr/%s/%s", info.ParentId, info.Name))
	body := new(bytes.Buffer)
	encoder := gob.NewEncoder(body)
	if err := encoder.Encode(info); err != nil {
		return fs.FileInfo{}, err
	} else if request, err := http.NewRequest("POST", url, body); err != nil {
		return fs.FileInfo{}, err
	} else {
		request.Header.Add("Accept", "application/gob")
		request.Header.Add("Content-Type", "application/gob")
		if resp, err := http.DefaultClient.Do(request); err != nil {
			return fs.FileInfo{}, err
		} else {
			defer resp.Body.Close()
			var fi fs.FileInfo
			decoder := gob.NewDecoder(resp.Body)
			if err = decoder.Decode(&fi); err != nil {
				return fs.FileInfo{}, err
			} else {
				return fi, nil
			}
		}
	}
}

func (client ApiClient) UpdateFile(fileInfo fs.FileInfo) error {
	url := client.url(fmt.Sprintf("update/%s", fileInfo.Id))
	body := new(bytes.Buffer)
	encoder := gob.NewEncoder(body)
	if err := encoder.Encode(fileInfo); err != nil {
		return err
	} else if request, err := http.NewRequest("PATCH", url, body); err != nil {
		return err
	} else {
		request.Header.Add("Content-Type", "application/gob")
		_, err := http.DefaultClient.Do(request)
		return err
	}
}

func (client ApiClient) HasBlocks(fileId string, blocks []string) ([]string, error) {
	url := client.url(fmt.Sprintf("hasBlocks/%s", fileId))
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

func (client ApiClient) SendBlock(fileId string, block fs.BlockInfo, data io.Reader) error {
	url := client.url(fmt.Sprintf("dd/%s/%s/%d", fileId, block.Hash, block.Offset))

	if request, err := http.NewRequest("POST", url, data); err != nil {
		return err
	} else {
		_, err := http.DefaultClient.Do(request)
		return err
	}
}
