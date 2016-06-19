package env

import (
	"os"
	"os/user"
	"path/filepath"

	"github.com/sdcoffey/olympus/Godeps/_workspace/src/github.com/stretchr/testify/assert"
	"github.com/sdcoffey/olympus/env"
	. "gopkg.in/check.v1"
)

func (suite *EnvironmentTestSuite) TestExists_returnsTrueForExistingFile(t *C) {
	file, _ := os.Create("test")
	defer os.Remove(file.Name())
	assert.True(t, env.Exists("test"))
}

func (suite *EnvironmentTestSuite) TestExists_returnsFalseForFakeFIle(t *C) {
	assert.False(t, env.Exists("test"))
}

func (suite *EnvironmentTestSuite) TestEnv(t *C) {
	os.Setenv("OLYMPUS_HOME", suite.testPath)

	assert.Equal(t, filepath.Join(suite.testPath, "dat"), env.EnvPath(env.DataPath))
	assert.Equal(t, filepath.Join(suite.testPath, "db"), env.EnvPath(env.DbPath))
	assert.Equal(t, filepath.Join(suite.testPath, "cfg"), env.EnvPath(env.ConfigPath))
	assert.Equal(t, filepath.Join(suite.testPath, "log"), env.EnvPath(env.LogPath))
}

func (suite *EnvironmentTestSuite) TestOlympusHome_returnsVarFromEnvWhenSet(t *C) {
	os.Setenv("OLYMPUS_HOME", suite.testPath)
	assert.Equal(t, filepath.Join(suite.testPath, "dat"), env.EnvPath(env.DataPath))
}

func (suite *EnvironmentTestSuite) TestOlympusHome_returnsDefaultPathWhenEnvNotSet(t *C) {
	os.Clearenv()
	user, _ := user.Current()
	assert.Equal(t, filepath.Join(user.HomeDir, ".olympus", "dat"), env.EnvPath(env.DataPath))
}

func (suite *EnvironmentTestSuite) TestInitializeEnvironment_createsCorrectDirctories(t *C) {
	os.Setenv("OLYMPUS_HOME", suite.testPath)
	err := env.InitializeEnvironment()
	assert.Nil(t, err)

	assert.True(t, env.Exists(filepath.Join(suite.testPath, "dat")))
	assert.True(t, env.Exists(filepath.Join(suite.testPath, "db")))
	assert.True(t, env.Exists(filepath.Join(suite.testPath, "cfg")))
	assert.True(t, env.Exists(filepath.Join(suite.testPath, "log")))
}
