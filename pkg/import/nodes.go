package _import

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	insaneJSON "github.com/vitkovskii/insane-json"
	"github.com/d-ulyanov/kube-scheduler-benchmarks/pkg/client"
)

// ImportNodes exports nodes from given json file to kubernetes-scheduler-simulator
func ImportNodes(importer *NodeImporter, filePath string, logEnabled bool) error {
	contents, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	root, err := insaneJSON.DecodeBytes(contents)
	if err != nil {
		return err
	}

	items := root.Dig("items")

	for i, item := range items.AsArray() {
		if importer.NeedSkipNode(item) {
			if logEnabled {
				log.Printf("Skip node: %s", item.Dig("metadata").Dig("name").AsString())
			}
			continue
		}

		resp, err := importer.Import(item)
		if err != nil {
			return err
		}
		b, _ := ioutil.ReadAll(resp.Body)
		if resp.StatusCode >= http.StatusMultipleChoices {
			return fmt.Errorf("got scheduler response status: %d: %s", resp.StatusCode, b)
		}

		err = resp.Body.Close()
		if err != nil {
			return err
		}

		if logEnabled {
			log.Printf(
				"Imported node: %s (imported: %d, handled: %d)",
				item.Dig("metadata").Dig("name").AsString(),
				importer.ImportedNodesCount(),
				i,
			)
		}
	}

	insaneJSON.Release(root)

	return nil
}

// NewNodeImporter returns new node importer
func NewNodeImporter(c *client.HTTPClient, limit, skipNodeWithCoresNotEq int) *NodeImporter {
	return &NodeImporter{
		c:                      c,
		importNodesLimit:       limit,
		skipNodeWithCoresNotEq: skipNodeWithCoresNotEq,
	}
}

// NodeImporter is a filter for importing nodes
type NodeImporter struct {
	c *client.HTTPClient

	skipNodeWithCoresNotEq int
	importNodesLimit       int
	importedNodesCount     int
}

// NeedSkipNode decides if we want to skip the node
func (i *NodeImporter) NeedSkipNode(node *insaneJSON.Node) bool {
	if i.importNodesLimit > 0 && i.importedNodesCount >= i.importNodesLimit {
		return true
	}

	if i.skipNodeWithCoresNotEq != 0 && i.skipNodeWithCoresNotEq != node.Dig("status").Dig("allocatable").Dig("cpu").AsInt() {
		return true
	}

	return false
}

func (i *NodeImporter) prepareNode(node *insaneJSON.Node) *insaneJSON.Node {
	// remove metadata.uid for import as a new node
	node.Dig("metadata").Dig("uid").Suicide()

	// remove status.images to avoid image locality scoring
	node.Dig("status").Dig("images").Suicide()

	return node
}

// Import imports node
func (i *NodeImporter) Import(node *insaneJSON.Node) (*http.Response, error) {
	node = i.prepareNode(node)

	resp, err := i.c.ApplyNodes(node.EncodeToByte())
	if err == nil {
		i.importedNodesCount++
	}

	return resp, err
}

// ImportedNodesCount returns imported nodes count
func (i *NodeImporter) ImportedNodesCount() int {
	return i.importedNodesCount
}
