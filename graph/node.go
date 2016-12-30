package graph

import (
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/cayleygraph/cayley"
	"github.com/cayleygraph/cayley/graph"
	"github.com/cayleygraph/cayley/graph/path"
	"github.com/cayleygraph/cayley/quad"
	"github.com/sdcoffey/olympus/util"
)

const (
	parentLink = "hasParent"
	nameLink   = "isNamed"
	modeLink   = "hasMode"
	mTimeLink  = "hasMTime"
)

type Node struct {
	Id    string
	graph *NodeGraph
}

func (nd *Node) Name() string {
	if val := nd.graphValue(nameLink); val != nil {
		return val.(string)
	} else {
		return ""
	}
}

func (nd *Node) Size() (sz int64) {
	for _, block := range nd.Blocks() {
		blockSize, _ := SizeOnDisk(block.Hash)
		sz += blockSize
	}

	return
}

func (nd *Node) Mode() os.FileMode {
	if val := nd.graphValue(modeLink); val != nil {
		return os.FileMode(val.(int))
	} else {
		return os.FileMode(0)
	}
}

func (nd *Node) MTime() time.Time {
	if val := nd.graphValue(mTimeLink); val != nil {
		return time.Unix(int64(val.(int)), 0)
	} else {
		return time.Time{}
	}
}

func (nd *Node) Type() string {
	return util.MimeType(nd.Name())
}

// Convenience method determining whether this node is a directory or not.
func (nd *Node) IsDir() bool {
	return nd.Mode()&os.ModeDir > 0
}

// Return the logical parent of this node, i.e. the node id with an incoming parent edge from this node.
func (nd *Node) Parent() *Node {
	if val := nd.graphValue(parentLink); val != nil {
		return nd.graph.NodeWithId(val.(string))
	} else {
		return nil
	}
}

// Return the logical children of this node, i.e, all nodes from which an incoming edge is pointed at this node.
func (nd *Node) Children() []*Node {
	if !nd.IsDir() {
		return make([]*Node, 0)
	}

	it := path.StartPath(nd.graph, quad.String(nd.Id)).In(parentLink).BuildIterator()
	children := make([]*Node, 0, 10)
	for it.Next() {
		child := nd.graph.NodeWithId(quad.NativeOf(nd.graph.NameOf(it.Result())).(string))
		children = append(children, child)
	}

	Sort(children, Alphabetical)

	return children
}

func (nd *Node) BlockWithOffset(offset int64) string {
	if nd.IsDir() {
		return ""
	}

	it := path.StartPath(nd.graph, quad.String(nd.Id)).Out(fmt.Sprint("offset-", offset)).BuildIterator()
	if it.Next() {
		return quad.NativeOf(nd.graph.NameOf(it.Result())).(string)
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
		it := path.StartPath(nd.graph, quad.String(nd.Id)).Out(fmt.Sprint("offset-", i)).BuildIterator()
		if it.Next() {
			info := BlockInfo{
				Hash:   quad.NativeOf(nd.graph.NameOf(it.Result())).(string),
				Offset: i,
			}
			blocks = append(blocks, info)
		} else {
			break
		}
	}

	return blocks
}

func (nd *Node) WriteData(data []byte, offset int64) error {
	if nd.IsDir() {
		return errors.New("Cannot write data to directory")
	} else if offset%BLOCK_SIZE != 0 {
		return errors.New(fmt.Sprintf("%d is not a valid offset for block size %d", offset, BLOCK_SIZE))
	}

	hash := Hash(data)
	transaction := graph.NewTransaction()

	// Determine if we already have a block for this offset
	linkName := fmt.Sprint("offset-", offset)
	if existingBlockHash := nd.BlockWithOffset(offset); existingBlockHash != "" {
		transaction.RemoveQuad(cayley.Triple(nd.Id, linkName, string(existingBlockHash)))
	}
	transaction.AddQuad(cayley.Triple(nd.Id, linkName, hash))

	if err := nd.graph.ApplyTransaction(transaction); err != nil {
		return err
	}

	if _, err := Write(hash, data); err != nil {
		return err
	}

	return nil
}

func (nd *Node) ancestorOf(maybeParentId string) bool {
	parent := nd.Parent()
	for parent != nil {
		if parent.Id == maybeParentId {
			return true
		}
		parent = parent.Parent()
	}

	return false
}

