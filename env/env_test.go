package env

import (
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
)

func TestMain(m *testing.M) {
	code := m.Run()
	os.Clearenv()
	os.Exit(code)
}

func TestExists_returnsTrueForExistingFile(t *testing.T) {
	file, _ := os.Create("test")
	defer os.Remove(file.Name())
	assert.True(t, exists("test"))
}

func TestExists_returnsFalseForFakeFIle(t *testing.T) {
	assert.False(t, exists("test"))
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
	assert.Equal(t, "/Users/scoffey/.olympus", oPath)
}

func TestInitializeEnvironment_createsCorrectDirctories(t *testing.T) {
	os.Setenv(envVar, "test_home")
	home, err := olympusHome()
	defer os.RemoveAll(home)
	assert.Nil(t, err)

	err = InitializeEnvironment()
	assert.Nil(t, err)

	assert.True(t, exists(filepath.Join(home, "dat")))
	assert.True(t, exists(filepath.Join(home, "db")))
	assert.True(t, exists(filepath.Join(home, "cfg")))
	assert.True(t, exists(filepath.Join(home, "log")))
}

func wd() string {
	wd, _ := os.Getwd()
	wd, _ = filepath.Abs(wd)
	return wd
}
