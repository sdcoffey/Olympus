package graph

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"testing"

	"github.com/sdcoffey/olympus/Godeps/_workspace/src/github.com/stretchr/testify/assert"
	"github.com/sdcoffey/olympus/env"
	"github.com/sdcoffey/olympus/graph"
	. "gopkg.in/check.v1"
)

func TestMain(m *testing.M) {
	m.Run()
}

func (suite *GraphTestSuite) TestReader_returnsCorrectReader(t *C) {
	dat := RandDat(1024)
	blockFingerprint := graph.Hash(dat)

	path := filepath.Join(env.EnvPath(env.DataPath), blockFingerprint)
	assert.NoError(t, ioutil.WriteFile(path, []byte(dat), 0644))

	reader, err := graph.Reader(blockFingerprint)
	assert.NoError(t, err)

	readDat, err := ioutil.ReadAll(reader)
	assert.NoError(t, err)
	assert.Equal(t, dat, readDat)
}

func (suite *GraphTestSuite) TestHash(t *C) {
	dat := (RandDat(graph.MEGABYTE))
	fingerprint := graph.Hash(dat)
	assert.NotEqual(t, "", fingerprint)
}

func (suite *GraphTestSuite) TestHash_similarDataHashesDifferently(t *C) {
	dat1 := RandDat(1024 * 1024)
	dat2 := make([]byte, len(dat1))
	copy(dat2, dat1)

	index := (len(dat2) / 5) * 2
	dat2[index] = byte(int(dat2[index]) + 1)

	fingerprint1 := graph.Hash(dat1)
	fingerprint2 := graph.Hash(dat2)
	assert.NotEqual(t, fingerprint1, fingerprint2)
}

func (suite *GraphTestSuite) TestWriteData_writesData(t *C) {
	dat := RandDat(graph.MEGABYTE)
	fingerprint := graph.Hash(dat)

	n, err := graph.Write(fingerprint, dat)
	assert.EqualValues(t, graph.MEGABYTE, n)
	assert.NoError(t, err)

	reader, err := graph.Reader(fingerprint)
	assert.NoError(t, err)

	readDat, err := ioutil.ReadAll(reader)
	assert.NoError(t, err)

	assert.Equal(t, dat, readDat)
}

func (suite *GraphTestSuite) TestWrite_throwsIfWrongHash(t *C) {
	dat := RandDat(graph.MEGABYTE)
	fingerprint := "abcd"

	n, err := graph.Write(fingerprint, dat)
	assert.Error(t, err)
	assert.EqualValues(t, 0, n)
}

func (suite *GraphTestSuite) TestWrite_doesNotDuplicateDataOnDisk(t *C) {
	dat := RandDat(graph.MEGABYTE)
	hash := graph.Hash(dat)

	_, err := graph.Write(hash, dat)
	assert.NoError(t, err)

	location := graph.LocationOnDisk(hash)
	assert.NotEmpty(t, graph.LocationOnDisk(hash))
	fi, err := os.Stat(location)

	createTime := fi.ModTime()

	time.Sleep(time.Second)
	_, err = graph.Write(hash, dat)
	assert.NoError(t, err)

	fi, err = os.Stat(location)
	assert.NoError(t, err)

	assert.EqualValues(t, graph.MEGABYTE, fi.Size())
	assert.True(t, time.Now().Sub(createTime) >= time.Second)
}

func (suite *GraphTestSuite) TestWrite_throwsIfBadSize(t *C) {
	dat := RandDat(graph.MEGABYTE + 1)
	fingerprint := graph.Hash(dat)

	n, err := graph.Write(fingerprint, dat)
	assert.Equal(t, 0, n)
	assert.Error(t, err)
}

func (suite *GraphTestSuite) TestSizeOnDisk_returnsCorrectSizeForHash(t *C) {
	dat := RandDat(1024)
	fingerprint := graph.Hash(dat)

	_, err := graph.Write(fingerprint, dat)
	assert.NoError(t, err)

	size, err := graph.SizeOnDisk(fingerprint)
	assert.NoError(t, err)
	assert.EqualValues(t, 1024, size)
}

func (suite *GraphTestSuite) TestSizeOnDisk_throwsForBadFingerprint(t *C) {
	fingerprint := "abcd"
	size, err := graph.SizeOnDisk(fingerprint)

	assert.EqualValues(t, 0, size)
	assert.Error(t, err)
}

func (suite *GraphTestSuite) TestLocationOnDisk_returnsCorrectLocationForFingerprint(t *C) {
	dat := RandDat(1024)
	fingerprint := graph.Hash(dat)

	_, err := graph.Write(fingerprint, dat)
	assert.NoError(t, err)

	location := graph.LocationOnDisk(fingerprint)

	baseLocation := filepath.Dir(env.EnvPath(env.DataPath))
	expectedLocation := filepath.Join(baseLocation, filepath.Join("dat", fingerprint))
	assert.Equal(t, expectedLocation, location)
}

func (suite *GraphTestSuite) BenchmarkBlockHash(t *C) {
	dat := RandDat(graph.MEGABYTE)

	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		graph.Hash(dat)
	}
}
