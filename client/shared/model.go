package shared

import (
	"errors"
	"fmt"

	"github.com/sdcoffey/olympus/client/apiclient"
	"github.com/sdcoffey/olympus/graph"
)

type Model struct {
	graph *graph.NodeGraph
	Root  *graph.Node
	api   apiclient.OlympusClient
}

func newModel(api apiclient.OlympusClient, rootNode *graph.Node, ng *graph.NodeGraph) *Model {
	return &Model{
		Root:  rootNode,
		api:   api,
		graph: ng,
	}
}

func (model *Model) init() error {
	if !model.Root.Exists() {
		return errors.New(fmt.Sprintf("Root with id: %s does not exist", model.Root.Id))
	}

	if err := model.Refresh(); err != nil {
		return err
	} else {
		return nil
	}
}

func (model *Model) Refresh() error {
	nodeSet := make(map[string]bool)
	for _, nodeOnDisk := range model.Root.Children() {
		nodeSet[nodeOnDisk.Id] = true
	}

	if nodeInfos, err := model.api.ListNodes(model.Root.Id); err != nil {
		return fmt.Errorf("Error listing nodes: %s", err.Error())
	} else {
		var err error
		for i := 0; i < len(nodeInfos) && err == nil; i++ {
			nodeInfo := nodeInfos[i]
			node := model.graph.NodeWithNodeInfo(nodeInfo)
			if err = node.Save(); err != nil {
				return fmt.Errorf("Error refreshing model: %s", err.Error())
			}
			if _, ok := nodeSet[nodeInfo.Id]; ok {
				delete(nodeSet, nodeInfo.Id)
			}
		}

		if err != nil {
			return err
		}
	}

	for id, _ := range nodeSet {
		staleNode := model.graph.NodeWithId(id)
		if err := model.graph.RemoveNode(staleNode); err != nil {
			return err
		}
	}

	return nil
}

func (model *Model) FindNodeByName(name string) *graph.Node {
	if node := model.graph.NodeWithName(model.Root.Id, name); node == nil {
		model.Refresh()
		node = model.graph.NodeWithName(model.Root.Id, name)
		return node
	} else {
		return node
	}
}
