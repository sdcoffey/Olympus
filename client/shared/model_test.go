package shared

import (
	"fmt"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/sdcoffey/olympus/Godeps/_workspace/src/github.com/google/cayley"
	. "github.com/sdcoffey/olympus/Godeps/_workspace/src/gopkg.in/check.v1"
	"github.com/sdcoffey/olympus/client/apiclient"
	"github.com/sdcoffey/olympus/graph"
	"github.com/sdcoffey/olympus/graph/testutils"
	"github.com/sdcoffey/olympus/server/api"
)

func init() {
	Suite(&ModelTestSuite{})
}

type ModelTestSuite struct {
	client    apiclient.ApiClient
	nodeGraph *graph.NodeGraph
	server    *httptest.Server
	tmpDir    string
}

func TestModelTestSuite(t *testing.T) {
	TestingT(t)
}

func (t *ModelTestSuite) SetUpTest(c *C) {
	memGraph, _ := cayley.NewMemoryGraph()
	memNg, _ := graph.NewGraph(memGraph)

	t.nodeGraph, t.tmpDir = testutils.TestInit()
	t.server = httptest.NewServer(api.NewApi(memNg))
	t.client = apiclient.ApiClient{t.server.URL, api.JsonEncoding}
}

func (t *ModelTestSuite) TearDownTest(c *C) {
	os.Remove(t.tmpDir)
}

func (t *ModelTestSuite) TestModel_Init_returnsErrIfRootDoestNotExist(c *C) {
	rootNode := t.nodeGraph.NodeWithId("not-an-id")
	model := newModel(t.client, rootNode, t.nodeGraph)

	c.Check(model.init(), ErrorMatches, "Root with id: not-an-id does not exist")
}

func (t *ModelTestSuite) TestModel_Init_doesNotReturnErrorIfRootExists(c *C) {
	model := newModel(t.client, t.nodeGraph.RootNode, t.nodeGraph)
	c.Check(model.init(), IsNil)
}

func (t *ModelTestSuite) TestModel_initRefreshesCache(c *C) {
	model := newModel(t.client, t.nodeGraph.RootNode, t.nodeGraph)

	for i := 0; i < 3; i++ {
		_, err := t.client.CreateNode(graph.NodeInfo{
			ParentId: graph.RootNodeId,
			Name:     fmt.Sprint(i),
		})
		c.Check(err, IsNil)
	}

	err := model.init()
	c.Check(err, IsNil)
	c.Check(3, Equals, len(model.Root.Children()))
}

func (t *ModelTestSuite) TestModel_Refresh_RemovesLocalItemsDeletedRemotely(c *C) {
	model := newModel(t.client, t.nodeGraph.RootNode, t.nodeGraph)

	for i := 0; i < 3; i++ {
		_, err := t.client.CreateNode(graph.NodeInfo{
			ParentId: graph.RootNodeId,
			Name:     fmt.Sprint(i),
		})
		c.Check(err, IsNil)
	}

	err := model.init()
	c.Check(err, IsNil)
	c.Check(3, Equals, len(model.Root.Children()))
	children := model.graph.NodeWithId(graph.RootNodeId).Children()
	err = t.client.RemoveNode(children[0].Id)
	c.Check(err, IsNil)

	err = model.Refresh()
	c.Check(err, IsNil)
	c.Check(2, Equals, len(model.Root.Children()))
}

func (t *ModelTestSuite) TestModel_FindNodeByName_ReturnsCorrectNode(c *C) {
	model := newModel(t.client, t.nodeGraph.RootNode, t.nodeGraph)

	for i := 0; i < 3; i++ {
		_, err := t.client.CreateNode(graph.NodeInfo{
			ParentId: graph.RootNodeId,
			Name:     fmt.Sprint(i),
		})
		c.Check(err, IsNil)
	}

	err := model.init()
	c.Check(err, IsNil)

	node := model.FindNodeByName("1")
	c.Check(node.Parent().Id, Equals, t.nodeGraph.RootNode.Id)
}
