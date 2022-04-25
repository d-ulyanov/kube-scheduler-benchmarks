package _import

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/d-ulyanov/kube-scheduler-benchmarks/pkg/client"
)

// ResetExportState fully resets kube-scheduler-simulator state
func ResetExportState(c *client.HTTPClient) error {
	resp, err := c.Reset()
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
