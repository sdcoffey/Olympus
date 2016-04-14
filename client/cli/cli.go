package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sdcoffey/olympus/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/sdcoffey/olympus/Godeps/_workspace/src/github.com/google/cayley"
	"github.com/sdcoffey/olympus/Godeps/_workspace/src/github.com/wsxiaoys/terminal/color"
	"github.com/sdcoffey/olympus/client/apiclient"
	"github.com/sdcoffey/olympus/client/shared"
	"github.com/sdcoffey/olympus/graph"
	"github.com/sdcoffey/olympus/peer"
	"gopkg.in/cheggaaa/pb.v1"
)

var (
	manager *shared.Manager
	model   *shared.Model
)

func main() {
	handle := initDb()

	var address string
	flag.StringVar(&address, "address", "", "Olympus address (CLI will listen for local servers if none given")
	flag.Parse()

	if address == "" {
		println("Searching for Olympus instances")

		if olympusAddress, err := peer.FindServer(time.Second * 5); err != nil {
			color.Println("@rCould not find Olympus Instance on network: " + err.Error())
		} else {
			address = olympusAddress.String()
			color.Println("@gFound Olympus At:", olympusAddress.String())
		}
	}

	if !strings.HasPrefix(address, "http") {
		address = "http://" + address
	}
	client := apiclient.ApiClient{Address: address}

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

	file, _ := os.Stat(target)
	bar := pb.StartNew(int(file.Size()))
	bar.SetUnits(pb.U_BYTES)

	updateCallback := func(total, progress int64) {
		bar.Set(int(progress))
	}

	if _, err := manager.UploadFile(model.Root.Id, target, updateCallback); err != nil {
		color.Println(fmt.Sprintf("@rError uploading %s: %s", target, err.Error()))
	} else {
		bar.FinishPrint("Finished Uploading")
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
