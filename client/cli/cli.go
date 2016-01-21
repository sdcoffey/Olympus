package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/google/cayley"
	"github.com/sdcoffey/olympus/client/apiclient"
	"github.com/sdcoffey/olympus/client/shared"
	"github.com/sdcoffey/olympus/peer"
	"os"
	"strings"
)

type Command func(string, []string, *shared.OManager) (string, error)

func main() {
	handle := initDb()

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

	manager := shared.NewManager(client, handle)
	if err := manager.Init(); err != nil {
		panic(err)
	}

	print("O> ")
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		stringCommand := strings.TrimSpace(string(scanner.Bytes()))
		var args []string
		var flags string
		if strings.Contains(stringCommand, " ") {
			if commandComponents := strings.Split(stringCommand, " "); len(commandComponents) > 1 {
				args = commandComponents[1:len(commandComponents)]
				flags = parseFlags(&args)
				stringCommand = commandComponents[0]
			}
		}
		if command := parseCommand(stringCommand); command != nil {
			if result, err := command(flags, args, manager); err != nil {
				println(err.Error())
			} else if result != "" {
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

func ls(flags string, args []string, manager *shared.OManager) (string, error) {
	model := manager.Model
	if err := model.Refresh(); err != nil {
		return "", err
	} else {
		var response bytes.Buffer
		for _, file := range model.Root.Children() {
			if strings.Contains(flags, "l") {
				response.WriteString(file.String())
				response.WriteString("\n")
			} else {
				response.WriteString(file.Name())
				response.WriteString("    ")
			}
		}
		return response.String(), nil
	}
}

func mkdir(flags string, args []string, manager *shared.OManager) (string, error) {
	var name string
	if len(args) == 0 {
		return "", errors.New("Not enough arguments in call to mkdir")
	}
	name = args[0]

	return "", manager.CreateDirectory(manager.Model.Root.Id, name)
}

func cd(flags string, args []string, manager *shared.OManager) (string, error) {
	if len(args) < 1 {
		return "", errors.New("Not enough arguments in call to cd")
	}

	dirname := args[0]
	if dirname == ".." {
		if manager.Model.Root.Parent() == nil {
			return "", nil
		}
		return "", manager.ChangeDirectory(manager.Model.Root.Parent().Id)
	} else if file := manager.Model.FindFileByName(dirname); file == nil {
		return "", errors.New(fmt.Sprintf("No such file: %s", dirname))
	} else {
		return "", manager.ChangeDirectory(file.Id)
	}
}

func pwd(flags string, args []string, manager *shared.OManager) (string, error) {
	here := manager.Model.Root
	var path string
	if here.Parent() != nil {
		path = here.Name()
	}
	for ; here.Parent() != nil && here.Parent().Parent() != nil; here = here.Parent() {
		path = here.Parent().Name() + "/" + path
	}

	path = "/" + path

	return path, nil
}

func parseFlags(rawArgs *[]string) string {
	filteredArgs := make([]string, 0)
	buf := bytes.NewBufferString("")
	for _, arg := range *rawArgs {
		arg = strings.TrimSpace(arg)
		if strings.HasPrefix(arg, "-") && len(arg) > 1 {
			for i := 1; i < len(arg); i++ {
				buf.WriteByte(arg[i])
			}
		} else {
			filteredArgs = append(filteredArgs, arg)
		}
	}

	*rawArgs = filteredArgs
	return buf.String()
}
