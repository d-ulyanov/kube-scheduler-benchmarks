package _import

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/d-ulyanov/kube-scheduler-benchmarks/pkg/client"
)

// ImportConfig imports scheduler configuration from json file to kubernetes-scheduler-simulator
func ImportConfig(c *client.HTTPClient, filePath string) error {
	cfg, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	resp, err := c.ApplyConfig(cfg)
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

	return nil
}