func (nd *Node) updateProperty(prop string, old, new interface{}) error {
	if path.StartPath(nd.graph, quad.String(nd.Id)).Out(prop).BuildIterator().Next() {
		nd.graph.RemoveQuad(cayley.Triple(nd.Id, prop, old))
	}

	return nd.graph.AddQuad(cayley.Triple(nd.Id, prop, new))
}

func (nd *Node) SetName(newName string) error {
	if existingName := nd.Name(); existingName == newName && newName != "" {
		return nil
	} else if nd.Id == RootNodeId {
		return errors.New("Error updating name: cannot rename root node")
	} else if newName == "" {
		return errors.New("Error updating name: name cannot be blank")
	} else if nd.Parent() != nil && nd.graph.NodeWithName(nd.Parent().Id, newName) != nil {
		return fmt.Errorf("Error moving node: Node with name %s already exists in %s", nd.Name(), nd.Parent().Name())
	} else if err := nd.updateProperty(nameLink, existingName, newName); err != nil {
		return fmt.Errorf("Error setting name: %s", err.Error())
	}

	return nil
}

func (nd *Node) SetMode(newMode os.FileMode) error {
	if existingMode := nd.Mode(); existingMode == newMode && int(newMode) != 0 {
		return nil
	} else if nd.Size() > 0 && newMode.IsDir() {
		return errors.New("File has size, cannot change to directory")
	} else if err := nd.updateProperty(modeLink, int(existingMode), int(newMode)); err != nil {
		return fmt.Errorf("Error setting mode: %s", err.Error())
	}

	return nil
}

func (nd *Node) Touch(newTime time.Time) error {
	if existingTime := nd.MTime(); existingTime.Equal(newTime) || newTime.IsZero() {
		return nil
	} else if newTime.After(time.Now()) {
		return errors.New("Cannot set modified time in the future")
	} else if err := nd.updateProperty(mTimeLink, existingTime.UTC().Unix(), newTime.UTC().Unix()); err != nil {
		return fmt.Errorf("Error setting mTime: %s", err.Error())
	}

	return nil
}

func (nd *Node) Move(newParentId string) error {
	newParent := nd.graph.NodeWithId(newParentId)

	if nd.Parent() != nil && nd.Parent().Id == newParentId {
		return nil
	} else if nd.Id == RootNodeId {
		return errors.New("Error moving node: Cannot move root node")
	} else if newParentId == nd.Id || newParent.ancestorOf(nd.Id) {
		return errors.New("Error moving node: Cannot move node inside itself")
	} else if !newParent.Exists() {
		return errors.New("Error moving node: Parent does not exist")
	} else if !newParent.IsDir() {
		return errors.New("Error moving node: Cannot add node to a non-directory")
	} else if nd.graph.NodeWithName(newParentId, nd.Name()) != nil {
		return fmt.Errorf("Error moving node: Node with name %s already exists in %s", nd.Name(), newParent.Name())
	}

	if nd.Parent() != nil {
		nd.graph.RemoveQuad(cayley.Triple(nd.Id, parentLink, nd.Parent().Id))
	}

	return nd.graph.AddQuad(cayley.Triple(nd.Id, parentLink, newParentId))
}

func (nd *Node) Update(info NodeInfo) error {
	updates := []func() error{
		func() error {
			if info.Name != "" {
				return nd.SetName(info.Name)
			}
			return nil
		},
		func() error {
			if info.ParentId != "" {
				return nd.Move(info.ParentId)
			}
			return nil
		},
		func() error {
			if int(info.Mode) > 0 {
				return nd.SetMode(info.Mode)
			}
			return nil
		},
		func() error {
			if !info.MTime.IsZero() {
				return nd.Touch(info.MTime)
			}
			return nil
		},
	}

	var err error
	for i := 0; i < len(updates) && err == nil; i++ {
		err = updates[i]()
	}
	return err
}

func (nd *Node) Exists() bool {
	return nd.Name() != ""
}

func (nd *Node) NodeInfo() NodeInfo {
	info := NodeInfo{
		Id:    nd.Id,
		Mode:  nd.Mode(),
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

func (nd *Node) graphValue(key string) (value interface{}) {
	it := path.StartPath(nd.graph, quad.String(nd.Id)).Out(key).BuildIterator()
	if it.Next() {
		value = quad.NativeOf(nd.graph.NameOf(it.Result()))
	} else {
		return nil
	}

	return
}
