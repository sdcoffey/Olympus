package fs

import (
	"errors"
	"fmt"
	"github.com/google/cayley"
	"os"
)

var globalFs *Fs
var rootNode *OFile

type Fs struct {
	Graph *cayley.Handle
}

type GraphWriter interface {
	Write() error
	Delete() error
}

func Init(graph *cayley.Handle) {
	globalFs = &Fs{graph}
	var err error
	rootNode, err = RootNode()
	if err != nil {
		panic(err)
	}
}

func GlobalFs() *Fs {
	return globalFs
}

func RootNode() (root *OFile, err error) {
	if root = FileWithId("rootNode"); !root.Exists() {
		root = newFile("root")
		root.Id = "rootNode"
		root.mode |= os.ModeDir
		err = root.Write()
	}

	return
}

func addChild(parentId string, child *OFile) (err error) {
	parent := FileWithId(parentId)
	if !parent.IsDir() {
		return errors.New("Cannot add file to a non-directory")
	} else if FileWithId(parentId).Exists() && FileWithName(parentId, child.name) != nil {
		return errors.New(fmt.Sprintf("File with name %s already exists in %s", child.name, FileWithId(parentId).Name()))
	} else if !FileWithId(parentId).Exists() {
		return errors.New(fmt.Sprint("Parent ", parentId, " does not exist"))
	}

	child.parentId = parent.Id
	return child.Write()
}

func Rm(of *OFile) (err error) {
	if of.Id == rootNode.Id && of.Parent() == nil {
		return errors.New("Cannot delete root node")
	}

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
	child.mode |= os.ModeDir

	if err = addChild(parentId, child); err != nil {
		return
	}
	return child, err
}

func Mv(of *OFile, newName, newParentId string) (err error) {
	if of.Parent() == nil {
		return errors.New("Cannot move root node")
	} else if newParentId == of.Id {
		return errors.New("Cannot move file inside itself")
	} else if newParentId == of.Parent().Id && newName == of.Name() {
		return nil
	}

	if of.Name() != newName {
		of.name = newName
	}

	return addChild(newParentId, of)
}

func Chmod(of *OFile, newMode os.FileMode) (err error) {
	if of.IsDir() != (newMode&os.ModeDir > 0) {
		return errors.New("Cannot merge file into folder")
	}

	of.mode = newMode
	return of.Write()
}
