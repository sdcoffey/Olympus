package ds

import (
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
)

func TestHash(t *testing.T) {
	dat := RandDat(1024 * 1024)
	fingerprint := blockHash(dat)
	assert.NotEmpty(t, fingerprint)
}

func TestHash_similarDataHashesDifferently(t *testing.T) {
	dat1 := RandDat(1024 * 1024)
	dat2 := make([]byte, len(dat1))
	copy(dat2, dat1)

	index := (len(dat2) / 5) * 2
	dat2[index] = byte(int(dat2[index]) + 1)

	fingerprint1 := blockHash(dat1)
	fingerprint2 := blockHash(dat2)

	assert.NotEqual(t, fingerprint1, fingerprint2)
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
