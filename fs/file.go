package fs

import (
	"code.google.com/p/go-uuid/uuid"
	"errors"
	"fmt"
	"github.com/google/cayley"
	"github.com/google/cayley/graph"
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
	Id    string
	cache map[string]string

	name     string
	parentId string
	size     int64
	mode     os.FileMode
	mTime    time.Time
}

func newFile(filename string) *OFile {
	return &OFile{Id: uuid.NewUUID().String(), mode: 700, mTime: time.Now(), name: filename, cache: make(map[string]string)}
}

func FileWithId(id string) *OFile {
	return &OFile{Id: id, cache: make(map[string]string)}
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
func (of *OFile) Name() (name string) {
	var ok bool
	if name, ok = of.cache[nameLink]; !ok {
		if name = of.graphValue(nameLink); name != "" {
			of.cache[nameLink] = name
		}
	}

	return name
}

func (of *OFile) Size() (size int64) {
	var sizeString string
	var ok bool
	if sizeString, ok = of.cache[sizeLink]; !ok {
		if sizeString = of.graphValue(sizeLink); sizeString != "" {
			of.cache[sizeLink] = sizeString
		}
	}

	var err error
	if size, err = strconv.ParseInt(sizeString, 10, 64); err != nil {
		return 0
	}
	return
}

func (of *OFile) Mode() os.FileMode {
	var modeString string
	var ok bool
	if modeString, ok = of.cache[modeLink]; !ok {
		if modeString = of.graphValue(modeLink); modeString != "" {
			of.cache[modeLink] = modeString
		}
	}

	if mode, err := strconv.ParseInt(modeString, 10, 64); err != nil {
		return 0
	} else {
		return os.FileMode(mode)
	}
}

func (of *OFile) ModTime() time.Time {
	var timeString string
	var ok bool
	if timeString, ok = of.cache[mTimeLink]; !ok {
		if timeString = of.graphValue(mTimeLink); timeString != "" {
			of.cache[mTimeLink] = timeString
		}
	}

	if unixTime, err := strconv.ParseInt(timeString, 10, 64); err != nil {
		return time.Time{}
	} else {
		return time.Unix(unixTime, 0)
	}
}

func (of *OFile) IsDir() bool {
	return of.Mode()&os.ModeDir > 0
}

func (of *OFile) Sys() interface{} {
	return nil
}

// interface NTree
func (of *OFile) Parent() *OFile {
	if parentId, ok := of.cache[parentLink]; !ok {
		parentId = of.graphValue(parentLink)
		if parentId != "" {
			of.cache[parentLink] = parentId
			return FileWithId(parentId)
		} else {
			return nil
		}
	}

	return FileWithId(of.cache[parentLink])
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
	if of.name == "" && of.Name() == "" {
		return errors.New("cannot add nameless file")
	}

	staleQuads := graph.NewTransaction()
	newQuads := graph.NewTransaction()

	if of.name != of.Name() {
		if of.Name() != "" {
			staleQuads.RemoveQuad(cayley.Quad(of.Id, nameLink, of.Name(), ""))
		}
		newQuads.AddQuad(cayley.Quad(of.Id, nameLink, of.name, ""))
	}
	if of.mode != of.Mode() {
		if of.Mode() > 0 {
			staleQuads.RemoveQuad(cayley.Quad(of.Id, modeLink, fmt.Sprint(int(of.Mode())), ""))
		}
		newQuads.AddQuad(cayley.Quad(of.Id, modeLink, fmt.Sprint(int(of.mode)), ""))
	}
	if of.size != of.Size() {
		if of.Size() > 0 {
			staleQuads.RemoveQuad(cayley.Quad(of.Id, sizeLink, fmt.Sprint(of.Size()), ""))
		}
		newQuads.AddQuad(cayley.Quad(of.Id, sizeLink, fmt.Sprint(of.size), ""))
	}
	if !of.mTime.Equal(of.ModTime()) {
		if !of.ModTime().IsZero() {
			staleQuads.RemoveQuad(cayley.Quad(of.Id, mTimeLink, fmt.Sprint(of.ModTime().Unix()), ""))
		}
		newQuads.AddQuad(cayley.Quad(of.Id, mTimeLink, fmt.Sprint(of.mTime.Unix()), ""))
	}
	if of.parentId != "" && of.Parent() == nil || of.Parent() != nil && of.parentId != of.Parent().Id {
		if of.Parent() != nil {
			staleQuads.RemoveQuad(cayley.Quad(of.Id, parentLink, of.Parent().Id, ""))
		}
		newQuads.AddQuad(cayley.Quad(of.Id, parentLink, of.parentId, ""))
	}

	if err = GlobalFs().Graph.ApplyTransaction(staleQuads); err != nil {
		return
	} else if err = GlobalFs().Graph.ApplyTransaction(newQuads); err != nil {
		return
	}

	for _, delta := range staleQuads.Deltas {
		delete(of.cache, delta.Quad.Predicate)
	}

	return
}

func (of *OFile) Exists() bool {
	return of.Name() != ""
}

func (of *OFile) graphValue(key string) (value string) {
	it := cayley.StartPath(GlobalFs().Graph, of.Id).Out(key).BuildIterator()
	if cayley.RawNext(it) {
		value = GlobalFs().Graph.NameOf(it.Result())
	}

	return
}

func (of *OFile) delete() (err error) {
	if len(of.Children()) > 0 {
		return errors.New("Can't delete file with children, must delete children first")
	}

	transaction := cayley.NewTransaction()
	if of.Mode() != 0 {
		transaction.RemoveQuad(cayley.Quad(of.Id, modeLink, fmt.Sprint(int(of.Mode())), ""))
	}
	if !of.ModTime().IsZero() {
		transaction.RemoveQuad(cayley.Quad(of.Id, mTimeLink, fmt.Sprint(of.ModTime().Unix()), ""))
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

	err = GlobalFs().Graph.ApplyTransaction(transaction)

	if err == nil {
		of.mode = 1
		of.size = 0
		of.mTime = time.Time{}
		of.name = ""
		of.parentId = ""
		of.Id = ""
		of.cache = make(map[string]string)
	}

	return
}
