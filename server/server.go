package main

import (
	"github.com/google/cayley"
	"github.com/google/cayley/graph"
	_ "github.com/google/cayley/graph/bolt"
	"github.com/gorilla/mux"
	"github.com/sdcoffey/olympus/env"
	"github.com/sdcoffey/olympus/fs"
	"github.com/sdcoffey/olympus/server/api"
	"net/http"
	"os"
	"path/filepath"
)

var debug = false

func main() {
	env.InitializeEnvironment()
	err := initDb()
	if err != nil {
		println(err.Error())
		os.Exit(1)
	} else {
		fs.RootNode()
	}

	r := mux.NewRouter()
	v1Router := r.PathPrefix("/v1").Subrouter()
	v1Router.HandleFunc("/ls/{parentId}", api.LsFiles).Methods("GET")
	v1Router.HandleFunc("/mv/{fileId}/{newParentId}", api.MvFile).Methods("PATCH")
	v1Router.HandleFunc("/rm/{fileId}", api.RmFile).Methods("DELETE")
	v1Router.HandleFunc("/mkdir/{parentId}/{name}", api.MkDir).Methods("POST")
	v1Router.HandleFunc("/cr/{parentId}/{name}", api.Cr).Methods("POST")
	v1Router.HandleFunc("/update/{fileId}", api.Update).Methods("PATCH")
	v1Router.HandleFunc("/hasBlocks/{fileId}", api.HasBlocks).Methods("GET")
	v1Router.HandleFunc("/dd/{fileId}/{blockHash}/{offset}", api.WriteBlock).Methods("POST")

	http.ListenAndServe(":3000", r)
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

	fs.Init(handle)
	return
}
