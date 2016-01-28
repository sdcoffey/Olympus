package fs

import (
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/sdcoffey/olympus/env"
	"github.com/stretchr/testify/assert"
)

func TestReader_returnsCorrectReader(t *testing.T) {
	testInit()

	os.Setenv("OLYMPUS_HOME", "test_home")
	env.InitializeEnvironment()
	defer os.RemoveAll("test_home")

	dat := RandDat(1024)
	blockFingerprint := Hash(dat)

	path := filepath.Join(env.EnvPath(env.DataPath), blockFingerprint)
	err := ioutil.WriteFile(path, []byte(dat), 0644)
	assert.NoError(t, err)

	reader, err := Reader(blockFingerprint)
	assert.NoError(t, err)

	readDat, err := ioutil.ReadAll(reader)
	assert.NoError(t, err)

	assert.Equal(t, []byte(dat), readDat)
	assert.EqualValues(t, len(dat), len(readDat))
}

func TestHash(t *testing.T) {
	testInit()

	dat := (RandDat(MEGABYTE))
	fingerprint := Hash(dat)
	assert.NotEmpty(t, fingerprint)
}

func TestHash_similarDataHashesDifferently(t *testing.T) {
	testInit()

	dat1 := RandDat(1024 * 1024)
	dat2 := make([]byte, len(dat1))
	copy(dat2, dat1)

	index := (len(dat2) / 5) * 2
	dat2[index] = byte(int(dat2[index]) + 1)

	fingerprint1 := Hash(dat1)
	fingerprint2 := Hash(dat2)

	assert.NotEqual(t, fingerprint1, fingerprint2)
}

func TestWriteData_writesData(t *testing.T) {
	testInit()

	os.Setenv("OLYMPUS_HOME", "test_home")
	env.InitializeEnvironment()
	defer os.RemoveAll("test_home")

	dat := RandDat(MEGABYTE)
	fingerprint := Hash(dat)

	n, err := Write(fingerprint, dat)
	assert.EqualValues(t, MEGABYTE, n)
	assert.NoError(t, err)

	reader, err := Reader(fingerprint)
	assert.NoError(t, err)

	readDat, err := ioutil.ReadAll(reader)
	assert.NoError(t, err)

	assert.Equal(t, dat, readDat)
}

func TestWrite_throwsIfWrongHash(t *testing.T) {
	testInit()

	os.Setenv("OLYMPUS_HOME", "test_home")
	env.InitializeEnvironment()
	defer os.RemoveAll("test_home")

	dat := RandDat(MEGABYTE)
	fingerprint := "abcd"

	n, err := Write(fingerprint, dat)
	assert.Error(t, err)
	assert.EqualValues(t, 0, n)
}

func TestWrite_throwsIfBadSize(t *testing.T) {
	testInit()

	os.Setenv("OLYMPUS_HOME", "test_home")
	env.InitializeEnvironment()
	defer os.RemoveAll("test_home")

	dat := RandDat(MEGABYTE + 1)
	fingerprint := Hash(dat)

	n, err := Write(fingerprint, dat)
	assert.Equal(t, 0, n)
	assert.Error(t, err)
}

func BenchmarkBlockHash(b *testing.B) {
	dat := RandDat(MEGABYTE)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Hash(dat)
	}
}

var randGen *rand.Rand = rand.New(rand.NewSource(34))

func RandDat(size int) []byte {
	dat := make([]byte, size)
	for i := 0; i < size; i++ {
		dat[i] = byte(randGen.Intn(255))
	}

	return dat
}
