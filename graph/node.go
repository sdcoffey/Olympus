package graph

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
	size     int64
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

func (nd *Node) Size() int64 {
	if nd.IsDir() {
		return 0
	}

	sizeString := nd.graphValue(sizeLink)
	if size, err := strconv.ParseInt(sizeString, 10, 64); err != nil {
		return 0
	} else {
		return size
	}
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

	blockCap := nd.Size() / BLOCK_SIZE
	if nd.Size()%BLOCK_SIZE > 0 {
		blockCap++
	}

	blocks := make([]BlockInfo, 0, blockCap)
	var i int64
	for i = 0; i < nd.Size(); i += BLOCK_SIZE {
		it := cayley.StartPath(nd.graph, nd.Id).Out(fmt.Sprint("offset-", i)).BuildIterator()
		for cayley.RawNext(it) {
			info := BlockInfo{
				Hash:   nd.graph.NameOf(it.Result()),
				Offset: i,
			}
			blocks = append(blocks, info)
		}
	}

	return blocks
}

func (nd *Node) WriteData(data []byte, offset int64) error {
	if offset%BLOCK_SIZE != 0 {
		return errors.New(fmt.Sprintf("%d is not a valid offset for block size %d", offset, BLOCK_SIZE))
	} else if int64(len(data)) > nd.Size() {
		return errors.New("Cannot write data that exceeds the size of file")
	}

	hash := Hash(data)
	transaction := graph.NewTransaction()

	// Determine if we already have a block for this offset
	if existingBlockHash := nd.BlockWithOffset(offset); existingBlockHash != "" {
		transaction.RemoveQuad(cayley.Quad(nd.Id, fmt.Sprint("offset-", offset), string(existingBlockHash), ""))
	}
	transaction.AddQuad(cayley.Quad(nd.Id, fmt.Sprint("offset-", offset), hash, ""))

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
	} else if nd.size < 0 {
		nd.size = 0
		return errors.New("File cannot have negative size")
	} else if (nd.mode&os.ModeDir > 0) && nd.size != 0 {
		nd.mode = os.FileMode(0)
		return errors.New("Dir cannot have non-zero size")
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

	if size := nd.Size(); nd.size != size && size != 0 && nd.size != 0 {
		staleQuads.RemoveQuad(cayley.Quad(nd.Id, sizeLink, fmt.Sprint(nd.Size()), ""))
	}
	if nd.size > 0 && nd.size != nd.Size() {
		newQuads.AddQuad(cayley.Quad(nd.Id, sizeLink, fmt.Sprint(nd.size), ""))
		nd.size = 0
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

func (nd *Node) Resize(newSize int64) error {
	nd.size = newSize
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
