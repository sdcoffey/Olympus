package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/codegangsta/cli"
	"github.com/google/cayley"
	"github.com/sdcoffey/olympus/client/apiclient"
	"github.com/sdcoffey/olympus/client/shared"
	"github.com/sdcoffey/olympus/graph"
	"github.com/sdcoffey/olympus/peer"
	"github.com/wsxiaoys/terminal/color"
)

var (
	manager *shared.Manager
	model   *shared.Model
)

func main() {
	handle := initDb()

	println("Searching for Olympus instances")
	var client apiclient.ApiClient
	if olympusAddress, err := peer.FindServer(time.Second * 5); err != nil {
		color.Println("@rCould not find Olympus Instance on network: " + err.Error())
		os.Exit(1)
	} else {
		resolvedPath := "http://" + olympusAddress.String() + ":3000"
		color.Println("@gFound Olympus At:", olympusAddress.String())
		client = apiclient.ApiClient{Address: resolvedPath}
	}

	var err error
	manager = shared.NewManager(client, handle)
	if model, err = manager.Model(graph.RootNodeId); err != nil {
		panic(err)
	}

	app := cli.NewApp()
	app.HelpName = "Olympus"
	app.Commands = []cli.Command{
		{
			Name:   "ls",
			Usage:  "List nodes in current directory",
			Action: ls,
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "l",
					Usage: "Prints each object on a new line",
				},
			},
		},
		{
			Name:   "cd",
			Usage:  "Change directory",
			Action: cd,
		},
		{
			Name:   "mkdir",
			Usage:  "Create directory in current path",
			Action: mkdir,
		},
		{
			Name:   "pwd",
			Usage:  "Print the current directory",
			Action: pwd,
		},
		{
			Name:   "rm",
			Usage:  "Remove node",
			Action: rm,
		},
		{
			Name:   "put",
			Usage:  "Upload file",
			Action: put,
		},
		{
			Name:  "exit",
			Usage: "Exit Olympus Cli",
			Action: func(c *cli.Context) {
				os.Exit(0)
			},
		},
	}

	linePrefix := func() {
		color.Print("@gOLYMPUS")
		color.Printf("@y (%s)", workingDirectory())
		fmt.Print(" $ ")
	}

	linePrefix()
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		args := strings.Split(strings.TrimSpace(scanner.Text()), " ")
		args = append([]string{"olympus"}, args...) // This is a hack :(
		app.Run(args)
		linePrefix()
	}
}

func initDb() *cayley.Handle {
	graph, err := cayley.NewMemoryGraph()
	if err != nil {
		panic(err)
	}
	return graph
}

func ls(c *cli.Context) {
	if err := model.Refresh(); err != nil {
		color.Println("@r", err.Error())
	} else {
		for _, node := range model.Root.Children() {
			if c.Bool("l") {
				fmt.Println(node.String())
			} else {
				name := node.Name()
				var col string
				if node.IsDir() {
					name += "/"
				}
				color.Print(col, name, "    ")
			}
		}
		if !c.Bool("l") {
			fmt.Println()
		}
	}
}

func cd(c *cli.Context) {
	if len(c.Args()) < 1 {
		color.Println("@yNot enough arguments in call to cd")
		return
	}

	dirname := c.Args()[0]
	var err error
	if dirname == ".." {
		if model.Root.Parent() == nil {
			return
		} else if model, err = manager.Model(model.Root.Parent().Id); err != nil {
			panic(err)
		}
	} else if node := model.FindNodeByName(dirname); node == nil {
		color.Println("@rNo such node: ", dirname)
	} else if model, err = manager.Model(node.Id); err != nil {
		color.Println("@r", err.Error())
	}
}

func mkdir(c *cli.Context) {
	var name string
	if len(c.Args()) == 0 {
		color.Println("@rNot enough arguments in call to mkdir")
	}
	name = c.Args()[0]

	if err := manager.CreateDirectory(model.Root.Id, name); err != nil {
		color.Println("@r", err.Error())
	} else {
		model.Refresh()
	}
}

func pwd(c *cli.Context) {
	fmt.Println(workingDirectory())
}

func rm(c *cli.Context) {
	if len(c.Args()) < 1 {
		color.Println("@yNot enough arguments in call to rm")
		return
	}
	victim := c.Args()[0]

	if node := model.FindNodeByName(victim); node == nil {
		color.Println("@rNo such node: ", victim)
	} else if err := manager.RemoveNode(node.Id); err != nil {
		color.Println("@r", err.Error())
	}
}

func put(c *cli.Context) {
	if len(c.Args()) < 1 {
		color.Println("@yNot enough arguments in call to rm")
		return
	}
	target := c.Args()[0]
	if _, err := manager.UploadFile(model.Root.Id, target); err != nil {
		color.Println(fmt.Sprintf("@rError uploading %s: %s", target, err.Error()))
	} else {
		model.Refresh()
	}
}

func workingDirectory() string {
	here := model.Root
	var path string
	if here.Parent() != nil {
		path = here.Name()
	}
	for ; here.Parent() != nil && here.Parent().Parent() != nil; here = here.Parent() {
		path = here.Parent().Name() + "/" + path
	}

	return "/" + path
}
