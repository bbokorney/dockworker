package main

import (
	"io"

	log "github.com/Sirupsen/logrus"
	"github.com/fsouza/go-dockerclient"
)

// LogService handles retrieving logs from containers
type LogService interface {
	GetLogs(job Job, output io.Writer) error
}

// NewLogService returns a new LogService
func NewLogService(jobStore JobStore, client *docker.Client) LogService {
	return logService{
		jobStore: jobStore,
		client:   client,
	}
}

type logService struct {
	jobStore JobStore
	client   *docker.Client
}

func (ls logService) GetLogs(job Job, output io.Writer) error {
	for _, container := range job.Containers {
		err := ls.client.Logs(docker.LogsOptions{
			Container:    string(container),
			OutputStream: output,
			ErrorStream:  output,
			Stdout:       true,
			Stderr:       true,
		})
		if err != nil {
			log.Errorf("Error getting logs from container %s: %s", container, err)
			return err
		}
	}
	return nil
}
