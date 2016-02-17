package env

import (
	"os"
	"os/user"
	"path/filepath"
	"testing"

	"github.com/sdcoffey/olympus/Godeps/_workspace/src/github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	code := m.Run()
	os.Clearenv()
	os.Exit(code)
}

func TestExists_returnsTrueForExistingFile(t *testing.T) {
	file, _ := os.Create("test")
	defer os.Remove(file.Name())
	assert.True(t, Exists("test"))
}

func TestExists_returnsFalseForFakeFIle(t *testing.T) {
	assert.False(t, Exists("test"))
}

func TestEnv(t *testing.T) {
	os.Setenv(envVar, "test_home")

	assert.Equal(t, filepath.Join(wd(), "test_home/dat"), EnvPath(DataPath))
	assert.Equal(t, filepath.Join(wd(), "test_home/db"), EnvPath(DbPath))
	assert.Equal(t, filepath.Join(wd(), "test_home/cfg"), EnvPath(ConfigPath))
	assert.Equal(t, filepath.Join(wd(), "test_home/log"), EnvPath(LogPath))
}

func TestOlympusHome_returnsVarFromEnvWhenSet(t *testing.T) {
	os.Setenv(envVar, "test_home")

	oPath, err := olympusHome()
	assert.Nil(t, err)
	assert.Equal(t, filepath.Join(wd(), "test_home"), oPath)
}

func TestOlympusHome_returnsDefaultPathWhenEnvNotSet(t *testing.T) {
	os.Clearenv()
	oPath, err := olympusHome()
	assert.Nil(t, err)
	user, _ := user.Current()
	assert.Equal(t, user.HomeDir+"/.olympus", oPath)
}

func TestInitializeEnvironment_createsCorrectDirctories(t *testing.T) {
	os.Setenv(envVar, "test_home")
	home, err := olympusHome()
	defer os.RemoveAll(home)
	assert.Nil(t, err)

	err = InitializeEnvironment()
	assert.Nil(t, err)

	assert.True(t, Exists(filepath.Join(home, "dat")))
	assert.True(t, Exists(filepath.Join(home, "db")))
	assert.True(t, Exists(filepath.Join(home, "cfg")))
	assert.True(t, Exists(filepath.Join(home, "log")))
}

func wd() string {
	wd, _ := os.Getwd()
	wd, _ = filepath.Abs(wd)
	return wd
}
