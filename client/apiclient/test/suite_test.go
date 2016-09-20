package test

import (
	"net/http/httptest"
	"os"
	"testing"

	"github.com/sdcoffey/olympus/client/apiclient"
	"github.com/sdcoffey/olympus/graph"
	"github.com/sdcoffey/olympus/graph/testutils"
	"github.com/sdcoffey/olympus/server/api"
	. "gopkg.in/check.v1"
)

func init() {
	Suite(&ApiClientTestSuite{})
}

type ApiClientTestSuite struct {
	server  *httptest.Server
	ng      *graph.NodeGraph
	client  apiclient.ApiClient
	testDir string
}

func (suite *ApiClientTestSuite) SetUpTest(t *C) {
	suite.ng, suite.testDir = testutils.TestInit()
	suite.server = httptest.NewServer(api.NewApi(suite.ng))
	suite.client = apiclient.ApiClient{suite.server.URL, api.JsonEncoding}
}

func (suite *ApiClientTestSuite) TearDownTest(t *C) {
	os.Remove(suite.testDir)
}

func TestApiClientTestSuite(t *testing.T) {
	TestingT(t)
}
