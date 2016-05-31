package graph

import (
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"

	"github.com/sdcoffey/olympus/Godeps/_workspace/src/github.com/google/cayley"
	cgraph "github.com/sdcoffey/olympus/Godeps/_workspace/src/github.com/google/cayley/graph"
	_ "github.com/sdcoffey/olympus/Godeps/_workspace/src/github.com/google/cayley/graph/bolt"
	"github.com/sdcoffey/olympus/env"
	"github.com/sdcoffey/olympus/graph"
)

func TestInit() (*graph.NodeGraph, string) {
	if dir, err := ioutil.TempDir(os.TempDir(), ".olympus"); err != nil {
		panic(err)
	} else {
		os.Setenv("OLYMPUS_HOME", dir)
		if err = env.InitializeEnvironment(); err != nil {
			panic(err)
		}

		dbPath := filepath.Join(env.EnvPath(env.DbPath), "db.dat")
		if !env.Exists(dbPath) {
			cgraph.InitQuadStore("bolt", dbPath, nil)
			if handle, err := cayley.NewGraph("bolt", dbPath, nil); err != nil {
				panic(err)
			} else if ng, err := graph.NewGraph(handle); err != nil {
				panic(err)
			} else {
				return ng, dir
			}
		} else {
			return nil, ""
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
