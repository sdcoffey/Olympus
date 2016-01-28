package fs

import (
	"bytes"
	"crypto"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"github.com/sdcoffey/olympus/env"
)

const (
	BYTE     = 1
	KILOBYTE = 1024 * BYTE
	MEGABYTE = 1024 * KILOBYTE
	GIGABYTE = 1024 * MEGABYTE
	TERABYTE = 1024 * GIGABYTE

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

func Write(hash string, d []byte) (int, error) {
	dataHash := Hash(d)
	if len(d) > BLOCK_SIZE {
		return 0, errors.New("Data length exceeds max block size")
	} else if dataHash != hash {
		return 0, errors.New("Data has does not match this blocks hash")
	}

	if file, err := os.OpenFile(LocationOnDisk(hash), os.O_CREATE|os.O_TRUNC|os.O_RDWR, os.FileMode(0644)); err != nil {
		return 0, err
	} else {
		syscall.Flock(int(file.Fd()), syscall.LOCK_EX)
		defer syscall.Flock(int(file.Fd()), syscall.LOCK_UN)

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

func IsOnDisk(hash string) bool {
	return env.Exists(LocationOnDisk(hash))
}

func backingFile(hash string) (*os.File, error) {
	if backingFile, err := os.Open(LocationOnDisk(hash)); err != nil {
		return nil, err
	} else {
		return backingFile, nil
	}
}
