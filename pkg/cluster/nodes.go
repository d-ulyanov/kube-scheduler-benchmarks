package cluster

import (
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"

	"github.com/olekukonko/tablewriter"
	insaneJSON "github.com/vitkovskii/insane-json"
	"github.com/d-ulyanov/kube-scheduler-benchmarks/pkg/client"
	"gonum.org/v1/gonum/stat"
)

type Node struct {
	Name             string
	AllocatableCores float64
	AllocatableMemGb float64
	AllocatedCores   float64
	AllocatedMemGb   float64
	AllocatedPods    int
	Pods             []string
}

// ListNodes lists cluster nodes from kubernetes-scheduler-simulator with advanced analytics
func ListNodes(c *client.HTTPClient) error {
	nodes, err := listNodes(c)
	if err != nil {
		return err
	}

	resp, err := c.ListPods()
	if err != nil {
		return err
	}
	if resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("got scheduler response status: %d", resp.StatusCode)
	}

	b, _ := ioutil.ReadAll(resp.Body)

	err = resp.Body.Close()
	if err != nil {
		return err
	}

	root, err := insaneJSON.DecodeBytes(b)
	if err != nil {
		return err
	}

	pods := root.Dig("items")

	//log.Printf("found pods: %d\n", len(pods.AsArray()))
	//log.Printf("found nodes: %d\n", len(nodes))

	prefillNodesWithPods(nodes, pods)

	insaneJSON.Release(root)

	minCPU, maxCPU := calculateCPUImbalance(nodes)
	minMem, maxMem := calculateMemImbalance(nodes)
	stddevCPU := stat.PopStdDev(nodesCPUAllocatedArray(nodes), nil)
	stddevMem := stat.PopStdDev(nodesMemAllocatedGbArray(nodes), nil)

	log.Printf("Imbalance CPU: %.2f, min=%.2f, max=%.2f, stddev=%.2f\n", maxCPU-minCPU, minCPU, maxCPU, stddevCPU)
	log.Printf("Imbalance Mem: %.2f, min=%.2fGb, max=%.2fGb, stddev=%.2f\n", maxMem-minMem, minMem, maxMem, stddevMem)

	return nodesChart(nodes)
}

func listNodes(c *client.HTTPClient) (map[string]*Node, error) {
	resp, err := c.ListNodes()
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("got scheduler response status: %d", resp.StatusCode)
	}

	b, _ := ioutil.ReadAll(resp.Body)

	err = resp.Body.Close()
	if err != nil {
		return nil, err
	}

	root, err := insaneJSON.DecodeBytes(b)
	if err != nil {
		return nil, err
	}

	nodes := root.Dig("items").AsArray()

	nodesList := map[string]*Node{}

	for _, node := range nodes {
		nodeMemGb, err := memToGb(node.Dig("status").Dig("allocatable").Dig("memory").AsString())
		if err != nil {
			log.Printf("Can't parse node mem allocatable: %s, error: %s\n", node.Dig("status").Dig("allocatable").Dig("memory").AsString(), err)
		}

		nodesList[node.Dig("metadata").Dig("name").AsString()] = &Node{
			Name:             node.Dig("metadata").Dig("name").AsString(),
			AllocatableCores: node.Dig("status").Dig("allocatable").Dig("cpu").AsFloat(),
			AllocatableMemGb: nodeMemGb,
			Pods:             []string{},
		}
	}

	root = nil
	insaneJSON.Release(root)

	return nodesList, nil
}

func prefillNodesWithPods(nodes map[string]*Node, pods *insaneJSON.Node) {
	for _, pod := range pods.AsArray() {
		nodeName := pod.Dig("spec").Dig("nodeName").AsString()

		n, ok := nodes[nodeName]
		if !ok {
			log.Printf("Unknown node: %s, pod: %s\n", nodeName, pod.Dig("metadata").Dig("name").AsString())
			continue
		}

		n.Pods = append(n.Pods, pod.Dig("metadata").Dig("name").AsString())

		for _, container := range pod.Dig("spec").Dig("containers").AsArray() {
			cpu, err := cpuToCores(container.Dig("resources").Dig("requests").Dig("cpu").AsString())
			if err != nil {
				log.Printf("Can't parse CPU requests: %s, error: %s\n", (container.Dig("resources").Dig("requests").Dig("cpu").AsString()), err)
			}

			n.AllocatedCores += cpu

			mem, err := memToGb(container.Dig("resources").Dig("requests").Dig("memory").AsString())
			if err != nil {
				log.Printf("Can't parse memory requests: %s, error: %s\n", container.Dig("resources").Dig("requests").Dig("memory").AsString(), err)
			}

			n.AllocatedMemGb += mem
		}
	}
}

