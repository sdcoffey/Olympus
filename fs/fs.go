package fs

import (
	"errors"
	"fmt"
	"github.com/google/cayley"
	"os"
	"time"
)

var globalFs *Fs
var rootNode *OFile

const RootNodeId = "rootNode"

type Fs struct {
	*cayley.Handle
}

func Init(graph *cayley.Handle) error {
	globalFs = &Fs{graph}
	var err error
	rootNode, err = RootNode()

	return err
}

func GlobalFs() *Fs {
	return globalFs
}

func RootNode() (root *OFile, err error) {
	if root = FileWithId(RootNodeId); !root.Exists() {
		root = newFile("root")
		root.Id = RootNodeId
		root.mode |= os.ModeDir
		err = root.Save()
	}

	return
}

func (of *OFile) addChild(child *OFile) (err error) {
	if !of.IsDir() {
		return errors.New("Cannot add file to a non-directory")
	} else if of.Exists() && FileWithName(of.Id, child.name) != nil {
		return errors.New(fmt.Sprintf("File with name %s already exists in %s", child.name, of.Name()))
	} else if !of.Exists() {
		return errors.New(fmt.Sprint("Parent ", of.Id, " does not exist"))
	}

	child.parentId = of.Id
	return child.Save()
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

func (of *OFile) MkDir(name string) (f *OFile, err error) {
	child := newFile(name)
	child.mode |= os.ModeDir

	if err = of.addChild(child); err != nil {
		return
	}
	return child, err
}

func (of *OFile) Mv(newName, newParentId string) (err error) {
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

	newParent := FileWithId(newParentId)
	return newParent.addChild(of)
}

func (of *OFile) Chmod(newMode os.FileMode) (err error) {
	if of.IsDir() != (newMode&os.ModeDir > 0) {
		return errors.New("Cannot merge file into folder")
	}

	of.mode = newMode
	return of.Save()
}

func MkFile(name, parentId string, size int64, mTime time.Time) (of *OFile, err error) {
	of = newFile(name)
	of.size = size
	of.mTime = mTime

	parent := FileWithId(parentId)
	if err = parent.addChild(of); err != nil {
		of = nil
		return
	}

	return
}

func (of *OFile) Touch(mTime time.Time) (err error) {
	of.mTime = mTime
	return of.Save()
}

func (of *OFile) Resize(size int64) error {
	of.size = size
	return of.Save()
}
