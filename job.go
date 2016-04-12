package dockworker

// Job is a job
type Job struct {
	ID         JobID             `json:"id"`
	ImageName  string            `json:"image"`
	Env        map[string]string `json:"env"`
	Status     JobStatus         `json:"status"`
	Cmds       []Cmd             `json:"cmds"`
	Message    string            `json:"message"`
	Results    []CmdResult       `json:"results"`
	Containers []Container       `json:"containers"`
	WebhookURL string            `json:"webhook_url"`
}

// CmdResult represents the result of running a command
type CmdResult int

// Cmd is a command to run in the job
type Cmd []string

// JobStatus represents the status of the Job
type JobStatus string

// JobID represents the ID of the Job
type JobID int

// Container represents the ID of the Job
type Container string

const (
	// JobStatusQueued state indicates the job is queued waiting to be run
	JobStatusQueued JobStatus = "queued"
	// JobStatusRunning state indicates the job is running
	JobStatusRunning JobStatus = "running"
	// JobStatusSuccessful state indicates the job has completed successfully
	JobStatusSuccessful JobStatus = "successful"
	// JobStatusFailed state indicates the job has completed with a failure
	JobStatusFailed JobStatus = "failed"
	// JobStatusError state indicates the job could not be run properly
	JobStatusError JobStatus = "error"
	// JobStatusStopped state indicates the job was stoped
	JobStatusStopped JobStatus = "stopped"
)
