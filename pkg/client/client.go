package client

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"time"
)

// HTTPClient is a scheduler simulator http client
type HTTPClient struct {
	c              *http.Client
	baseURLPattern string
}

// New returns new client
func New(host string, port int) *HTTPClient {
	return &HTTPClient{
		c: &http.Client{
			Timeout: 5 * time.Minute,
		},
		baseURLPattern: fmt.Sprintf("http://%s:%d/api/v1/", host, port) + "%s",
	}
}

// ApplyNodes push nodes to scheduler simulator
func (c *HTTPClient) ApplyNodes(nodesJSON []byte) (*http.Response, error) {
	req, err := http.NewRequest("POST", c.methodURL("nodes"), bytes.NewReader(nodesJSON))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	return c.c.Do(req)
}

// ApplyPods push pods to scheduler simulator
func (c *HTTPClient) ApplyPods(podJSON []byte) (*http.Response, error) {
	req, err := http.NewRequest("POST", c.methodURL("pods"), bytes.NewReader(podJSON))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	return c.DoWithRetry(req, 30, 3*time.Second)
}

// ApplyConfig push config to scheduler simulator
func (c *HTTPClient) ApplyConfig(cfg []byte) (*http.Response, error) {
	req, err := http.NewRequest("POST", c.methodURL("schedulerconfiguration"), bytes.NewReader(cfg))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	return c.c.Do(req)
}

// Reset fully resets scheduler simulator state
func (c *HTTPClient) Reset() (*http.Response, error) {
	req, err := http.NewRequest("PUT", c.methodURL("reset"), nil)
	if err != nil {
		return nil, err
	}

	return c.DoWithRetry(req, 10, 2*time.Second)
}

// ListPods returns all pods in cluster
func (c *HTTPClient) ListPods() (*http.Response, error) {
	req, err := http.NewRequest("GET", c.methodURL("pods"), nil)
	if err != nil {
		return nil, err
	}

	return c.c.Do(req)
}

// ListNodes returns all nodes in cluster
func (c *HTTPClient) ListNodes() (*http.Response, error) {
	req, err := http.NewRequest("GET", c.methodURL("nodes"), nil)
	if err != nil {
		return nil, err
	}

	return c.c.Do(req)
}

func (c *HTTPClient) methodURL(method string) string {
	return fmt.Sprintf(c.baseURLPattern, method)
}

// DoWithRetry retries to send request N attempts
func (c *HTTPClient) DoWithRetry(req *http.Request, attempts int, sleep time.Duration) (*http.Response, error) {
	var err error
	var resp *http.Response

	for i := 0; i < attempts; i++ {
		if i > 0 {
			status := 0
			if resp != nil {
				status = resp.StatusCode
			}

			log.Printf("retrying in 2s after error: %s, code: %d", err, status)
			time.Sleep(sleep)
			sleep *= 2
		}
		resp, err = c.c.Do(req)

		if err != nil {
			continue
		}

		if resp.StatusCode < http.StatusInternalServerError && err == nil {
			return resp, nil
		}
	}
	return nil, fmt.Errorf("after %d attempts, last error: %s", attempts, err)
}
