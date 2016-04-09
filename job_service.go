package main

// JobService handles the jobs
type JobService interface {
	Add(job Job) (Job, error)
	Find(ID JobID) (Job, error)
	UpdateStatus(job Job) error
}

// NewJobService returns a new JobService
func NewJobService(jobStore JobStore, jobManager JobManager) JobService {
	return jobService{
		jobStore:   jobStore,
		jobManager: jobManager,
	}
}

type jobService struct {
	jobStore   JobStore
	jobManager JobManager
}

func (service jobService) Add(job Job) (Job, error) {
	// TODO: validations
	job.Status = JobStatusQueued
	job, err := service.jobStore.Add(job)
	if err != nil {
		return Job{}, err
	}

	service.jobManager.NotifyNewJob(job)

	return job, nil
}

func (service jobService) Find(ID JobID) (Job, error) {
	return service.jobStore.Find(ID)
}

func (service jobService) UpdateStatus(job Job) error {
	// make sure nothing but the status gets updated was changed
	j, err := service.jobStore.Find(job.ID)
	if err != nil {
		return err
	}
	j.Status = job.Status
	return service.jobStore.Update(j)
}