func nodesChart(nodes map[string]*Node) error {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Node", "Allocatable CPU", "Allocated CPU", "Allocatable Mem, Gb", "Allocated Mem, Gb"})

	headColor := tablewriter.Colors{tablewriter.Bold, tablewriter.BgGreenColor}
	table.SetHeaderColor(headColor, headColor, headColor, headColor, headColor)

	colColor := tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiBlackColor}
	table.SetColumnColor(colColor, colColor, colColor, colColor, colColor)

	data := [][]string{}

	for _, node := range nodes {
		data = append(data, []string{
			node.Name,
			fmt.Sprintf("%.1f", node.AllocatableCores),
			fmt.Sprintf("%.2f", node.AllocatedCores),
			fmt.Sprintf("%.2f", node.AllocatableMemGb),
			fmt.Sprintf("%.2f", node.AllocatedMemGb)},
		)
	}

	table.AppendBulk(data)
	table.Render()

	return nil
}

func cpuToCores(v string) (float64, error) {
	if v[len(v)-1:] == "m" {
		cpuMillis, err := strconv.ParseFloat(v[:len(v)-1], 10)
		if err != nil {
			return 0, err
		}

		cpuCores := cpuMillis / 1000

		return float64(cpuCores), nil
	}

	cpuCores, err := strconv.ParseFloat(v, 10)
	if err != nil {
		return 0, err
	}
	return cpuCores, nil
}

func memToGb(v string) (float64, error) {
	memStr := v
	multiplicator := float64(1)

	switch v[len(v)-2:] {
	case "Ki":
		memStr = v[:len(v)-2]
		multiplicator = math.Pow(2, 30) / math.Pow(10, 3)
	case "Mi":
		memStr = v[:len(v)-2]
		multiplicator = math.Pow(2, 30) / math.Pow(10, 6)
	case "Gi":
		memStr = v[:len(v)-2]
		multiplicator = math.Pow(2, 30) / math.Pow(10, 9)
	}

	switch v[len(v)-1:] {
	case "K":
		memStr = v[:len(v)-1]
		multiplicator = math.Pow(2, 20)
	case "M":
		memStr = v[:len(v)-1]
		multiplicator = math.Pow(2, 10)
	case "G":
		memStr = v[:len(v)-1]
		multiplicator = 1
	}

	mem, err := strconv.ParseFloat(memStr, 10)
	if err != nil {
		return 0, err
	}

	memGb := mem / multiplicator

	return float64(memGb), nil
}

func calculateCPUImbalance(nodes map[string]*Node) (lowerBound, higherBound float64) {
	var isInited bool

	for _, node := range nodes {
		if !isInited || lowerBound > node.AllocatedCores {
			lowerBound = node.AllocatedCores
			isInited = true
		}
		if higherBound < node.AllocatedCores {
			higherBound = node.AllocatedCores
		}
	}

	return
}

func calculateMemImbalance(nodes map[string]*Node) (lowerBound, higherBound float64) {
	var isInited bool

	for _, node := range nodes {
		if !isInited || lowerBound > node.AllocatedMemGb {
			lowerBound = node.AllocatedMemGb
			isInited = true
		}
		if higherBound < node.AllocatedMemGb {
			higherBound = node.AllocatedMemGb
		}
	}

	return
}

func nodesCPUAllocatedArray(nodes map[string]*Node) []float64 {
	out := []float64{}
	for _, node := range nodes {
		out = append(out, node.AllocatedCores)
	}

	return out
}

func nodesMemAllocatedGbArray(nodes map[string]*Node) []float64 {
	out := []float64{}
	for _, node := range nodes {
		out = append(out, node.AllocatedMemGb)
	}

	return out
}
