package ds
import (
	"github.com/sdcoffey/olympus/fs"
	"github.com/google/cayley"
	"strconv"
)

const (
	partLink = "hasPart"
	offsetLint = "hasOffset"
)

func FileParts(of *fs.OFile) []OFilePart {
	if of.IsDir() {
		return make([]OFilePart, 0)
	}

	it := cayley.StartPath(fs.GlobalFs().Graph, of.Id).In(partLink).BuildIterator()
	parts := make([]OFilePart, 0, 10)
	for cayley.RawNext(it) {
		part := OFilePart{fs.GlobalFs().Graph.NameOf(it.Result()), "",}
		part = append(parts, part)
	}

	for idx, part := range parts {
		it := cayley.StartPath(fs.GlobalFs().Graph, part.Fingerprint).Out(offsetLint).BuildIterator()
		if cayley.RawNext(it) {
			if offset, err := strconv.ParseInt(fs.GlobalFs().Graph.NameOf(it.Result), 10, 64); err == nil {
				parts[idx] = OFilePart{part.Fingerprint, offset}
			} else if err != nil {
				return make([]OFilePart, 0)
			}
		}
	}

	return parts
}
