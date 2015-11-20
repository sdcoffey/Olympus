package fs

import (
	"github.com/google/cayley"
	"github.com/sdcoffey/olympus/fs/model"
	"testing"
)

func TestAddFile(t *testing.T) {
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
	if f.Id == "" {
		t.Error("Id should not have been empty")
	}

	p := cayley.StartPath(cGraph, f.Id).Out("hasParent")
	p.Or(cayley.StartPath(cGraph, f.Id).Out("hasSize"))
	p.Or(cayley.StartPath(cGraph, f.Id).Out("hasAttr"))
	p.Or(cayley.StartPath(cGraph, f.Id).Out("isNamed"))

	it := p.BuildIterator()
	cayley.RawNext(it)

	if cGraph.NameOf(it.Result()) != "" {
		t.Error("Parent should have been zero")
	}
	cayley.RawNext(it)
	if cGraph.NameOf(it.Result()) != "1024" {
		t.Error("Size should have been 1024")
	}
	cayley.RawNext(it)
	if cGraph.NameOf(it.Result()) != "16" {
		t.Error("Attr should have been 16")
	}
	cayley.RawNext(it)
	if cGraph.NameOf(it.Result()) != "Folder" {
		t.Error("Name should have been Folder")
	}
}

/* todo:
   - TestStatFile
   - TestListFiles
   - TestSetsIdForNewFile
   - TestThrowsWhenAddingFileToNonFolder
*/
