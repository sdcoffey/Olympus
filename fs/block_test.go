package fs

import (
	"github.com/google/cayley"
	"github.com/sdcoffey/olympus/env"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
)

func TestBlockWithHash(t *testing.T) {
	testInit()

	block := BlockWithHash("abcd")
	assert.Equal(t, "abcd", block.Hash)
	assert.EqualValues(t, -1, block.Offset())
}

func TestRead_readsData(t *testing.T) {
	testInit()

	os.Setenv("OLYMPUS_HOME", "test_home")
	env.InitializeEnvironment()
	defer os.RemoveAll("test_home")

	dat := RandDat(1024)
	block := BlockWithHash(blockHash(dat))

	path := filepath.Join(env.EnvPath(env.DataPath), block.Hash)
	err := ioutil.WriteFile(path, dat, 0700)
	assert.Nil(t, err)

	readDat, err := block.Read()
	assert.Nil(t, err)
	assert.Equal(t, dat, readDat)
}

func TestSave_savesBlock(t *testing.T) {
	testInit()

	block := BlockWithHash("abcd")
	block.offset = 0

	err := block.Save()
	assert.Nil(t, err)

	it := cayley.StartPath(GlobalFs().Graph, block.Hash).Out(offsetLink).BuildIterator()
	assert.True(t, cayley.RawNext(it))
	assert.Equal(t, "0", GlobalFs().Graph.NameOf(it.Result()))
}

func TestSave_throwsOnBadOffset(t *testing.T) {
	testInit()

	block := BlockWithHash("abcd")
	block.offset = 1

	err := block.Save()
	assert.NotNil(t, err)
}

func TestSave_throwsWhenNoOffsetSet(t *testing.T) {
	testInit()

	block := BlockWithHash("abcd")
	block.offset = -MEGABYTE

	err := block.Save()
	assert.NotNil(t, err)
}

func TestOffset_returnsNegativeWhenNotSet(t *testing.T) {
	testInit()

	block := BlockWithHash("abcd")
	assert.EqualValues(t, -1, block.Offset())
}

func TestOffset_returnsOffsetWhenSet(t *testing.T) {
	testInit()

	block := BlockWithHash("abcd")
	block.offset = 0
	block.Save()

	assert.EqualValues(t, 0, block.Offset())
}

func TestHash(t *testing.T) {
	testInit()

	dat := RandDat(1024 * 1024)
	fingerprint := blockHash(dat)
	assert.NotEmpty(t, fingerprint)
}

func TestHash_similarDataHashesDifferently(t *testing.T) {
	testInit()

	dat1 := RandDat(1024 * 1024)
	dat2 := make([]byte, len(dat1))
	copy(dat2, dat1)

	index := (len(dat2) / 5) * 2
	dat2[index] = byte(int(dat2[index]) + 1)

	fingerprint1 := blockHash(dat1)
	fingerprint2 := blockHash(dat2)

	assert.NotEqual(t, fingerprint1, fingerprint2)
}

func TestWrite_writesData(t *testing.T) {
	testInit()

	os.Setenv("OLYMPUS_HOME", "test_home")
	env.InitializeEnvironment()
	defer os.RemoveAll("test_home")

	dat := RandDat(MEGABYTE)
	block := BlockWithHash(blockHash(dat))

	n, err := block.Write(dat)
	assert.EqualValues(t, MEGABYTE, n)
	assert.Nil(t, err)
}

func TestWrite_throwsIfWrongHash(t *testing.T) {
	testInit()

	os.Setenv("OLYMPUS_HOME", "test_home")
	env.InitializeEnvironment()
	defer os.RemoveAll("test_home")

	dat := RandDat(MEGABYTE)
	block := BlockWithHash("abcd")

	n, err := block.Write(dat)
	assert.EqualValues(t, 0, n)
	assert.NotNil(t, err)
}

func TestWrite_throwsIfBadSize(t *testing.T) {
	testInit()

	os.Setenv("OLYMPUS_HOME", "test_home")
	env.InitializeEnvironment()
	defer os.RemoveAll("test_home")

	dat := RandDat(MEGABYTE + 1)
	block := BlockWithHash(blockHash(dat))

	n, err := block.Write(dat)
	assert.EqualValues(t, 0, n)
	assert.NotNil(t, err)
}

func BenchmarkBlockHash(b *testing.B) {
	dat := RandDat(1024 * 1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		blockHash(dat)
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
