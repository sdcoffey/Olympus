package util

import (
	"mime"
	"path/filepath"
	"strings"
)

func MimeType(filename string) string {
	mimeType := mime.TypeByExtension(filepath.Ext(filename))
	if strings.Contains(mimeType, ";") {
		mimeType = strings.Split(mimeType, ";")[0]
	}

	return mimeType
}
