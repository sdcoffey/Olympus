package main

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var logfile *os.File
var logger *log.Logger

func init() {
	logfile, _ = os.OpenFile("./log.l", os.O_CREATE|os.O_TRUNC|os.O_RDWR, os.FileMode(0755))
	logger = log.New(logfile, "", 0)
}

func FsCompelter(line string) (strs []string) {
	p := strings.Split(line, " ")
	if len(p) > 1 {
		path := p[1]
		comps := strings.Split(path, string(filepath.Separator))
		path = filepath.Join(comps[:len(comps)-1]...)

		if strings.HasPrefix(p[1], string(filepath.Separator)) {
			path = "/" + path + "/"
		} else if path == "" {
			path = "./" + path
		}
		logger.Println(path)

		infos, err := ioutil.ReadDir(path)
		if err != nil {
			panic(err)
		}

		logger.Println(infos)

		for _, info := range infos {
			strs = append(strs, info.Name())
		}
	}

	logger.Println(strs)
	return strs
}
