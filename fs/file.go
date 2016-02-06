package fs

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"code.google.com/p/go-uuid/uuid"
	"github.com/google/cayley"
	"github.com/google/cayley/graph"
)

const (
	parentLink = "hasParent"
	nameLink   = "isNamed"
	modeLink   = "modeLink"
	mTimeLink  = "hasMTime"
	sizeLink   = "hasSize"

	timeFormat = time.RFC3339Nano
)

type NTree interface {
	Parent() NTree
	Children() []NTree
}

type OFile struct {
	Id       string
	fs       *Filesystem
	parentId string
	name     string
	size     int64
	mTime    time.Time
	mode     os.FileMode
}

func newFile(filename string, fs *Filesystem) *OFile {
	file := new(OFile)
	file.Id = uuid.New()
	file.mTime = time.Now()
	file.name = filename
	file.fs = fs
	return file
}

func FileWithFileInfo(info FileInfo, fs *Filesystem) *OFile {
	file := FileWithId(info.Id, fs)
	file.fs = fs
	file.name = info.Name
	file.mode = os.FileMode(info.Mode)
	file.mTime = info.MTime
	file.parentId = info.ParentId

	return file
}

func FileWithId(id string, fs *Filesystem) *OFile {
	return &OFile{Id: id, fs: fs}
}

// interface FileInfo
func (of *OFile) Name() string {
	if name := of.graphValue(nameLink); name != "" {
		return name
	} else {
		return ""
	}
}

func (of *OFile) Size() int64 {
	if of.IsDir() {
		return 0
	}

	sizeString := of.graphValue(sizeLink)
	if size, err := strconv.ParseInt(sizeString, 10, 64); err != nil {
		return 0
	} else {
		return size
	}
}

func (of *OFile) Mode() os.FileMode {
	modeString := of.graphValue(modeLink)
	if mode, err := strconv.ParseInt(modeString, 10, 64); err != nil {
		return os.FileMode(0)
	} else {
		return os.FileMode(mode)
	}
}

func (of *OFile) MTime() time.Time {
	timeString := of.graphValue(mTimeLink)
	if t, err := time.Parse(timeFormat, timeString); err != nil {
		return time.Time{}
	} else {
		return t
	}
}

func (of *OFile) IsDir() bool {
	return of.Mode()&os.ModeDir > 0
}

// interface NTree
func (of *OFile) Parent() *OFile {
	var parentId string
	if parentId = of.graphValue(parentLink); parentId == "" {
		return nil
	}

	return FileWithId(parentId, of.fs)
}

func (of *OFile) Children() []*OFile {
	if !of.IsDir() {
		return make([]*OFile, 0)
	}

	it := cayley.StartPath(of.fs, of.Id).In(parentLink).BuildIterator()
	children := make([]*OFile, 0, 10)
	for cayley.RawNext(it) {
		child := FileWithId(of.fs.NameOf(it.Result()), of.fs)
		child.parentId = of.Id
		children = append(children, child)
	}

	return children
}

func (of *OFile) BlockWithOffset(offset int64) string {
	if of.IsDir() {
		return ""
	}

	it := cayley.StartPath(of.fs, of.Id).Out(fmt.Sprint("offset-", offset)).BuildIterator()
	if cayley.RawNext(it) {
		return of.fs.NameOf(it.Result())
	} else {
		return ""
	}
}

func (of *OFile) Blocks() []BlockInfo {
	if of.IsDir() {
		return make([]BlockInfo, 0)
	}

	blockCap := of.Size() / BLOCK_SIZE
	if of.Size()%BLOCK_SIZE > 0 {
		blockCap++
	}

	blocks := make([]BlockInfo, 0, blockCap)
	var i int64
	for i = 0; i < of.Size(); i += BLOCK_SIZE {
		it := cayley.StartPath(of.fs, of.Id).Out(fmt.Sprint("offset-", i)).BuildIterator()
		for cayley.RawNext(it) {
			info := BlockInfo{
				Hash:   of.fs.NameOf(it.Result()),
				Offset: i,
			}
			blocks = append(blocks, info)
		}
	}

	return blocks
}

