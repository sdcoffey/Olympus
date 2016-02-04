package main

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/google/cayley"
	"github.com/google/cayley/graph"
	_ "github.com/google/cayley/graph/bolt"
	"github.com/sdcoffey/olympus/env"
	"github.com/sdcoffey/olympus/fs"
	"github.com/sdcoffey/olympus/peer"
	"github.com/sdcoffey/olympus/server/api"
)

var debug = false

func main() {
	env.InitializeEnvironment()
	if err := initDb(); err != nil {
		println(err.Error())
		os.Exit(1)
	} else {
		fs.RootNode()
	}

	go peer.ClientHeartbeat()
	http.ListenAndServe(":3000", api.Router())
}

func initDb() (err error) {
	var handle *cayley.Handle
	if !debug {
		dbPath := filepath.Join(env.EnvPath(env.DbPath), "db.dat")
		if !env.Exists(dbPath) {
			if err = graph.InitQuadStore("bolt", dbPath, nil); err != nil {
				return
			}
		}
		if handle, err = cayley.NewGraph("bolt", dbPath, nil); err != nil {
			return
		}
	} else {
		if handle, err = cayley.NewMemoryGraph(); err != nil {
			return
		}
	}

	return fs.Init(handle)
}
