package apiclient

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/sdcoffey/olympus/graph"
	"github.com/sdcoffey/olympus/server/api"
)

type OlympusClient interface {
	ListNodes(parentId string) ([]graph.NodeInfo, error)
	ListBlocks(nodeId string) ([]graph.BlockInfo, error)
	WriteBlock(nodeId string, offset int64, hash string, data io.Reader) error
	RemoveNode(nodeId string) error
	CreateNode(info graph.NodeInfo) (graph.NodeInfo, error)
	UpdateNode(info graph.NodeInfo) error
	ReadBlock(nodeId string, offset int64) (io.Reader, error)
}

type ApiClient struct {
	Address  string
	Encoding api.Encoding
}

func (client ApiClient) ListNodes(parentId string) ([]graph.NodeInfo, error) {
	if request, err := client.request(api.ListNodes, parentId); err != nil {
		return make([]graph.NodeInfo, 0), err
	} else {
		var infos []graph.NodeInfo
		if err := client.do(request, nil, &infos); err != nil {
			return make([]graph.NodeInfo, 0), err
		}
		return infos, nil
	}
}

func (client ApiClient) ListBlocks(nodeId string) ([]graph.BlockInfo, error) {
	if request, err := client.request(api.ListBlocks, nodeId); err != nil {
		return make([]graph.BlockInfo, 0), err
	} else {
		var infos []graph.BlockInfo
		if err := client.do(request, nil, &infos); err != nil {
			return make([]graph.BlockInfo, 0), err
		}
		return infos, nil
	}
}

func (client ApiClient) WriteBlock(nodeId string, offset int64, hash string, data io.Reader) error {
	if request, err := client.request(api.WriteBlock, nodeId, offset); err != nil {
		return err
	} else {
		request.Header.Add("Content-Hash", hash)
		if err := client.do(request, data, nil); err != nil {
			return err
		}
	}

	return nil
}

func (client ApiClient) RemoveNode(nodeId string) error {
	if request, err := client.request(api.RemoveNode, nodeId); err != nil {
		return err
	} else {
		return client.do(request, nil, nil)
	}
}

func (client ApiClient) CreateNode(info graph.NodeInfo) (graph.NodeInfo, error) {
	if request, err := client.request(api.CreateNode, info.ParentId); err != nil {
		return graph.NodeInfo{}, err
	} else {
		var returnInfo graph.NodeInfo
		if err := client.do(request, info, &returnInfo); err != nil {
			return graph.NodeInfo{}, err
		}

		return returnInfo, nil
	}
}

func (client ApiClient) UpdateNode(nodeInfo graph.NodeInfo) error {
	if request, err := client.request(api.UpdateNode, nodeInfo.Id); err != nil {
		return err
	} else {
		if err := client.do(request, nodeInfo, nil); err != nil {
			return err
		}

		return nil
	}
}

func (client ApiClient) ReadBlock(nodeId string, offset int64) (io.Reader, error) {
	var resp []byte
	if request, err := client.request(api.ReadBlock, nodeId, offset); err != nil {
		return nil, err
	} else if err := client.do(request, nil, &resp); err != nil {
		return nil, err
	} else {
		return bytes.NewBuffer(resp), nil
	}
}

func (client ApiClient) request(endpoint api.Endpoint, args ...interface{}) (*http.Request, error) {
	path := endpoint.Build(args...).String()
	if req, err := http.NewRequest(endpoint.Verb, client.url(path), nil); err != nil {
		return nil, err
	} else {
		req.Header.Add("Accept", string(client.Encoding))
		if endpoint.Verb == "POST" || endpoint.Verb == "PATCH" || endpoint.Verb == "PUT" {
			req.Header.Add("Content-Type", string(client.Encoding))
		}
		return req, nil
	}
}

func (client ApiClient) do(req *http.Request, body interface{}, responseBody interface{}) error {
	var buf *bytes.Buffer
	if body != nil {
		switch body.(type) {
		case io.Reader:
			req.Body = ioutil.NopCloser(body.(io.Reader))
		default:
			buf = new(bytes.Buffer)
			encoder := client.encoder(buf)
			if err := encoder.Encode(body); err != nil {
				return err
			}
			req.Body = ioutil.NopCloser(buf)
		}
	}

	if resp, err := http.DefaultClient.Do(req); err != nil {
		return err
	} else if responseBody != nil {
		var response = api.ApiResponse{
			Data: responseBody,
		}

		defer resp.Body.Close()
		decoder := client.decoder(resp.Body)
		if err = decoder.Decode(&response); err != nil {
			return fmt.Errorf("Error unmarsharaling response => %s", err)
		} else if response.Error != nil {
			return response.Error
		} else {
			return nil
		}
	} else {
		return nil
	}
}

func (client ApiClient) url(path string) string {
	return fmt.Sprint(client.Address, "/v1", path)
}

func (client ApiClient) encoder(wr io.Writer) api.Encoder {
	switch client.Encoding {
	case api.GobEncoding:
		return gob.NewEncoder(wr)
	case api.XmlEncoding:
		return xml.NewEncoder(wr)
	default:
		return json.NewEncoder(wr)
	}
}

func (client ApiClient) decoder(rd io.Reader) api.Decoder {
	switch client.Encoding {
	case api.GobEncoding:
		return gob.NewDecoder(rd)
	case api.XmlEncoding:
		return xml.NewDecoder(rd)
	default:
		return json.NewDecoder(rd)
	}
}
