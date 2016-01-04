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

type Command func(string, []string, *shared.Model) (string, error)

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

	model := shared.NewModel(client, handle)
	if err := model.Init(); err != nil {
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
			if result, err := command(flags, args, model); err != nil {
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

func ls(flags string, args []string, model *shared.Model) (string, error) {
	if err := model.Refresh(); err != nil {
		return "", err
	} else {
		var response bytes.Buffer
		for i := 0; i < model.Count(); i++ {
			response.WriteString(model.At(i).Name())
			if strings.Contains(flags, "l") {
				response.WriteString("\n")
			} else {
				response.WriteString("    ")
			}
		}
		return response.String(), nil
	}
}

func mkdir(flags string, args []string, model *shared.Model) (string, error) {
	newName := args[0]
	return "", model.CreateDirectory(newName)
}

func cd(flags string, args []string, model *shared.Model) (string, error) {
	if len(args) < 1 {
		return "", errors.New("Not enough arguments in call to cd")
	}
	// todo path/to/file

	dirname := args[0]
	if file := model.FindFileByName(dirname); file == nil {
		return "", errors.New(fmt.Sprintf("No such file: %s", dirname))
	} else {
		return "", model.MoveToNode(file.Id)
	}
}

func pwd(flags string, args []string, model *shared.Model) (string, error) {
	here := model.Root()
	path := here.Name()
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
