package _import

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	insaneJSON "github.com/vitkovskii/insane-json"
	"github.com/d-ulyanov/kube-scheduler-benchmarks/pkg/client"
)

// ImportPods exports pods from given json file to kubernetes-scheduler-simulator
func ImportPods(importer *PodImporter, filePath string, logEnabled bool) error {
	contents, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	root, err := insaneJSON.DecodeBytes(contents)
	if err != nil {
		return err
	}

	items := root.Dig("items")

	for i, pod := range items.AsArray() {
		if importer.NeedSkipPod(pod) {
			if logEnabled {
				log.Printf("Skip pod: %s", pod.Dig("metadata").Dig("name").AsString())
			}
			continue
		}

		if i > 0 && i%100 == 0 {
			time.Sleep(1 * time.Second)
		}

		resp, err := importer.Import(pod)
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
				"Imported pod: %s (imported: %d, handled: %d)",
				pod.Dig("metadata").Dig("name").AsString(),
				importer.ImportedPodsCount(),
				i,
			)
		}
	}

	insaneJSON.Release(root)

	return nil
}

// PodImporter is a filter for importing pods
type PodImporter struct {
	c *client.HTTPClient

	maxPodsPerService int
	importPodsLimit   int
	importedPodsCount int

	importedPodsPerSvc map[string]int
}

// NewPodImporter returns new pod importer
func NewPodImporter(c *client.HTTPClient, limit, maxPodsPerService int) *PodImporter {
	return &PodImporter{
		c:                  c,
		importPodsLimit:    limit,
		maxPodsPerService:  maxPodsPerService,
		importedPodsPerSvc: map[string]int{},
	}
}

// NeedSkipPod decides if we want to skip the pod
func (i *PodImporter) NeedSkipPod(pod *insaneJSON.Node) bool {
	if i.importPodsLimit > 0 && i.importedPodsCount >= i.importPodsLimit {
		return true
	}

	svc := i.extractServiceName(pod)
	if svc != "" {
		if _, ok := i.importedPodsPerSvc[svc]; ok {
			if i.importedPodsPerSvc[svc] >= i.maxPodsPerService {
				return true
			}
		}
	}

	return false
}

// Import imports the pod
func (i *PodImporter) Import(pod *insaneJSON.Node) (*http.Response, error) {
	pod = i.preparePodForImport(pod)

	resp, err := i.c.ApplyPods(pod.EncodeToByte())
	if err == nil {
		i.importedPodsCount++

		svc := i.extractServiceName(pod)
		if svc != "" {
			if _, ok := i.importedPodsPerSvc[svc]; !ok {
				i.importedPodsPerSvc[svc] = 1
			} else {
				i.importedPodsPerSvc[svc]++
			}
		}
	}

	return resp, err
}

// extractServiceName returns service name
func (i *PodImporter) extractServiceName(pod *insaneJSON.Node) string {
	return pod.Dig("metadata").Dig("labels").Dig("service").AsString()
}

// preparePodForImport cleans up excess pod data for import
func (i *PodImporter) preparePodForImport(pod *insaneJSON.Node) *insaneJSON.Node {
	// remove metadata.uid for import as a new pod
	pod.Dig("metadata").Dig("uid").Suicide()
	pod.Dig("metadata").Dig("namespace").Suicide()
	pod.Dig("spec").Dig("nodeName").Suicide()

	i.deduplicateEnvVars(pod)

	return pod
}

// deduplicateEnvVars removes duplicated keys in env (there are such strange cases)
func (i *PodImporter) deduplicateEnvVars(pod *insaneJSON.Node) {
	containers := pod.Dig("spec").Dig("containers")
	for _, container := range containers.AsArray() {
		env := container.Dig("env")
		if env.IsArray() {
			envVarKeys := map[string]bool{}

		cycle:
			for _, envVar := range env.AsArray() {
				envVarName := envVar.Dig("name").AsString()

				if _, ok := envVarKeys[envVarName]; ok {
					envVar.Suicide()
					goto cycle
				}

				envVarKeys[envVarName] = true
			}
		}
	}
}

// ImportedPodsCount returns imported pods count
func (i *PodImporter) ImportedPodsCount() int {
	return i.importedPodsCount
}
