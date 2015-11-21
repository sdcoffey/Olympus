package fs

import (
	"github.com/google/cayley"
	"github.com/sdcoffey/olympus/fs/model"
	testify "github.com/stretchr/testify/assert"
	"testing"
)

func TestAddFile(t *testing.T) {
	cGraph, _ := cayley.NewMemoryGraph()
	fs := NewFs(cGraph)

	f := model.NewFile()
	f.ParentId = ""
	f.Size = 1024
	f.Name = "Folder"
	f.Attr = 16

	err := fs.AddFile(f)

	assert := testify.New(t)

	assert.Nil(err)
	assert.NotEmpty(f.Id)

	p := cayley.StartPath(cGraph, f.Id).Out("hasParent")
	p.Or(cayley.StartPath(cGraph, f.Id).Out("hasSize"))
	p.Or(cayley.StartPath(cGraph, f.Id).Out("hasAttr"))
	p.Or(cayley.StartPath(cGraph, f.Id).Out("isNamed"))

	it := p.BuildIterator()
	cayley.RawNext(it)

	assert.Empty(cGraph.NameOf(it.Result()))

	cayley.RawNext(it)
	assert.Equal("1024", cGraph.NameOf(it.Result()))

	cayley.RawNext(it)
	assert.Equal("16", cGraph.NameOf(it.Result()))

	cayley.RawNext(it)
	assert.Equal("Folder", cGraph.NameOf(it.Result()))
}

func TestStat(t *testing.T) {
	cGraph, _ := cayley.NewMemoryGraph()
	fs := NewFs(cGraph)

	f := model.OFile{
		ParentId: "",
		Size:     1024,
		Name:     "Folder",
		Attr:     16,
	}
	err := fs.AddFile(&f)
	if err != nil {
		t.Error(err)
	}

	file := model.NewFileWithId(f.Id)
	err = fs.Stat(file)
	if err != nil {
		t.Error(err)
	}

	assert := testify.New(t)
	assert.EqualValues(f.Size, file.Size)
	assert.Equal(f.Name, file.Name)
	assert.EqualValues(f.Attr, file.Attr)
	assert.Equal(f.ParentId, file.ParentId)
}

func TestListChildren(t *testing.T) {
	assert := testify.New(t)

	cGraph, _ := cayley.NewMemoryGraph()
	fs := NewFs(cGraph)

	parent := model.NewFile()
	parent.Attr = model.AttrDir
	parent.Name = "root"
	parent.Size = 1034

	child := model.NewFile()
	child.ParentId = parent.Id
	child.Name = "child1"
	child.Attr = model.AttrDir

	child2 := model.NewFile()
	child2.ParentId = parent.Id
	child2.Name = "child2"

	child3 := model.NewFile()
	child3.ParentId = child.Id
	child3.Name = "child3"

	err := fs.AddFile(parent)
	assert.Nil(err)

	err = fs.AddFile(child)
	assert.Nil(err)

	err = fs.AddFile(child2)
	assert.Nil(err)

	err = fs.AddFile(child3)
	assert.Nil(err)

	children, err := fs.ListFiles(parent.Id)
	assert.Nil(err)
	assert.EqualValues(2, len(children))
	assert.EqualValues(child.Name, children[0].Name)
	assert.EqualValues(child2.Name, children[1].Name)

	children, err = fs.ListFiles(child.Id)
	assert.Nil(err)
	assert.EqualValues(1, len(children))
	assert.EqualValues(child3.Name, children[0].Name)
}

func TestReturnsErrorWhenParentIsNonFolder(t *testing.T) {
	cGraph, _ := cayley.NewMemoryGraph()
	fs := NewFs(cGraph)

	parent := model.NewFile()
	parent.Name = "root"

	fs.AddFile(parent)

	child := model.NewFile()
	child.Name = "child"
	child.ParentId = parent.Id

	err := fs.AddFile(child)
	testify.NotNil(t, err)
}
