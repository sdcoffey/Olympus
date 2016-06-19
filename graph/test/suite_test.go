package graph

import (
	"os"
	"testing"

	"github.com/sdcoffey/olympus/graph"
	. "gopkg.in/check.v1"
)

func init() {
	Suite(&GraphTestSuite{})
}

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

func TestNodeSuite(t *testing.T) {
	TestingT(t)
}
