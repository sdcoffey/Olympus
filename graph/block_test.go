package graph

import (
	"io/ioutil"
	"math/rand"
	"path/filepath"
	"testing"

	"github.com/sdcoffey/olympus/Godeps/_workspace/src/github.com/stretchr/testify/assert"
	"github.com/sdcoffey/olympus/env"
)

func TestReader_returnsCorrectReader(t *testing.T) {
	testInit()

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

	dat := RandDat(MEGABYTE)
	fingerprint := "abcd"

	n, err := Write(fingerprint, dat)
	assert.Error(t, err)
	assert.EqualValues(t, 0, n)
}

func TestWrite_throwsIfBadSize(t *testing.T) {
	testInit()

	dat := RandDat(MEGABYTE + 1)
	fingerprint := Hash(dat)

	n, err := Write(fingerprint, dat)
	assert.Equal(t, 0, n)
	assert.Error(t, err)
}

func TestSizeOnDisk_returnsCorrectSizeForHash(t *testing.T) {
	testInit()

	dat := RandDat(1024)
	fingerprint := Hash(dat)

	_, err := Write(fingerprint, dat)
	assert.NoError(t, err)

	size, err := SizeOnDisk(fingerprint)
	assert.NoError(t, err)
	assert.EqualValues(t, 1024, size)
}

func TestSizeOnDisk_throwsForBadFingerprint(t *testing.T) {
	testInit()

	fingerprint := "abcd"
	size, err := SizeOnDisk(fingerprint)
	assert.EqualValues(t, 0, size)
	assert.Error(t, err)
}

func TestLocationOnDisk_returnsCorrectLocationForFingerprint(t *testing.T) {
	testInit()

	dat := RandDat(1024)
	fingerprint := Hash(dat)

	_, err := Write(fingerprint, dat)
	assert.NoError(t, err)

	location := LocationOnDisk(fingerprint)

	baseLocation := filepath.Dir(env.EnvPath(env.DataPath))
	expectedLocation := filepath.Join(baseLocation, filepath.Join("dat", fingerprint))
	assert.Equal(t, expectedLocation, location)
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
