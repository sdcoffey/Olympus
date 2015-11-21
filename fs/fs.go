package fs

import (
	"errors"
	"github.com/google/cayley"
	"os"
)

var globalFs *Fs

type Fs struct {
	Graph *cayley.Handle
}

type GraphWriter interface {
	Write() error
	Delete() error
}

func Init(graph *cayley.Handle) {
	globalFs = &Fs{graph}
}

func GlobalFs() *Fs {
	return globalFs
}

func addChild(parentId string, child *OFile) (err error) {
	parent := FileWithId(parentId)
	if !parent.IsDir() {
		err = errors.New("Can't add file to a non-directory")
		return
	}

	child.parentId = parent.Id
	return child.Write()
}

func Rm(of *OFile) (err error) {
	children := of.Children()
	if len(children) > 0 {
		for i := 0; i < len(children) && err == nil; i++ {
			err = Rm(children[i])
		}
	}

	return of.delete()
}

func MkDir(parentId string, name string) (f *OFile, err error) {
	child := newFile(name)
	child.mode = os.ModeDir

	if err = addChild(parentId, child); err != nil {
		return
	}
	return child, err
}
