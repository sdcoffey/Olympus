package api

import (
	"testing"

	"github.com/sdcoffey/olympus/Godeps/_workspace/src/github.com/stretchr/testify/assert"
)

func TestBuild(t *testing.T) {
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
