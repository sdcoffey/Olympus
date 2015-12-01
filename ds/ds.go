package ds

import (
	"crypto"
	_ "crypto/sha256"
	"encoding/hex"
	"github.com/google/cayley"
	"github.com/sdcoffey/olympus/fs"
	"strconv"
)

const (
	partLink   = "hasPart"
	offsetLint = "hasOffset"
)

func FileParts(of *fs.OFile) []OFileBlock {
	if of.IsDir() {
		return make([]OFileBlock, 0)
	}

	it := cayley.StartPath(fs.GlobalFs().Graph, of.Id).In(partLink).BuildIterator()
	parts := make([]OFileBlock, 0, 10)
	for cayley.RawNext(it) {
		part := OFileBlock{fs.GlobalFs().Graph.NameOf(it.Result()), 0}
		parts = append(parts, part)
	}

	for idx, part := range parts {
		it := cayley.StartPath(fs.GlobalFs().Graph, part.Hash).Out(offsetLint).BuildIterator()
		if cayley.RawNext(it) {
			if offset, err := strconv.ParseInt(fs.GlobalFs().Graph.NameOf(it.Result), 10, 64); err == nil {
				parts[idx] = OFileBlock{part.Hash, offset}
			} else if err != nil {
				return make([]OFileBlock, 0)
			}
		}
	}

	return parts
}

func WriteFilePart(of *fs.OFile, data []byte, offset int64) {

}

func blockHash(data []byte) string {
	sha := crypto.SHA1.New()
	sha.Write(data)
	return hex.EncodeToString(sha.Sum(nil))
}
