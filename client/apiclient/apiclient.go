package apiclient

import (
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/sdcoffey/olympus/fs"
	"net/http"
)

type ApiClient struct {
	Address string
}

func (client ApiClient) Ls(parentId string) ([]fs.FileInfo, error) {
	url := fmt.Sprintf("%s/v1/ls/%s", client.Address, parentId)
	if resp, err := http.Get(url); err != nil {
		return make([]fs.FileInfo, 0), err
	} else {
		defer resp.Body.Close()
		var infos []fs.FileInfo
		decoder := gob.NewDecoder(resp.Body)
		err = decoder.Decode(&infos)

		return infos, err
	}
}

func (client ApiClient) Mkdir(parentId, name string) (string, error) {
	url := fmt.Sprintf("%s/v1/mkdir/%s/%s", client.Address, parentId, name)
	if resp, err := http.Post(url, "application/gob", nil); err != nil {
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
