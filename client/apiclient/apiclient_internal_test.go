package apiclient

import (
	"encoding/gob"
	"encoding/json"
	"encoding/xml"
	"testing"

	"github.com/sdcoffey/olympus/Godeps/_workspace/src/github.com/stretchr/testify/assert"
	"github.com/sdcoffey/olympus/server/api"
)

func TestApiClient_TestEncoder_returnsCorrectEncoder(t *testing.T) {
	client := ApiClient{"", api.JsonEncoding}
	assert.IsType(t, &json.Encoder{}, client.encoder(nil))

	client = ApiClient{"", api.GobEncoding}
	assert.IsType(t, &gob.Encoder{}, client.encoder(nil))

	client = ApiClient{"", api.XmlEncoding}
	assert.IsType(t, &xml.Encoder{}, client.encoder(nil))
}

func TestApiClient_TestDecoder_returnsCorrectEncoder(t *testing.T) {
	client := ApiClient{"", api.JsonEncoding}
	assert.IsType(t, &json.Decoder{}, client.decoder(nil))

	client = ApiClient{"", api.GobEncoding}
	assert.IsType(t, &gob.Decoder{}, client.decoder(nil))

	client = ApiClient{"", api.XmlEncoding}
	assert.IsType(t, &xml.Decoder{}, client.decoder(nil))
}

func TestApiClient_request_setsCorrectHeaders(t *testing.T) {
	client := ApiClient{"http://localhost", api.JsonEncoding}

	request, err := client.request(api.ListNodes, "abcd")
	assert.NoError(t, err)
	assert.Equal(t, string(api.JsonEncoding), request.Header.Get("Accept"))

	request, err = client.request(api.CreateNode, "abcd")
	assert.NoError(t, err)
	assert.Equal(t, string(api.JsonEncoding), request.Header.Get("Accept"))
	assert.Equal(t, string(api.JsonEncoding), request.Header.Get("Content-Type"))

	request, err = client.request(api.UpdateNode, "abcd")
	assert.NoError(t, err)
	assert.Equal(t, string(api.JsonEncoding), request.Header.Get("Accept"))
	assert.Equal(t, string(api.JsonEncoding), request.Header.Get("Content-Type"))

	request, err = client.request(api.WriteBlock, "abcd", 0)
	assert.NoError(t, err)
	assert.Equal(t, string(api.JsonEncoding), request.Header.Get("Accept"))
	assert.Equal(t, string(api.JsonEncoding), request.Header.Get("Content-Type"))
}

func TestApiCLient_request_buildsCorrectUrl(t *testing.T) {
	address := "http://localhost"
	client := ApiClient{address, api.JsonEncoding}

	request, err := client.request(api.CreateNode, "abcd")
	assert.NoError(t, err)

	assert.Equal(t, address+"/v1/node/abcd", request.URL.String())
}

func TestApiClient_request_setsCorrectVerb(t *testing.T) {
	client := ApiClient{"http://localhost", api.JsonEncoding}

	request, err := client.request(api.ListNodes, "abcd")
	assert.NoError(t, err)
	assert.Equal(t, "GET", request.Method)

	request, err = client.request(api.WriteBlock, "abcd", 0)
	assert.NoError(t, err)
	assert.Equal(t, "PUT", request.Method)

	request, err = client.request(api.RemoveNode, "abcd")
	assert.NoError(t, err)
	assert.Equal(t, "DELETE", request.Method)

	request, err = client.request(api.CreateNode, "abcd")
	assert.NoError(t, err)
	assert.Equal(t, "POST", request.Method)

	request, err = client.request(api.UpdateNode, "abcd")
	assert.NoError(t, err)
	assert.Equal(t, "PATCH", request.Method)
}
