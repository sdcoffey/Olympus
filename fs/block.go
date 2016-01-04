package fs

import (
	"crypto"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/google/cayley"
	"github.com/sdcoffey/olympus/env"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
)

const (
	BYTE     = 1
	KILOBYTE = 1024 * BYTE
	MEGABYTE = 1024 * KILOBYTE
	GIGABYTE = 1024 * MEGABYTE
	TERABYTE = 1024 * GIGABYTE

	BLOCK_SIZE = MEGABYTE

	offsetLink = "hasOffset"
)

type OFileBlock struct {
	Hash   string
	offset int64
}

func BlockWithHash(hash string) *OFileBlock {
	return &OFileBlock{Hash: hash, offset: -1}
}

func (ofp *OFileBlock) Read() ([]byte, error) {
	filePath := filepath.Join(env.EnvPath(env.DataPath), ofp.Hash)
	if !env.Exists(filePath) {
		return make([]byte, 0), os.ErrNotExist
	}

	return ioutil.ReadFile(filePath)
}

func (ofb *OFileBlock) Save() (err error) {
	if ofb.offset%BLOCK_SIZE != 0 {
		return errors.New(fmt.Sprint("Block has invalid offset ", ofb.offset))
	} else if ofb.offset < 0 {
		return errors.New("Cannot add block without offset")
	}

	staleQuads := cayley.NewTransaction()
	newQuads := cayley.NewTransaction()

	if ofb.Offset() > 0 && ofb.offset != ofb.Offset() {
		staleQuads.RemoveQuad(cayley.Quad(ofb.Hash, offsetLink, fmt.Sprint(ofb.Offset()), ""))
	}
	newQuads.AddQuad(cayley.Quad(ofb.Hash, offsetLink, fmt.Sprint(ofb.offset), ""))

	if err = GlobalFs().Graph.ApplyTransaction(staleQuads); err != nil {
		return
	} else if err = GlobalFs().Graph.ApplyTransaction(newQuads); err != nil {
		return
	}

	return nil
}

// Assume that we're writing one block.
// Clients will be responsible for parting files, but we'll validate the hash here
func (ofb *OFileBlock) Write(bytes []byte) (n int, err error) {
	if hash := blockHash(bytes); hash != ofb.Hash {
		return 0, errors.New(fmt.Sprint("Block with hash ", hash, " does not match this block's hash"))
	} else if len(bytes) > BLOCK_SIZE {
		return 0, errors.New(fmt.Sprint("Block exceeds max block size by ", (len(bytes) - BLOCK_SIZE)))
	}

	filename := filepath.Join(env.EnvPath(env.DataPath), ofb.Hash)
	if err = ioutil.WriteFile(filename, bytes, 700); err != nil {
		return
	} else {
		n = len(bytes)
	}

	return
}

func (ofb *OFileBlock) Offset() int64 {
	it := cayley.StartPath(GlobalFs().Graph, ofb.Hash).Out(offsetLink).BuildIterator()
	if cayley.RawNext(it) {
		if val, err := strconv.ParseInt(GlobalFs().Graph.NameOf(it.Result()), 10, 64); err == nil {
			return val
		}
	}

	return -1
}

func (ofb *OFileBlock) IsOnDisk() bool {
	filename := filepath.Join(env.EnvPath(env.DataPath), ofb.Hash)
	return env.Exists(filename)
}

func blockHash(data []byte) string {
	sha := crypto.SHA1.New()
	sha.Write(data)
	return hex.EncodeToString(sha.Sum(nil))
}
