package main

import log "github.com/Sirupsen/logrus"

// JobUpdater handles updating jobs
type JobUpdater interface {
	UpdateStatus(job *Job, status JobStatus) error
}

// NewJobUpdater returns a new JobUpdater
func NewJobUpdater(jobStore JobStore) JobUpdater {
	return jobUpdater{
		jobStore: jobStore,
	}
}

type jobUpdater struct {
	jobStore JobStore
}

func (ju jobUpdater) UpdateStatus(job *Job, status JobStatus) error {
	// make sure nothing but the status gets updated was changed
	j, err := ju.jobStore.Find(job.ID)
	if err != nil {
		log.Errorf("Error finding job during update %d: %s", job.ID, err)
		return err
	}
	job.Status = status
	j.Status = status
	err = ju.jobStore.Update(j)
	if err != nil {
		log.Errorf("Error updating job %d: %s", job.ID, err)
		return err
	}
	return nil
}
