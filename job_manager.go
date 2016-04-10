package dockworker

import (
	log "github.com/Sirupsen/logrus"
	"github.com/fsouza/go-dockerclient"
)

// JobManager manages Jobs
type JobManager interface {
	NotifyNewJob(job Job)
	Start()
	Stop()
}

// NewJobManager returns a new JobManager
func NewJobManager(jobStore JobStore, client *docker.Client, eventListner EventListener, jobUpdater JobUpdater) JobManager {
	return jobManager{
		jobStore:     jobStore,
		client:       client,
		newJobs:      make(chan Job, 100),
		eventListner: eventListner,
		jobUpdater:   jobUpdater,
	}
}

type jobManager struct {
	jobStore     JobStore
	client       *docker.Client
	newJobs      chan Job
	eventListner EventListener
	jobUpdater   JobUpdater
}

func (jm jobManager) Start() {
	log.Info("Job manager starting up...")
	go jm.manager()
}

func (jm jobManager) Stop() {
	// TODO: implement
}

func (jm jobManager) NotifyNewJob(job Job) {
	log.Debugf("Notifying new job %d", job.ID)
	jm.newJobs <- job
}

func (jm jobManager) manager() {
	for {
		select {
		case job := <-jm.newJobs:
			// start new job worker
			log.Debugf("Starting new job %d", job.ID)
			go jm.jobWorker(job)
		}
	}
}
