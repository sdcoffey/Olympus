package graph

import (
	"testing"

	"os"

	"github.com/sdcoffey/olympus/graph"
	. "gopkg.in/check.v1"
)

type GraphTestSuite struct {
	ng      *graph.NodeGraph
	testDir string
}

func (suite *GraphTestSuite) SetUpTest(t *C) {
	suite.ng, suite.testDir = TestInit()
}

func (suite *GraphTestSuite) TearDownTest(t *C) {
	os.Remove(suite.testDir)
}

func init() {
	Suite(&GraphTestSuite{})
}

func TestNodeSuite(t *testing.T) {
	TestingT(t)
}
