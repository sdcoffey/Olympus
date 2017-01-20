package api

import (
	"os"
	"testing"

	"net/http"
	"net/http/httptest"

	"github.com/sdcoffey/olympus/graph"
	"github.com/sdcoffey/olympus/graph/testutils"
	"github.com/sdcoffey/olympus/server/api"
	. "gopkg.in/check.v1"
)

func init() {
	Suite(&ApiTestSuite{})
}

type ApiTestSuite struct {
	server  *httptest.Server
	client  *http.Client
	ng      *graph.NodeGraph
	testDir string
}

func (suite *ApiTestSuite) SetUpTest(t *C) {
	suite.ng, suite.testDir = testutils.TestInit()
	suite.server = httptest.NewServer(api.NewApi(suite.ng))
	suite.client = http.DefaultClient
}

func (suite *ApiTestSuite) TearDownTest(t *C) {
	os.Remove(suite.testDir)
}

func TestNodeSuite(t *testing.T) {
	TestingT(t)
}
