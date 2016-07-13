package api

import (
	"encoding/gob"
	"encoding/json"
	"encoding/xml"
	"net/http"
	"testing"

	"github.com/sdcoffey/olympus/Godeps/_workspace/src/github.com/stretchr/testify/assert"
)

func TestEncoderFromHeader_returnsCorrectEncoder(t *testing.T) {
	header := http.Header(make(map[string][]string))
	header.Set("Accept", string(GobEncoding))

	encoder := encoderFromHeader(nil, header)
	assert.IsType(t, &gob.Encoder{}, encoder)

	header.Set("Accept", string(XmlEncoding))

	encoder = encoderFromHeader(nil, header)
	assert.IsType(t, &xml.Encoder{}, encoder)
}

func TestEncoderFromHeader_returnsJsonEcoderByDefault(t *testing.T) {
	header := http.Header(make(map[string][]string))

	encoder := encoderFromHeader(nil, header)
	assert.IsType(t, &json.Encoder{}, encoder)
}

func TestDecoderFromHeader_returnsCorrectDecoder(t *testing.T) {
	header := http.Header(make(map[string][]string))
	header.Set("Content-Type", string(GobEncoding))

	decoder := decoderFromHeader(nil, header)
	assert.IsType(t, &gob.Decoder{}, decoder)

	header.Set("Content-Type", string(XmlEncoding))

	decoder = decoderFromHeader(nil, header)
	assert.IsType(t, &xml.Decoder{}, decoder)
}

func TestDecoderFromHeader_returnsJsonDecoderByDefault(t *testing.T) {
	header := http.Header(make(map[string][]string))

	decoder := decoderFromHeader(nil, header)
	assert.IsType(t, &json.Decoder{}, decoder)
}

// Endpoint tests
func TestEndpoint_Build(t *testing.T) {
	assert.Equal(t, "/node/abcd", ListNodes.Build("abcd").String())
	assert.Equal(t, "/node/abcd/block", ListBlocks.Build("abcd").String())
	assert.Equal(t, "/node/efgh", CreateNode.Build("efgh").String())
	assert.Equal(t, "/node/abcd/block/1024", WriteBlock.Build("abcd", 1024).String())
	assert.Equal(t, "/node/abcd", RemoveNode.Build("abcd").String())
	assert.Equal(t, "/node/abcd", UpdateNode.Build("abcd").String())
	assert.Equal(t, "/node/abcd/block/1024", ReadBlock.Build("abcd", 1024).String())
	assert.Equal(t, "/node/abcd/stream", DownloadNode.Build("abcd").String())
}

func TestEndpoint_Query(t *testing.T) {
	path := ListNodes.Build("abcd").Query("watermark", "1").Query("limit", "2").String()
	assert.Contains(t, path, "watermark=1")
	assert.Contains(t, path, "limit=2")
	assert.Contains(t, path, "/node/abcd?")

	path = ListNodes.Build("abcd").Query("watermark", "1").Query("limit", "2").String()
	assert.Contains(t, path, "limit=2")
	assert.Contains(t, path, "/node/abcd?")
}
