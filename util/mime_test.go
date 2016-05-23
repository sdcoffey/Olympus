package util

import (
	"github.com/sdcoffey/olympus/Godeps/_workspace/src/github.com/stretchr/testify/assert"
	"testing"
)

func TestMimeType(t *testing.T) {
	mime := MimeType("text.json")
	assert.Equal(t, "application/json", mime)
}

func TestMimeType_worksWithExtensionOnly(t *testing.T) {
	mime := MimeType(".json")
	assert.Equal(t, "application/json", mime)
}

func TestMimeType_stripsCharsetValues(t *testing.T) {
	mime := MimeType("file.txt")
	assert.Equal(t, "text/plain", mime)
}
