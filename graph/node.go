package graph

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/sdcoffey/olympus/Godeps/_workspace/src/code.google.com/p/go-uuid/uuid"
	"github.com/sdcoffey/olympus/Godeps/_workspace/src/github.com/google/cayley"
	"github.com/sdcoffey/olympus/Godeps/_workspace/src/github.com/google/cayley/graph"
)

const (
	parentLink = "hasParent"
	nameLink   = "isNamed"
	modeLink   = "modeLink"
	mTimeLink  = "hasMTime"
	typeLink   = "hasType"

	timeFormat = time.RFC3339Nano
)

type NTree interface {
	Parent() NTree
	Children() []NTree
}

type Node struct {
	Id       string
	graph    *NodeGraph
	parentId string
	name     string
	mTime    time.Time
	mode     os.FileMode
	mimeType string
}

func newNode(name string, graph *NodeGraph) *Node {
	file := new(Node)
	file.Id = uuid.New()
	file.mTime = time.Now()
	file.name = name
	file.graph = graph
	return file
}

func (nd *Node) Name() string {
	return nd.graphValue(nameLink)
}

func (nd *Node) Size() (sz int64) {
	for _, block := range nd.Blocks() {
		blockSize, _ := SizeOnDisk(block.Hash)
		sz += blockSize
	}

	return
}

func (nd *Node) Mode() os.FileMode {
	modeString := nd.graphValue(modeLink)
	if mode, err := strconv.ParseInt(modeString, 10, 64); err != nil {
		return os.FileMode(0)
	} else {
		return os.FileMode(mode)
	}
}

func (nd *Node) MTime() time.Time {
	timeString := nd.graphValue(mTimeLink)
	if t, err := time.Parse(timeFormat, timeString); err != nil {
		return time.Time{}
	} else {
		return t
	}
}

func (nd *Node) Type() string {
	return nd.graphValue(typeLink)
}

func (nd *Node) IsDir() bool {
	return nd.Mode()&os.ModeDir > 0
}

// interface NTree
func (nd *Node) Parent() *Node {
	var parentId string
	if parentId = nd.graphValue(parentLink); parentId == "" {
		return nil
	}

	return nd.graph.NodeWithId(parentId)
}

func (nd *Node) Children() []*Node {
	if !nd.IsDir() {
		return make([]*Node, 0)
	}

	it := cayley.StartPath(nd.graph, nd.Id).In(parentLink).BuildIterator()
	children := make([]*Node, 0, 10)
	for cayley.RawNext(it) {
		child := nd.graph.NodeWithId(nd.graph.NameOf(it.Result()))
		child.parentId = nd.Id
		children = append(children, child)
	}

	return children
}

func (nd *Node) BlockWithOffset(offset int64) string {
	if nd.IsDir() {
		return ""
	}

	it := cayley.StartPath(nd.graph, nd.Id).Out(fmt.Sprint("offset-", offset)).BuildIterator()
	if cayley.RawNext(it) {
		return nd.graph.NameOf(it.Result())
	} else {
		return ""
	}
}

func (nd *Node) Blocks() []BlockInfo {
	if nd.IsDir() {
		return make([]BlockInfo, 0)
	}

	blocks := make([]BlockInfo, 0, 4) // TODO: don't guess

	var i int64
	for i = 0; ; i += BLOCK_SIZE {
		it := cayley.StartPath(nd.graph, nd.Id).Out(fmt.Sprint("offset-", i)).BuildIterator()
		if cayley.RawNext(it) {
			info := BlockInfo{
				Hash:   nd.graph.NameOf(it.Result()),
				Offset: i,
			}
			blocks = append(blocks, info)
		} else {
			return blocks
		}
	}

	return blocks
}

func (nd *Node) WriteData(data []byte, offset int64) error {
	if offset%BLOCK_SIZE != 0 {
		return errors.New(fmt.Sprintf("%d is not a valid offset for block size %d", offset, BLOCK_SIZE))
	}

	hash := Hash(data)
	transaction := graph.NewTransaction()

	// Determine if we already have a block for this offset
	linkName := fmt.Sprint("offset-", offset)
	if existingBlockHash := nd.BlockWithOffset(offset); existingBlockHash != "" {
		transaction.RemoveQuad(cayley.Quad(nd.Id, linkName, string(existingBlockHash), ""))
	}
	transaction.AddQuad(cayley.Quad(nd.Id, linkName, hash, ""))

	if err := nd.graph.ApplyTransaction(transaction); err != nil {
		return err
	}

	if _, err := Write(hash, data); err != nil {
		return err
	}

	return nil
}

