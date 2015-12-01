package ds

import (
	"github.com/sdcoffey/olympus/env"
	"io/ioutil"
	"os"
	"path/filepath"
)

type OFileBlock struct {
	Hash   string
	offset int64
}

func (ofp *OFileBlock) Read() ([]byte, error) {
	filePath := filepath.Join(env.EnvPath(env.DataPath), ofp.Hash)
	if !env.Exists(filePath) {
		return make([]byte, 0), os.ErrNotExist
	}

	return ioutil.ReadFile(filePath)
}

// todo
func (ofp *OFileBlock) Write(bytes []byte) error {
	return nil
}