func (of *OFile) WriteData(data []byte, offset int64) error {
	if offset%BLOCK_SIZE != 0 {
		return errors.New(fmt.Sprintf("%d is not a valid offset for block size %d", offset, BLOCK_SIZE))
	} else if int64(len(data)) > of.Size() {
		return errors.New("Cannot write data that exceeds the size of file")
	}

	hash := Hash(data)
	transaction := graph.NewTransaction()

	// Determine if we already have a block for this offset
	if existingBlockHash := of.BlockWithOffset(offset); existingBlockHash != "" {
		transaction.RemoveQuad(cayley.Quad(of.Id, fmt.Sprint("offset-", offset), string(existingBlockHash), ""))
	}
	transaction.AddQuad(cayley.Quad(of.Id, fmt.Sprint("offset-", offset), hash, ""))

	if err := of.fs.ApplyTransaction(transaction); err != nil {
		return err
	}

	if _, err := Write(hash, data); err != nil {
		return err
	}

	return nil
}

func (of *OFile) Save() (err error) {
	if of.name == "" && of.Name() == "" {
		return errors.New("Cannot add nameless file")
	} else if of.mTime.After(time.Now()) {
		of.mTime = time.Time{}
		return errors.New("Cannot set futuristic mTime")
	} else if of.size < 0 {
		of.size = 0
		return errors.New("File cannot have negative size")
	} else if (of.mode&os.ModeDir > 0) && of.size != 0 {
		of.mode = os.FileMode(0)
		return errors.New("Dir cannot have non-zero size")
	} else if (of.mode&os.ModeDir > 0) != of.IsDir() {
		of.mode = os.FileMode(0)
		return errors.New("Cannot change between file and directory")
	}

	staleQuads := graph.NewTransaction()
	newQuads := graph.NewTransaction()
	if of.name != of.Name() {
		staleQuads.RemoveQuad(cayley.Quad(of.Id, nameLink, of.Name(), ""))
	}
	newQuads.AddQuad(cayley.Quad(of.Id, nameLink, of.name, ""))

	if of.size != of.Size() {
		staleQuads.RemoveQuad(cayley.Quad(of.Id, sizeLink, fmt.Sprint(of.Size()), ""))
	}
	if of.size > 0 {
		newQuads.AddQuad(cayley.Quad(of.Id, sizeLink, fmt.Sprint(of.size), ""))
		of.size = 0
	}

	if of.mode != of.Mode() {
		staleQuads.RemoveQuad(cayley.Quad(of.Id, modeLink, fmt.Sprint(int(of.Mode())), ""))
	}
	if int(of.mode) > 0 {
		newQuads.AddQuad(cayley.Quad(of.Id, modeLink, fmt.Sprint(int(of.mode)), ""))
		of.mode = os.FileMode(0)
	}

	if of.mTime != of.MTime() {
		staleQuads.RemoveQuad(cayley.Quad(of.Id, mTimeLink, of.MTime().Format(timeFormat), ""))
	}
	if !of.mTime.IsZero() {
		newQuads.AddQuad(cayley.Quad(of.Id, mTimeLink, of.mTime.Format(timeFormat), ""))
		of.mTime = time.Time{}
	}

	if of.parentId != "" && of.Parent() != nil && of.parentId != of.Parent().Id {
		staleQuads.RemoveQuad(cayley.Quad(of.Id, parentLink, of.Parent().Id, ""))
	}
	if of.parentId != "" {
		newQuads.AddQuad(cayley.Quad(of.Id, parentLink, of.parentId, ""))
		of.parentId = ""
	}

	if err = of.fs.ApplyTransaction(staleQuads); err != nil {
		return
	} else if err = of.fs.ApplyTransaction(newQuads); err != nil {
		return
	}

	return nil
}

func (of *OFile) graphValue(key string) (value string) {
	it := cayley.StartPath(of.fs, of.Id).Out(key).BuildIterator()
	if cayley.RawNext(it) {
		value = of.fs.NameOf(it.Result())
	}

	return
}

func (of *OFile) Chmod(newMode os.FileMode) error {
	of.mode = newMode
	return of.Save()
}

func (of *OFile) Resize(newSize int64) error {
	of.size = newSize
	return of.Save()
}

func (of *OFile) Touch(mTime time.Time) error {
	of.mTime = mTime
	return of.Save()
}

func (of *OFile) Exists() bool {
	return of.Name() != ""
}

func (of *OFile) FileInfo() FileInfo {
	info := FileInfo{}
	info.Mode = uint32(of.Mode())
	info.Id = of.Id
	info.MTime = of.MTime()
	info.Name = of.Name()
	info.Size = of.Size()
	if of.Parent() != nil {
		info.ParentId = of.Parent().Id
	}

	return info
}

func (of *OFile) String() string {
	return fmt.Sprintf("%s	%d	%s	%s (%s)", of.Mode(), of.Size(), of.MTime().Format(time.Stamp), of.Name(), of.Id)
}
