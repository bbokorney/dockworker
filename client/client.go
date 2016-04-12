package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/bbokorney/dockworker"
)

// Client represents a client which can be used to
// interact with Dockworker
type Client interface {
	BaseURL() string
	CreateJob(job dockworker.Job) (dockworker.Job, error)
	GetJob(ID dockworker.JobID) (dockworker.Job, error)
	StopJob(ID dockworker.JobID) error
	GetLogs(ID dockworker.JobID) ([]byte, error)
}

// TODO: Move into dockworker package so the imports make more sense

// NewClient returns a new Client
func NewClient(baseURL string) Client {
	return client{
		baseURL: baseURL,
	}
}

type client struct {
	baseURL string
}

func (c client) BaseURL() string {
	return c.baseURL
}

func (c client) CreateJob(job dockworker.Job) (dockworker.Job, error) {
	body, err := json.Marshal(job)
	if err != nil {
		return dockworker.Job{}, nil
	}

	resp, err := http.Post(fmt.Sprintf("%s/jobs", c.baseURL), "application/json", bytes.NewReader(body))
	if err != nil {
		return dockworker.Job{}, err
	}

	if resp.StatusCode != http.StatusCreated {
		return dockworker.Job{}, fmt.Errorf("Expected code %d but received %d", http.StatusCreated, resp.StatusCode)
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return dockworker.Job{}, err
	}

	createdJob := &dockworker.Job{}
	err = json.Unmarshal(respBody, createdJob)
	if err != nil {
		return dockworker.Job{}, err
	}
	return *createdJob, nil
}

func (c client) GetLogs(ID dockworker.JobID) ([]byte, error) {
	resp, err := http.Get(fmt.Sprintf("%s/jobs/%d/logs", c.baseURL, ID))
	if err != nil {
		return []byte{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return []byte{}, fmt.Errorf("Expected code %d but received %d", http.StatusCreated, resp.StatusCode)
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, err
	}
	return respBody, err
}

func (c client) StopJob(ID dockworker.JobID) error {
	resp, err := http.Post(fmt.Sprintf("%s/%d/stop", c.baseURL, ID), "application/json", nil)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("Expected code %d but received %d", http.StatusCreated, resp.StatusCode)
	}
	return nil
}

func (c client) GetJob(ID dockworker.JobID) (dockworker.Job, error) {
	// TODO: implement
	return dockworker.Job{}, nil
}
