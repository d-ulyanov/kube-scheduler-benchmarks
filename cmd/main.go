package main

import (
	"fmt"
	"github.com/d-ulyanov/kube-scheduler-benchmarks/pkg/client"
	"github.com/d-ulyanov/kube-scheduler-benchmarks/pkg/cluster"
	"github.com/d-ulyanov/kube-scheduler-benchmarks/pkg/import"
	"log"
	"os"
	"time"
)

var availableCommands = "Available commands: \n\t- import-nodes\n\t- import-pods\n\t- list-nodes\n\t- reset"

func main() {
	args := os.Args[1:]

	if len(args) > 0 {
		runCommand(args[0])
	} else {
		log.Println(availableCommands)
	}
}

func runCommand(cmd string) {
	var err error

	c := client.New("127.0.0.1", 1212)

	switch cmd {
	case "all":
		configs := []string{
			"./testdata/config_default.json",
			"./testdata/config_all_leastalloc_cpu5.json",
			"./testdata/config_all_leastalloc_cpu100.json",
			"./testdata/config_leastalloc_cpu5.json",
			"./testdata/config_leastalloc_cpu100.json",
		}

		nodesFilePath := "./testdata/nodes.json"
		podsFilePath := "./testdata/pods.json"
		nodesLimit := 5
		podsLimit := 300
		iterationsPerConfig := 5

		log.Printf("Run modelling for nodes: %d, pods: %d, iterations per config: %d...\n", nodesLimit, podsLimit, iterationsPerConfig)

		for i := range configs {
			for j := 0; j < iterationsPerConfig; j++ {
				log.Printf("Config: %s, iteration: %d...\n", configs[i], j+1)
				if err = _import.ResetExportState(c); err != nil {
					log.Fatal("error:", err)
				}
				//log.Printf("Importing config %s...\n", configs[i])
				if err = _import.ImportConfig(c, configs[i]); err != nil {
					log.Fatal("error:", err)
				}

				//log.Println("Config imported - OK")
				//log.Println("Importing nodes...")
				if err = _import.ImportNodes(_import.NewNodeImporter(c, nodesLimit, 88), nodesFilePath, false); err != nil {
					log.Fatal("error:", err)
				}
				//log.Println("Nodes imported - OK")
				//log.Println("Importing pods...")
				if err = _import.ImportPods(_import.NewPodImporter(c, podsLimit, 3), podsFilePath, false); err != nil {
					log.Fatal("error:", err)
				}
				//log.Println("Pods imported - OK")
				time.Sleep(3 * time.Second)

				if err = cluster.ListNodes(c); err != nil {
					log.Fatal("error:", err)
				}
			}
		}
	case "import-nodes":
		nodeImporter := _import.NewNodeImporter(c, 50, 88)
		err = _import.ImportNodes(nodeImporter, "./testdata/nodes.json", true)
	case "cut-pods":
		err = _import.CutPods("./testdata/pods.json", 5000)
	case "import-pods":
		podImporter := _import.NewPodImporter(c, 4000, 10)
		err = _import.ImportPods(podImporter, "./testdata/pods.json", true)
	case "import-config":
		err = _import.ImportConfig(c, "./testdata/config_default.json")
	case "list-nodes":
		err = cluster.ListNodes(c)
	case "reset":
		err = _import.ResetExportState(c)
	default:
		err = fmt.Errorf("command %s not implemented. %s", cmd, availableCommands)
	}

	if err != nil {
		log.Fatal("error:", err)
		return
	}

	log.Printf("Command `%s` finished successfully", cmd)
}
