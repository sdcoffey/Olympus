package env

import (
	"io/ioutil"
	"os"
	"testing"

	. "github.com/sdcoffey/olympus/Godeps/_workspace/src/gopkg.in/check.v1"
)

func init() {
	Suite(&EnvironmentTestSuite{})
}

type EnvironmentTestSuite struct {
	testPath string
}

func (suite *EnvironmentTestSuite) SetUpTest(t *C) {
	if dir, err := ioutil.TempDir(os.TempDir(), ".olympus"); err != nil {
		panic(err)
	} else {
		suite.testPath = dir
	}
}

func (suite *EnvironmentTestSuite) TearDowntest(t *C) {
	os.Clearenv()
}

func TestEnvironmentSuite(t *testing.T) {
	TestingT(t)
}
