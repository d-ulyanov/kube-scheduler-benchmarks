package _import

import (
	"fmt"
	"io/ioutil"

	insaneJSON "github.com/vitkovskii/insane-json"
)

// CutPods cuts pods from file up to limit and writes rest pods to file with suffix
func CutPods(filePath string, limit int) error {
	contents, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	root, err := insaneJSON.DecodeBytes(contents)
	if err != nil {
		return err
	}

start:
	for i, pod := range root.Dig("items").AsArray() {
		if i >= limit {
			pod.Suicide()
			goto start
		}
	}

	err = ioutil.WriteFile(fmt.Sprintf("%s-%d", filePath, limit), root.EncodeToByte(), 0644)
	if err != nil {
		return err
	}

	return nil
}
