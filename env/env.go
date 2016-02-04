package env

import (
	"os"
	"os/user"
	"path/filepath"
)

type PathMap string

const (
	envVar             = "OLYMPUS_HOME"
	DataPath   PathMap = "dat"
	LogPath    PathMap = "log"
	DbPath     PathMap = "db"
	ConfigPath PathMap = "cfg"
)

func InitializeEnvironment() (err error) {
	var home string
	if home, err = olympusHome(); err != nil {
		return
	}

	mkdir := func(dirpath string) {
		if err != nil || Exists(dirpath) {
			return
		}
		err = os.Mkdir(dirpath, 0744)
		if err != nil {
			panic(err)
		}
	}

	mkdir(home)
	mkdir(filepath.Join(home, string(DataPath)))
	mkdir(filepath.Join(home, string(LogPath)))
	mkdir(filepath.Join(home, string(DbPath)))
	mkdir(filepath.Join(home, string(ConfigPath)))

	return
}

func EnvPath(pathmap PathMap) (home string) {
	var err error
	if home, err = olympusHome(); err != nil {
		return ""
	}
	switch pathmap {
	case DataPath:
		return filepath.Join(home, string(DataPath))
	case LogPath:
		return filepath.Join(home, string(LogPath))
	case DbPath:
		return filepath.Join(home, string(DbPath))
	case ConfigPath:
		return filepath.Join(home, string(ConfigPath))
	default:
		return ""
	}
}

func olympusHome() (path string, err error) {
	if path = os.Getenv(envVar); path != "" {
		return filepath.Abs(path)
	}
	var currentUser *user.User
	if currentUser, err = user.Current(); err != nil {
		return
	} else {
		path = filepath.Join(currentUser.HomeDir, ".olympus")
		return
	}
}

func Exists(filename string) bool {
	if _, err := os.Stat(filename); err != nil {
		return !os.IsNotExist(err)
	}
	return true
}
