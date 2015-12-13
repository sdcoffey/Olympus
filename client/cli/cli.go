package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/google/cayley"
	"github.com/sdcoffey/olympus/client/apiclient"
	"github.com/sdcoffey/olympus/fs"
	"os"
	"strings"
)

type Command interface {
	Execute(apiclient.ApiClient) (string, error)
}

var wd *fs.OFile

func main() {
	fs.Init(initDb())
	if root, err := fs.RootNode(); err != nil {
		println("Error creating memory graph: " + err.Error())
	} else {
		wd = root
	}

	println("Searching for Olympus instances")
	var olympusAddress string
	var client apiclient.ApiClient
	if olympusAddress = findOlympus(); olympusAddress == "" {
		println("Could not find Olympus Instance on network")
		os.Exit(1)
	} else {
		client = apiclient.ApiClient{Address: olympusAddress}
	}

	print(">>> ")
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		if cmd, err := parseCommand(scanner.Bytes()); err != nil {
			println(err.Error())
		} else {
			if resp, err := cmd.Execute(client); err != nil {
				println(err.Error())
			} else if resp != "" {
				println(resp)
			}
		}
		print(">>> ")
	}
}

func findOlympus() string {
	return "http://localhost:3000"
}

func parseCommand(command []byte) (Command, error) {
	strCmd := strings.TrimSpace(string(command))
	var args []string
	if strings.Contains(strCmd, " ") {
		if commandComponents := strings.Split(strCmd, " "); len(commandComponents) > 1 {
			args = commandComponents[1:len(commandComponents)]
			strCmd = commandComponents[0]
		}
	}

	function := strings.ToLower(strCmd)
	switch function {
	case "ls":
		return ls{args}, nil
	case "mkdir":
		if len(args) < 1 {
			return nil, errors.New("Not enough arguments in call to mkdir")
		}
		return mkdir{args[0]}, nil
	case "pwd":
		return pwd{}, nil
	case "cd":
		if len(args) < 1 {
			return nil, errors.New("Not enough arguments in call to cd")
		}
		return cd{args[0]}, nil
	default:
		return nil, errors.New(fmt.Sprint("Unrecognized command: ", function))
	}
}

func initDb() *cayley.Handle {
	graph, err := cayley.NewMemoryGraph()
	if err != nil {
		panic(err)
	}
	return graph
}

type ls struct {
	args []string
}

func (cmd ls) Execute(client apiclient.ApiClient) (string, error) {
	// todo lh
	if fis, err := client.Ls(wd.Id); err != nil {
		return "", err
	} else {
		var response bytes.Buffer
		for _, fi := range fis {
			file := fs.FileWithFileInfo(fi)
			response.WriteString(fi.Name)
			response.WriteString("    ")
			file.Save()
		}

		return response.String(), nil
	}
}

type mkdir struct {
	name string
}

func (cmd mkdir) Execute(client apiclient.ApiClient) (string, error) {
	// todo -p
	if _, err := client.Mkdir(wd.Id, cmd.name); err != nil {
		return "", err
	} else {
		return "", nil
	}
}

type cd struct {
	dirname string
}

func (cmd cd) Execute(client apiclient.ApiClient) (string, error) {
	// todo path/to/file
	if cmd.dirname == ".." {
		if wd.Parent() != nil {
			wd = wd.Parent()
		}
		return "", nil
	} else if file := fs.FileWithName(wd.Id, cmd.dirname); file == nil || !file.Exists() {
		return "", errors.New("No such folder " + cmd.dirname)
	} else if !file.IsDir() {
		return "", errors.New(cmd.dirname + " not a directory")
	} else {
		wd = file
		return "", nil
	}
}

type pwd struct{}

func (cmd pwd) Execute(client apiclient.ApiClient) (string, error) {
	here := wd
	path := wd.Name()
	for ; here.Parent() != nil && here.Parent().Parent() != nil; here = here.Parent() {
		path = here.Parent().Name() + "/" + path
	}

	path = "/" + path

	return path, nil
}
