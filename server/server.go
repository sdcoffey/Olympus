package main

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/cayleygraph/cayley"
	cgraph "github.com/cayleygraph/cayley/graph"
	_ "github.com/cayleygraph/cayley/graph/bolt"
	"github.com/sdcoffey/olympus/env"
	"github.com/sdcoffey/olympus/graph"
	"github.com/sdcoffey/olympus/peer"
	"github.com/sdcoffey/olympus/server/api"
	"github.com/wsxiaoys/terminal/color"
)

var debug = false

func main() {
	env.InitializeEnvironment()
	if nodeGraph, err := initDb(); err != nil {
		color.Println("@r", err)
		os.Exit(1)
	} else {
		go peer.ClientHeartbeat()
		http.ListenAndServe(":3000", api.NewApi(nodeGraph))
	}
}

func initDb() (*graph.NodeGraph, error) {
	var handle *cayley.Handle
	var err error
	if !debug {
		dbPath := filepath.Join(env.EnvPath(env.DbPath), "db.dat")
		if !env.Exists(dbPath) {
			if err = cgraph.InitQuadStore("bolt", dbPath, nil); err != nil {
				return nil, err
			}
		}
		if handle, err = cayley.NewGraph("bolt", dbPath, nil); err != nil {
			return nil, err
		}
	} else {
		if handle, err = cayley.NewMemoryGraph(); err != nil {
			return nil, err
		}
	}

	return graph.NewGraph(handle)
}
