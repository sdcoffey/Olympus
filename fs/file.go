package fs

import (
	"code.google.com/p/go-uuid/uuid"
	"errors"
	"fmt"
	"github.com/google/cayley"
	"github.com/google/cayley/quad"
	"os"
	"strconv"
	"time"
)

const (
	parentLink = "hasParent"
	sizeLink   = "hasSize"
	nameLink   = "isNamed"
	modeLink   = "modeLink"
	mTimeLink  = "hasMTime"
)

type NTree interface {
	Parent() NTree
	Children() []NTree
}

type OFile struct {
	Id       string
	name     string
	parentId string
	size     int64
	mode     os.FileMode
	mTime    time.Time
}

func newFile(filename string) *OFile {
	return &OFile{Id: uuid.NewUUID().String(), mTime: time.Now(), mode: 1, name: filename}
}

func FileWithId(id string) *OFile {
	return &OFile{Id: id, mode: 1}
}

func FileWithName(parentId, name string) *OFile {
	namePath := cayley.StartPath(GlobalFs().Graph, name).In(nameLink)
	parentpath := cayley.StartPath(GlobalFs().Graph, parentId).In(parentLink)

	it := namePath.And(parentpath).BuildIterator()
	if cayley.RawNext(it) {
		return FileWithId(GlobalFs().Graph.NameOf(it.Result()))
	}

	return nil
}

// interface FileInfo
func (of *OFile) Name() string {
	if of.name == "" {
		it := cayley.StartPath(GlobalFs().Graph, of.Id).Out(nameLink).BuildIterator()
		if cayley.RawNext(it) {
			of.name = GlobalFs().Graph.NameOf(it.Result())
		} else {
			of.name = ""
		}
	}

	return of.name
}

func (of *OFile) Size() int64 {
	if of.size == 0 {
		it := cayley.StartPath(GlobalFs().Graph, of.Id).Out(sizeLink).BuildIterator()
		if cayley.RawNext(it) {
			of.size, _ = strconv.ParseInt(GlobalFs().Graph.NameOf(it.Result()), 10, 64)
		} else {
			of.size = 0
		}
	}

	return of.size
}

func (of *OFile) Mode() os.FileMode {
	if of.mode > 0 && of.mode < os.ModeSticky {
		it := cayley.StartPath(GlobalFs().Graph, of.Id).Out(modeLink).BuildIterator()
		if cayley.RawNext(it) {
			mode, _ := strconv.ParseInt(GlobalFs().Graph.NameOf(it.Result()), 10, 64)
			of.mode = os.FileMode(mode)
		} else {
			of.mode = 1
		}
	}

	return of.mode
}

func (of *OFile) ModTime() time.Time {
	if of.mTime.IsZero() {
		it := cayley.StartPath(GlobalFs().Graph, of.Id).Out(mTimeLink).BuildIterator()
		if cayley.RawNext(it) {
			unixTime, _ := strconv.ParseInt(GlobalFs().Graph.NameOf(it.Result()), 10, 64)
			of.mTime = time.Unix(unixTime, 0)
		} else {
			of.mTime = time.Time{}
		}
	}

	return of.mTime
}

func (of *OFile) IsDir() bool {
	return of.Mode()&os.ModeDir > 0
}

func (of *OFile) Sys() interface{} {
	return nil
}

// interface NTree
func (of *OFile) Parent() *OFile {
	if of.parentId == "" {
		it := cayley.StartPath(GlobalFs().Graph, of.Id).Out(parentLink).BuildIterator()
		if cayley.RawNext(it) {
			id := GlobalFs().Graph.NameOf(it.Result())
			if id != "" {
				of.parentId = id
				return FileWithId(id)
			}
		} else {
			of.parentId = ""
		}
	} else {
		return FileWithId(of.parentId)
	}

	return nil
}

func (of *OFile) Children() []*OFile {
	if !of.IsDir() {
		return make([]*OFile, 0)
	}

	it := cayley.StartPath(GlobalFs().Graph, of.Id).In(parentLink).BuildIterator()
	children := make([]*OFile, 0, 10)
	for cayley.RawNext(it) {
		child := FileWithId(GlobalFs().Graph.NameOf(it.Result()))
		child.parentId = of.Id
		children = append(children, child)
	}

	return children
}

// interface GraphWriter
func (of *OFile) Write() (err error) {
	if of.Parent() != nil && FileWithName(of.Parent().Id, of.name) != nil {
		return errors.New(fmt.Sprintf("File with name %s already exists in %s", of.Name(), of.Parent().Name()))
	} else if of.name == "" {
		return errors.New("cannot add nameless file")
	}

	quads := make([]quad.Quad, 4, 5)
	quads[0] = cayley.Quad(of.Id, sizeLink, fmt.Sprint(of.size), "")
	quads[1] = cayley.Quad(of.Id, modeLink, fmt.Sprint(int(of.mode)), "")
	quads[2] = cayley.Quad(of.Id, mTimeLink, fmt.Sprint(of.mTime.Unix()), "")
	quads[3] = cayley.Quad(of.Id, nameLink, of.name, "")

	if of.parentId != "" {
		quads = append(quads, cayley.Quad(of.Id, parentLink, of.parentId, ""))
	}

	for i := 0; i < len(quads) && err == nil; i++ {
		err = GlobalFs().Graph.AddQuad(quads[i])
		if err != nil && err.Error() == "quad exists" {
			err = nil
		}
	}

	return
}

func (of *OFile) Exists() bool {
	return of.Name() != ""
}

func (of *OFile) delete() (err error) {
	if len(of.Children()) > 0 {
		return errors.New("Can't delete file with children, must delete children first")
	}

	transaction := cayley.NewTransaction()
	transaction.RemoveQuad(cayley.Quad(of.Id, modeLink, fmt.Sprint(int(of.Mode())), ""))
	transaction.RemoveQuad(cayley.Quad(of.Id, mTimeLink, fmt.Sprint(of.ModTime().Unix()), ""))
	transaction.RemoveQuad(cayley.Quad(of.Id, nameLink, of.Name(), ""))

	if of.Parent() != nil {
		transaction.RemoveQuad(cayley.Quad(of.Id, parentLink, of.Parent().Id, ""))
	}
	if of.Size() > 0 {
		transaction.RemoveQuad(cayley.Quad(of.Id, sizeLink, fmt.Sprint(of.Size()), ""))
	}

	err = GlobalFs().Graph.ApplyTransaction(transaction)

	if err == nil {
		of.mode = 1
		of.size = 0
		of.mTime = time.Time{}
		of.name = ""
		of.parentId = ""
		of.Id = ""
	}

	return
}
