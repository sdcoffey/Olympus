package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cayleygraph/cayley"
	"github.com/chzyer/readline"
	"github.com/codegangsta/cli"
	"github.com/sdcoffey/olympus/client/apiclient"
	"github.com/sdcoffey/olympus/client/shared"
	"github.com/sdcoffey/olympus/graph"
	"github.com/sdcoffey/olympus/peer"
	"github.com/sdcoffey/olympus/server/api"
	"github.com/wsxiaoys/terminal/color"
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
			address = olympusAddress.String() + ":3000"
			color.Println("@gFound Olympus At:", olympusAddress.String())
		}
	}

	if !strings.HasPrefix(address, "http") {
		address = "http://" + address
	}

	client := apiclient.ApiClient{Address: address, Encoding: api.JsonEncoding}

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
			Name:   "mv",
			Usage:  "Move or rename file",
			Action: mv,
		},
	}

	config := &readline.Config{
		Prompt:          color.Sprintf("@g O @y(%s) $ ", workingDirectory()),
		HistoryFile:     "/tmp/readline.tmp",
		InterruptPrompt: "^C",
		AutoComplete: readline.NewPrefixCompleter(
			readline.PcItem("put", readline.PcItemDynamic(FsCompelter)),
		),
		EOFPrompt: "exit",
	}

	l, err := readline.NewEx(config)
	if err != nil {
		panic(err)
	}

	defer l.Close()

	for {
		line, err := l.Readline()
		if err == readline.ErrInterrupt {
			if len(line) == 0 {
				break
			} else {
				continue
			}
		} else if err == io.EOF {
			break
		}

		line = strings.TrimSpace(line)
		args := strings.Split(strings.TrimSpace(line), " ")
		args = append([]string{"olympus"}, args...) // This is a hack :(
		app.Run(args)
		config.Prompt = color.Sprintf("@g O @y(%s) $ ", workingDirectory())
	}
}

func initDb() *cayley.Handle {
	g, err := cayley.NewMemoryGraph()
	if err != nil {
		panic(err)
	}
	return g
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

	model.Refresh()
}

func put(c *cli.Context) {
	if len(c.Args()) < 1 {
		color.Println("@yNot enough arguments in call to rm")
		return
	}
	target := c.Args()[0]

	file, err := os.Stat(target)
	if err != nil {
		color.Println("@rError uploading file -> ", err.Error())
	} else {
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
}

func mv(c *cli.Context) {
	if len(c.Args()) < 2 {
		color.Println("@yNot enough arguments in call to mv")
		return
	}

	target := c.Args()[0]
	destination := c.Args()[1]

	if !filepath.IsAbs(target) {
		target = filepath.Join(workingDirectory(), target)
	}

	if !filepath.IsAbs(destination) {
		destination = filepath.Join(workingDirectory(), destination)
	}

	var targetNode, destinationNode *graph.Node
	var tErr, dErr error

	targetNode, tErr = manager.FindNodeByPath(target)
	destinationNode, dErr = manager.FindNodeByPath(destination)

	if tErr != nil || dErr != nil {
		color.Println("@rError resolving path => ", tErr.Error(), dErr.Error())
		return
	}

	newFilename := filepath.Base(target)
	if destinationNode == nil {
		destinationNode, dErr = manager.FindNodeByPath(filepath.Dir(destination))
		if dErr != nil {
			color.Println("@rError resolving path => ", tErr.Error(), dErr.Error())
			return
		}
		newFilename = filepath.Base(destination)
	}

	err := manager.MoveNode(targetNode.Id, destinationNode.Id, newFilename)
	if err != nil {
		color.Println("@rError moving node => ", err.Error())
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
