package fs

import (
	"errors"
	"github.com/google/cayley"
	"github.com/google/cayley/graph"
	"github.com/sdcoffey/olympus/fs/model"
)

type FsWriteable interface {
	Transaction() *graph.Transaction
}

type FsReadable interface {
	Fields() []string
	SetProp(link, value string) error
	Iterator(graph *cayley.Handle) graph.Iterator
}

type Fs struct {
	Graph *cayley.Handle
}

func NewFs(graph *cayley.Handle) *Fs {
	return &Fs{graph}
}

func (fs *Fs) AddFile(of *model.OFile) error {
	if of.ParentId != "" {
		parent := model.NewFileWithId(of.ParentId)
		fs.Stat(parent)
		if !isDir(parent) {
			return errors.New("Cannot add file as a child of a non-directory")
		}
	}

	fs.Graph.ApplyTransaction(of.Transaction())
	return nil
}

func (fs *Fs) Stat(readable FsReadable) (err error) {
	it := readable.Iterator(fs.Graph)
	defer it.Close()

	for i := 0; err == nil && cayley.RawNext(it); i++ {
		readable.SetProp(readable.Fields()[i], fs.Graph.NameOf(it.Result()))
	}

	return
}

func (fs *Fs) ListFiles(parentId string) (files []*model.OFile, err error) {
	it := cayley.StartPath(fs.Graph, parentId).In("hasParent").BuildIterator()
	files = make([]*model.OFile, 0, 10)

	for i := 0; err == nil && cayley.RawNext(it); i++ {
		of := model.NewFileWithId(fs.Graph.NameOf(it.Result()))
		err = fs.Stat(of)
		files = append(files, of)
	}
	return
}

func isDir(of *model.OFile) bool {
	return of.Attr&int64(model.AttrDir) > 0
}
