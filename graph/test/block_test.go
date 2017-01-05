package graph

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/sdcoffey/olympus/env"
	"github.com/sdcoffey/olympus/graph"
	"github.com/sdcoffey/olympus/graph/testutils"
	. "gopkg.in/check.v1"
)

func (suite *GraphTestSuite) TestReader_returnsCorrectReader(t *C) {
	dat := testutils.RandDat(1024)
	blockFingerprint := graph.Hash(dat)

	path := filepath.Join(env.EnvPath(env.DataPath), blockFingerprint)
	t.Check(ioutil.WriteFile(path, []byte(dat), 0644), IsNil)

	reader, err := graph.Reader(blockFingerprint)
	t.Check(err, IsNil)

	readDat, err := ioutil.ReadAll(reader)
	t.Check(err, IsNil)
	t.Check(dat, DeepEquals, readDat)
}

func (suite *GraphTestSuite) TestHash(t *C) {
	dat := (testutils.RandDat(graph.MEGABYTE))
	fingerprint := graph.Hash(dat)
	t.Check(fingerprint, Not(Equals), "")
}

func (suite *GraphTestSuite) TestHash_similarDataHashesDifferently(t *C) {
	dat1 := testutils.RandDat(1024 * 1024)
	dat2 := make([]byte, len(dat1))
	copy(dat2, dat1)

	index := (len(dat2) / 5) * 2
	dat2[index] = byte(int(dat2[index]) + 1)

	fingerprint1 := graph.Hash(dat1)
	fingerprint2 := graph.Hash(dat2)
	t.Check(fingerprint1, Not(Equals), fingerprint2)
}

func (suite *GraphTestSuite) TestWriteData_writesData(t *C) {
	dat := testutils.RandDat(graph.MEGABYTE)
	fingerprint := graph.Hash(dat)

	n, err := graph.Write(fingerprint, dat)
	t.Check(graph.MEGABYTE, Equals, n)
	t.Check(err, IsNil)

	reader, err := graph.Reader(fingerprint)
	t.Check(err, IsNil)

	readDat, err := ioutil.ReadAll(reader)
	t.Check(err, IsNil)

	t.Check(dat, DeepEquals, readDat)
}

func (suite *GraphTestSuite) TestWrite_throwsIfWrongHash(t *C) {
	dat := testutils.RandDat(graph.MEGABYTE)
	fingerprint := "abcd"

	n, err := graph.Write(fingerprint, dat)
	t.Check(err, ErrorMatches, "Data hash does not match this block's hash")
	t.Check(0, Equals, n)
}

func (suite *GraphTestSuite) TestWrite_doesNotDuplicateDataOnDisk(t *C) {
	dat := testutils.RandDat(graph.MEGABYTE)
	hash := graph.Hash(dat)

	_, err := graph.Write(hash, dat)
	t.Check(err, IsNil)

	location := graph.LocationOnDisk(hash)
	t.Check(graph.LocationOnDisk(hash), Not(Equals), "")
	fi, err := os.Stat(location)

	createTime := fi.ModTime()

	time.Sleep(time.Second)
	_, err = graph.Write(hash, dat)
	t.Check(err, IsNil)

	fi, err = os.Stat(location)
	t.Check(err, IsNil)

	t.Check(fi.Size(), Equals, int64(graph.MEGABYTE))
	t.Check(time.Now().UTC().Sub(createTime) >= time.Second, Equals, true)
}

func (suite *GraphTestSuite) TestWrite_throwsIfBadSize(t *C) {
	dat := testutils.RandDat(graph.MEGABYTE + 1)
	fingerprint := graph.Hash(dat)

	n, err := graph.Write(fingerprint, dat)
	t.Check(n, Equals, 0)
	t.Check(err, ErrorMatches, "Data length exceeds max block size")
}

func (suite *GraphTestSuite) TestSizeOnDisk_returnsCorrectSizeForHash(t *C) {
	dat := testutils.RandDat(1024)
	fingerprint := graph.Hash(dat)

	_, err := graph.Write(fingerprint, dat)
	t.Check(err, IsNil)

	size, err := graph.SizeOnDisk(fingerprint)
	t.Check(err, IsNil)
	t.Check(size, Equals, int64(1024))
}

func (suite *GraphTestSuite) TestSizeOnDisk_throwsForBadFingerprint(t *C) {
	fingerprint := "abcd"
	size, err := graph.SizeOnDisk(fingerprint)

	t.Check(size, Equals, int64(0))
	t.Check(err, Not(IsNil))
}

func (suite *GraphTestSuite) TestLocationOnDisk_returnsCorrectLocationForFingerprint(t *C) {
	dat := testutils.RandDat(1024)
	fingerprint := graph.Hash(dat)

	_, err := graph.Write(fingerprint, dat)
	t.Check(err, IsNil)

	location := graph.LocationOnDisk(fingerprint)

	baseLocation := filepath.Dir(env.EnvPath(env.DataPath))
	expectedLocation := filepath.Join(baseLocation, filepath.Join("dat", fingerprint))
	t.Check(location, Equals, expectedLocation)
}

func (suite *GraphTestSuite) BenchmarkBlockHash(t *C) {
	dat := testutils.RandDat(graph.MEGABYTE)

	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		graph.Hash(dat)
	}
}
