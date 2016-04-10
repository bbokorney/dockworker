package client

import (
	"io"
	"strings"

	"github.com/bbokorney/dockworker"
)

// Client represents a client which can be used to
// interact with Dockworker
type Client interface {
	CreateJob(job dockworker.Job) (dockworker.Job, error)
	GetJob(ID dockworker.JobID) (dockworker.Job, error)
	Logs(ID dockworker.JobID) (io.Reader, error)
}

// NewClient returns a new Client
func NewClient(baseURL string) Client {
	return client{
		baseURL: baseURL,
	}
}

type client struct {
	baseURL string
}

func (c client) CreateJob(job dockworker.Job) (dockworker.Job, error) {
	return dockworker.Job{}, nil
}
func (c client) GetJob(ID dockworker.JobID) (dockworker.Job, error) {
	return dockworker.Job{}, nil
}
func (c client) Logs(ID dockworker.JobID) (io.Reader, error) {
	return strings.NewReader(""), nil
}
