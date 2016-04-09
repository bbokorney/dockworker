package main

import log "github.com/Sirupsen/logrus"

// JobUpdater handles updating jobs
type JobUpdater interface {
	UpdateStatus(job *Job, status JobStatus) error
	AddCmdResult(job *Job, result CmdResult) error
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
		log.Errorf("Error finding job during status update %d: %s", job.ID, err)
		return err
	}
	// TODO: this is redundant, see if we can improve
	job.Status = status
	j.Status = status
	err = ju.jobStore.Update(j)
	if err != nil {
		log.Errorf("Error updating job status %d: %s", job.ID, err)
		return err
	}
	return nil
}

func (ju jobUpdater) AddCmdResult(job *Job, result CmdResult) error {
	j, err := ju.jobStore.Find(job.ID)
	if err != nil {
		log.Errorf("Error finding job during results update %d: %s", job.ID, err)
		return err
	}
	job.Results = append(job.Results, result)
	j.Results = append(j.Results, result)
	err = ju.jobStore.Update(j)
	if err != nil {
		log.Errorf("Error updating job results %d: %s", job.ID, err)
		return err
	}
	return nil
}
