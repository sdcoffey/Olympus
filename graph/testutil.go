package graph

import (
	"io/ioutil"
	"math/rand"
	"os"

	"github.com/sdcoffey/olympus/Godeps/_workspace/src/github.com/google/cayley"
	"github.com/sdcoffey/olympus/env"
)

func TestInit() *NodeGraph {
	if dir, err := ioutil.TempDir(os.TempDir(), ".olympus"); err != nil {
		panic(err)
	} else {
		os.Setenv("OLYMPUS_HOME", dir)
		if err = env.InitializeEnvironment(); err != nil {
			panic(err)
		}
		handle, _ := cayley.NewMemoryGraph()
		if ng, err := NewGraph(handle); err != nil {
			panic(err)
		} else {
			return ng
		}
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
