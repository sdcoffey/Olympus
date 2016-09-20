package graph

import (
	"bytes"
	"crypto"
	"encoding/hex"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/sdcoffey/olympus/env"
)

const (
	BYTE = 1 << (iota * 10)
	KILOBYTE
	MEGABYTE
	GIGABTYE
	TERABYTE

	BLOCK_SIZE = MEGABYTE
)

func Hash(d []byte) string {
	sha := crypto.SHA1.New()
	sha.Write(d)
	return hex.EncodeToString(sha.Sum(nil))
}

func Reader(hash string) (io.Reader, error) {
	return backingFile(hash)
}

func RawData(hash string) ([]byte, error) {
	if file, err := backingFile(hash); err != nil {
		return []byte{}, err
	} else if p, err := ioutil.ReadAll(file); err != nil {
		return []byte{}, err
	} else {
		return p, nil
	}
}

func Write(hash string, d []byte) (int, error) {
	dataHash := Hash(d)
	if len(d) > BLOCK_SIZE {
		return 0, errors.New("Data length exceeds max block size")
	} else if dataHash != hash {
		return 0, errors.New("Data hash does not match this block's hash")
	}

	if file, err := os.OpenFile(LocationOnDisk(hash), os.O_CREATE|os.O_EXCL|os.O_RDWR, os.FileMode(0644)); err != nil {
		if os.IsExist(err) { // If we've already written this data, short circuit
			return len(d), nil
		} else {
			return 0, err
		}
	} else {
		buf := bytes.NewBuffer([]byte(d))
		n, err := io.Copy(file, buf)
		return int(n), err
	}
}

func SizeOnDisk(hash string) (int64, error) {
	if fi, err := os.Stat(LocationOnDisk(hash)); err != nil {
		return 0, err
	} else {
		return fi.Size(), nil
	}
}

func LocationOnDisk(hash string) string {
	return filepath.Join(env.EnvPath(env.DataPath), hash)
}

func backingFile(hash string) (*os.File, error) {
	if backingFile, err := os.Open(LocationOnDisk(hash)); err != nil {
		return nil, err
	} else {
		return backingFile, nil
	}
}