func (nd *Node) Save() (err error) {
	if nd.name == "" && nd.Name() == "" {
		return errors.New("Cannot add nameless file")
	} else if nd.mTime.After(time.Now()) {
		nd.mTime = time.Time{}
		return errors.New("Cannot set futuristic mTime")
	} else if nd.Parent() == nil && nd.parentId == "" && nd.Id != RootNodeId {
		return errors.New("Cannot add file without parent")
	}

	staleQuads := graph.NewTransaction()
	newQuads := graph.NewTransaction()
	if name := nd.Name(); nd.name != name && name != "" && nd.name != "" {
		staleQuads.RemoveQuad(cayley.Quad(nd.Id, nameLink, name, ""))
	}
	if nd.name != "" && nd.name != nd.Name() {
		newQuads.AddQuad(cayley.Quad(nd.Id, nameLink, nd.name, ""))
		nd.name = ""
	}

	if mimeType := nd.Type(); nd.mimeType != mimeType && mimeType != "" && nd.mimeType != "" {
		staleQuads.RemoveQuad(cayley.Quad(nd.Id, typeLink, mimeType, ""))
	}
	if nd.mimeType != "" && nd.mimeType != nd.Type() {
		newQuads.AddQuad(cayley.Quad(nd.Id, typeLink, nd.mimeType, ""))
		nd.mimeType = ""
	}

	if mode := int(nd.Mode()); int(nd.mode) != mode && mode != 0 && int(nd.mode) != 0 {
		staleQuads.RemoveQuad(cayley.Quad(nd.Id, modeLink, fmt.Sprint(mode), ""))
	}
	if int(nd.mode) > 0 && nd.mode != nd.Mode() {
		newQuads.AddQuad(cayley.Quad(nd.Id, modeLink, fmt.Sprint(int(nd.mode)), ""))
		nd.mode = os.FileMode(0)
	}

	if mTime := nd.MTime(); nd.mTime != mTime && !mTime.IsZero() && !nd.mTime.IsZero() {
		staleQuads.RemoveQuad(cayley.Quad(nd.Id, mTimeLink, nd.MTime().Format(timeFormat), ""))
	}
	if !nd.mTime.IsZero() && nd.mTime != nd.MTime() {
		newQuads.AddQuad(cayley.Quad(nd.Id, mTimeLink, nd.mTime.Format(timeFormat), ""))
		nd.mTime = time.Time{}
	}

	if parent := nd.Parent(); parent != nil && parent.Id != nd.parentId && nd.parentId != "" {
		staleQuads.RemoveQuad(cayley.Quad(nd.Id, parentLink, nd.Parent().Id, ""))
	} else if parent != nil && parent.Id == nd.parentId {
		nd.parentId = ""
	}
	if nd.parentId != "" {
		newQuads.AddQuad(cayley.Quad(nd.Id, parentLink, nd.parentId, ""))
		nd.parentId = ""
	}

	if err = nd.graph.ApplyTransaction(staleQuads); err != nil {
		return
	} else if err = nd.graph.ApplyTransaction(newQuads); err != nil {
		return
	}

	return nil
}

func (nd *Node) Chmod(newMode os.FileMode) error {
	nd.mode = newMode
	return nd.Save()
}

func (nd *Node) Touch(mTime time.Time) error {
	nd.mTime = mTime
	return nd.Save()
}

func (nd *Node) Exists() bool {
	return nd.Name() != ""
}

func (nd *Node) NodeInfo() NodeInfo {
	info := NodeInfo{
		Id:    nd.Id,
		Mode:  uint32(nd.Mode()),
		MTime: nd.MTime(),
		Name:  nd.Name(),
		Size:  nd.Size(),
		Type:  nd.Type(),
	}
	if nd.Parent() != nil {
		info.ParentId = nd.Parent().Id
	}

	return info
}

func (nd *Node) ReadSeeker() *NodeSeeker {
	rs := new(NodeSeeker)
	rs.node = nd
	return rs
}

type NodeSeeker struct {
	node   *Node
	offset int64
}

func (ns *NodeSeeker) Seek(offset int64, whence int) (int64, error) {
	if whence == 0 {
		ns.offset = offset
	} else if whence == 1 {
		ns.offset += offset
	} else if whence == 2 {
		ns.offset = ns.node.Size() - offset
	}

	if offset < 0 {
		return offset, errors.New("Offset set before beginning of file")
	} else {
		return ns.offset, nil
	}
}

func (ns *NodeSeeker) Read(p []byte) (n int, err error) {
	if ns.offset > ns.node.Size() {
		return 0, io.EOF
	}

	blockOffset := (ns.offset / BLOCK_SIZE) * BLOCK_SIZE
	block := ns.node.BlockWithOffset(blockOffset)

	if dat, err := RawData(block); err != nil {
		return 0, err
	} else {
		relOffset := int(ns.offset % BLOCK_SIZE)
		end := 0
		if len(p) > len(dat)-relOffset {
			end = len(dat) - relOffset
		} else {
			end = len(p)
		}

		var i int
		for i = 0; i < end; i++ {
			p[i] = dat[i+relOffset]
		}
		ns.offset += int64(i)

		return i, nil
	}
}

func (nd *Node) String() string {
	return fmt.Sprintf("%s	%d	%s	%s (%s)", nd.Mode(), nd.Size(), nd.MTime().Format(time.Stamp), nd.Name(), nd.Id)
}

func (nd *Node) graphValue(key string) (value string) {
	it := cayley.StartPath(nd.graph, nd.Id).Out(key).BuildIterator()
	if cayley.RawNext(it) {
		value = nd.graph.NameOf(it.Result())
	}

	return
}
