package main

// Job is a job
type Job struct {
	ID        JobID       `json:"id"`
	ImageName string      `json:"image"`
	Status    JobStatus   `json:"status"`
	Cmds      []Cmd       `json:"cmds"`
	Message   string      `json:"message"`
	Results   []CmdResult `json:"results"`
}

// CmdResult represents the result of running a command
type CmdResult int

// Cmd is a command to run in the job
type Cmd []string

// JobStatus represents the status of the Job
type JobStatus string

// JobID represents the ID of the Job
type JobID int

const (
	// Queued state indicates the job is queued waiting to be run
	Queued JobStatus = "queued"
	// Running state indicates the job is running
	Running JobStatus = "running"
	// Successful state indicates the job has completed successfully
	Successful JobStatus = "successful"
	// Failed state indicates the job has completed with a failure
	Failed JobStatus = "failed"
)
