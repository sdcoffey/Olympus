package fs

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/google/cayley"
)

const RootNodeId = "rootNode"

type Filesystem struct {
	*cayley.Handle
	RootNode *OFile
}

func NewFs(graph *cayley.Handle) (*Filesystem, error) {
	fs := &Filesystem{graph, nil}

	root := newFile("root", fs)
	root.Id = RootNodeId
	root.mode |= os.ModeDir

	fs.RootNode = root

	if err := root.Save(); err != nil {
		return nil, err
	} else {
		fs.RootNode = root
	}

	return fs, nil
}

func (fs *Filesystem) FileWithName(parentId, name string) *OFile {
	namePath := cayley.StartPath(fs, name).In(nameLink)
	parentpath := cayley.StartPath(fs, parentId).In(parentLink)

	it := namePath.And(parentpath).BuildIterator()
	if cayley.RawNext(it) {
		return FileWithId(fs.NameOf(it.Result()), fs)
	}

	return nil
}

func (fs *Filesystem) addObject(parent, child *OFile) (err error) {
	if !parent.IsDir() {
		return errors.New("Cannot add file to a non-directory")
	} else if parent.Exists() && fs.FileWithName(parent.Id, child.name) != nil {
		return errors.New(fmt.Sprintf("File with name %s already exists in %s", child.name, parent.Name()))
	} else if !parent.Exists() {
		return errors.New(fmt.Sprint("Parent ", child.Id, " does not exist"))
	}

	child.parentId = parent.Id
	return child.Save()
}

func (fs *Filesystem) DeleteObject(of *OFile) (err error) {
	if of.Id == fs.RootNode.Id && of.Parent() == nil {
		return errors.New("Cannot delete root node")
	}

	children := of.Children()
	if len(children) > 0 {
		for i := 0; i < len(children) && err == nil; i++ {
			err = fs.DeleteObject(children[i])
		}
	}

	return fs.DeleteObject(of)
}

func (fs *Filesystem) deleteObject(of *OFile) (err error) {
	if len(of.Children()) > 0 {
		return errors.New("Can't delete file with children, must delete children first")
	}

	transaction := cayley.NewTransaction()
	if of.Mode() > 0 {
		transaction.RemoveQuad(cayley.Quad(of.Id, modeLink, fmt.Sprint(int(of.Mode())), ""))
	}
	if !of.MTime().IsZero() {
		transaction.RemoveQuad(cayley.Quad(of.Id, mTimeLink, fmt.Sprint(of.MTime().Format(timeFormat)), ""))
	}
	if of.Name() != "" {
		transaction.RemoveQuad(cayley.Quad(of.Id, nameLink, of.Name(), ""))
	}
	if of.Parent() != nil {
		transaction.RemoveQuad(cayley.Quad(of.Id, parentLink, of.Parent().Id, ""))
	}
	if of.Size() > 0 {
		transaction.RemoveQuad(cayley.Quad(of.Id, sizeLink, fmt.Sprint(of.Size()), ""))
	}

	err = fs.ApplyTransaction(transaction)

	if err == nil {
		of.mode = os.FileMode(0)
		of.size = 0
		of.mTime = time.Time{}
		of.name = ""
		of.parentId = ""
		of.Id = ""
	}

	return
}

func (fs *Filesystem) CreateDirectory(parent *OFile, name string) (f *OFile, err error) {
	child := newFile(name, fs)
	child.mode |= os.ModeDir

	if err = fs.addObject(parent, child); err != nil {
		return
	}
	return child, err
}

func (fs *Filesystem) MoveObject(of *OFile, newName, newParentId string) (err error) {
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

	newParent := FileWithId(newParentId, fs)
	if err = fs.addObject(newParent, of); err != nil {
		of.name = ""
		return
	}

	return nil
}
