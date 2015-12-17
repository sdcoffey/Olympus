package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/google/cayley"
	"github.com/sdcoffey/olympus/client/apiclient"
	"github.com/sdcoffey/olympus/fs"
	"github.com/sdcoffey/olympus/peer"
	"os"
	"strings"
	"flag"
)

type Command func([]string, apiclient.ApiClient) (string, error)

var wd *fs.OFile

func main() {
	fs.Init(initDb())
	if root, err := fs.RootNode(); err != nil {
		println("Error creating memory graph: " + err.Error())
	} else {
		wd = root
	}

	println("Searching for Olympus instances")
	var client apiclient.ApiClient
	if olympusAddress, err := peer.FindServer(); err != nil {
		println("Could not find Olympus Instance on network: " + err.Error())
		os.Exit(1)
	} else {
		resolvedPath := "http://" + olympusAddress.String() + ":3000"
		println("Found Olympus At: " + olympusAddress.String())
		client = apiclient.ApiClient{Address: resolvedPath}
	}

	print("O> ")
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		stringCommand := strings.TrimSpace(string(scanner.Bytes()))
		var args []string
		if strings.Contains(stringCommand, " ") {
			if commandComponents := strings.Split(stringCommand, " "); len(commandComponents) > 1 {
				args = commandComponents[1:len(commandComponents)]
				stringCommand = commandComponents[0]
			}
		}
		if command := parseCommand(stringCommand); command != nil {
			if result, err := command(args, client); err != nil {
				println(fmt.Sprint("Error: ", err.Error()))
			} else {
				println(result)
			}
		} else {
			println("Unrecognized command: " + stringCommand)
		}
		print("O> ")
	}
}

func parseCommand(command string) Command {
	strCmd := strings.TrimSpace(string(command))

	function := strings.ToLower(strCmd)
	switch function {
	case "ls":
		return ls
	case "mkdir":
		return mkdir
	case "pwd":
		return pwd
	case "cd":
		return cd
	default:
		return nil
	}
}

func initDb() *cayley.Handle {
	graph, err := cayley.NewMemoryGraph()
	if err != nil {
		panic(err)
	}
	return graph
}

func ls(args []string, client apiclient.ApiClient) (string, error) {
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

func mkdir(args []string, client apiclient.ApiClient) (string, error) {
	var dirsToCreate []string
	if len(args) == 0 {
		return "", errors.New("Not enough arguments in call to mkdir")
	} else if len(args) > 1 || strings.Contains(args[0], "/"){
		parser := flag.NewFlagSet("ls", flag.ContinueOnError)
		p := parser.Bool("p", false, "Create with parents")
		parser.Parse(args)
		dirIdx := 1
		if p != nil && !*p {
			dirIdx = 0
		}
		dirsToCreate = strings.Split(args[dirIdx], "/")
	} else if len(args) == 1 {
		dirsToCreate = []string{args[0]}
	}

	lastParentId := wd.Id
	var err error
	for i := 0; i < len(dirsToCreate) && err == nil; i++ {
		dirToCreate := dirsToCreate[i]
		lastParentId, err = client.Mkdir(lastParentId, dirToCreate)
	}
	if err != nil {
		return "", err
	} else {
		return "", nil
	}
}

func cd(args []string, client apiclient.ApiClient) (string, error) {
	if len(args) < 2 {
		return "", errors.New("Not enough arguments in call to cd")
	}
	// todo path/to/file

	dirname := args[0]
	if dirname == ".." {
		if wd.Parent() != nil {
			wd = wd.Parent()
		}
		return "", nil
	} else if file := fs.FileWithName(wd.Id, dirname); file == nil || !file.Exists() {
		return "", errors.New("No such folder " + dirname)
	} else if !file.IsDir() {
		return "", errors.New(dirname + " not a directory")
	} else {
		wd = file
		return "", nil
	}
}

func pwd(args []string, client apiclient.ApiClient) (string, error) {
	here := wd
	path := wd.Name()
	for ; here.Parent() != nil && here.Parent().Parent() != nil; here = here.Parent() {
		path = here.Parent().Name() + "/" + path
	}

	path = "/" + path

	return path, nil
}
