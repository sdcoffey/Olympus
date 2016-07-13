package api

import (
	"os"
	"testing"

	"net/http"
	"net/http/httptest"

	. "github.com/sdcoffey/olympus/Godeps/_workspace/src/gopkg.in/check.v1"
	"github.com/sdcoffey/olympus/graph"
	"github.com/sdcoffey/olympus/graph/testutils"
	"github.com/sdcoffey/olympus/server/api"
)

func init() {
	Suite(&ApiTestSuite{})
}

type ApiTestSuite struct {
	server    *httptest.Server
	client    *http.Client
	serverUrl string
	ng        *graph.NodeGraph
	testDir   string
}

func (suite *ApiTestSuite) SetUpTest(t *C) {
	suite.ng, suite.testDir = testutils.TestInit()
	suite.server = httptest.NewServer(api.NewApi(suite.ng))
	suite.client = http.DefaultClient
	suite.serverUrl = suite.server.URL
}

func (suite *ApiTestSuite) TearDownTest(t *C) {
	os.Remove(suite.testDir)
}

func TestNodeSuite(t *testing.T) {
	TestingT(t)
}
