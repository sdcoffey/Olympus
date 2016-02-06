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
	"github.com/sdcoffey/olympus/peer"
	"github.com/wsxiaoys/terminal/color"
)

var manager *shared.OManager

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

	manager = shared.NewManager(client, handle)
	if err := manager.Init(); err != nil {
		panic(err)
	}

	app := cli.NewApp()
	app.HelpName = "Olympus"
	app.Commands = []cli.Command{
		{
			Name:   "ls",
			Usage:  "List files in current directory",
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
			Usage:  "Remove file",
			Action: rm,
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
	model := manager.Model
	if err := model.Refresh(); err != nil {
		color.Println("@r", err.Error())
	} else {
		for _, file := range model.Root.Children() {
			if c.Bool("l") {
				fmt.Println(file.String())
			} else {
				name := file.Name()
				var col string
				if file.IsDir() {
					col = "@b"
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
	if dirname == ".." {
		if manager.Model.Root.Parent() == nil {
			return
		} else if err := manager.ChangeDirectory(manager.Model.Root.Parent().Id); err != nil {
			color.Println("@r", err.Error())
		}
	} else if file := manager.Model.FindFileByName(dirname); file == nil {
		color.Println("@rNo such file: ", dirname)
	} else if err := manager.ChangeDirectory(file.Id); err != nil {
		color.Println("@r", err.Error())
	}
}

func mkdir(c *cli.Context) {
	var name string
	if len(c.Args()) == 0 {
		color.Println("@rNot enough arguments in call to mkdir")
	}
	name = c.Args()[0]

	if err := manager.CreateDirectory(manager.Model.Root.Id, name); err != nil {
		color.Println("@r", err.Error())
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

	if file := manager.Model.FindFileByName(victim); file == nil {
		color.Println("@rNo such file: ", victim)
	} else if err := manager.RemoveFile(file.Id); err != nil {
		color.Println("@r", err.Error())
	}
}

func workingDirectory() string {
	here := manager.Model.Root
	var path string
	if here.Parent() != nil {
		path = here.Name()
	}
	for ; here.Parent() != nil && here.Parent().Parent() != nil; here = here.Parent() {
		path = here.Parent().Name() + "/" + path
	}

	return "/" + path
}
