package fs

import (
	"code.google.com/p/go-uuid/uuid"
	"fmt"
	"github.com/google/cayley"
	"github.com/google/cayley/graph"
	"github.com/sdcoffey/olympus/fs/model"
	"reflect"
)

type ReadWriter interface {
	SetProp(link, value string) error
	Iterator(graph *cayley.Handle) graph.Iterator
	Transaction() *graph.Transaction
}

type Fs struct {
	Graph *cayley.Handle
}

func NewFs(graph *cayley.Handle) *Fs {
	return &Fs{graph}
}

func (fs *Fs) AddFile(file *model.OFile) error {
	file.Id = uuid.NewUUID().String()
	return fs.Graph.ApplyTransaction(file.Transaction())
}

func (fs *Fs) StatFile(id string, rw ReadWriter) error {
	it := rw.Iterator(fs.Graph)
	defer it.Close()
	var err error
	for err == nil && cayley.RawNext(it) {
		fmt.Println(it.Result())
		err = rw.SetProp("idk", fs.Graph.NameOf(it.Result()))
	}

	return err
}

func (fs *Fs) ListFiles(parentId string, children []ReadWriter) (err error) {
	it := cayley.StartPath(fs.Graph, parentId).In("hasParent").BuildIterator()

	readWriter := children[0]
	for i := 0; err == nil && cayley.RawNext(it); i++ {
		if len(children) == cap(children) {
			rw := reflect.New(reflect.TypeOf(readWriter)).Interface().(ReadWriter)
			err = fs.StatFile(fs.Graph.NameOf(it.Result()), rw)
			children = append(children, rw)
		} else {
			err = fs.StatFile(fs.Graph.NameOf(it.Result()), children[i])
		}
	}
	return
}
