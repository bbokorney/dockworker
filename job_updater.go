package dockworker

import (
	"time"

	log "github.com/Sirupsen/logrus"
)

// JobUpdater handles updating jobs
type JobUpdater interface {
	UpdateStatus(job *Job, status JobStatus) error
	UpdateStartTime(job *Job, startTime time.Time) error
	UpdateEndTime(job *Job, endTime time.Time) error
	AddCmdResult(job *Job, result CmdResult) error
	AddContainer(job *Job, container Container) error
	AddImage(job *Job, image ImageName) error
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

func (ju jobUpdater) UpdateEndTime(job *Job, endTime time.Time) error {
	// make sure nothing but the status gets updated
	j, err := ju.jobStore.Find(job.ID)
	if err != nil {
		log.Errorf("Error finding job during end time update %d: %s", job.ID, err)
		return err
	}
	// TODO: this is redundant, see if we can improve
	job.EndTime = endTime
	j.EndTime = endTime
	err = ju.jobStore.Update(j)
	if err != nil {
		log.Errorf("Error updating job end time %d: %s", job.ID, err)
		return err
	}
	return nil
}

func (ju jobUpdater) UpdateStartTime(job *Job, startTime time.Time) error {
	// make sure nothing but the status gets updated
	j, err := ju.jobStore.Find(job.ID)
	if err != nil {
		log.Errorf("Error finding job during start time update %d: %s", job.ID, err)
		return err
	}
	// TODO: this is redundant, see if we can improve
	job.StartTime = startTime
	j.StartTime = startTime
	err = ju.jobStore.Update(j)
	if err != nil {
		log.Errorf("Error updating job start time %d: %s", job.ID, err)
		return err
	}
	return nil
}

func (ju jobUpdater) UpdateStatus(job *Job, status JobStatus) error {
	// make sure nothing but the status gets updated
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

func (ju jobUpdater) AddImage(job *Job, image ImageName) error {
	j, err := ju.jobStore.Find(job.ID)
	if err != nil {
		log.Errorf("Error finding job during images update %d: %s", job.ID, err)
		return err
	}
	job.Images = append(job.Images, image)
	j.Images = append(j.Images, image)
	err = ju.jobStore.Update(j)
	if err != nil {
		log.Errorf("Error updating job images %d: %s", job.ID, err)
		return err
	}
	return nil
}

func (ju jobUpdater) AddContainer(job *Job, container Container) error {
	j, err := ju.jobStore.Find(job.ID)
	if err != nil {
		log.Errorf("Error finding job during containers update %d: %s", job.ID, err)
		return err
	}
	job.Containers = append(job.Containers, container)
	j.Containers = append(j.Containers, container)
	err = ju.jobStore.Update(j)
	if err != nil {
		log.Errorf("Error updating job containers %d: %s", job.ID, err)
		return err
	}
	return nil
}
