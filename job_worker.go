package dockworker

import (
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/fsouza/go-dockerclient"
)

func (jm jobManager) jobWorker(job Job) {
	log.Debugf("Running job %+v", job)
	jr, err := newJobRunner(&job, jm.client, jm.eventListner, jm.jobUpdater, jm.stopEventListener)
	if err != nil {
		log.Errorf("Error creating job runner: %s", err)
		jm.jobUpdater.UpdateStatus(&job, JobStatusFailed)
		return
	}
	jr.runJob()
	go SendWebhook(*jr.job)
}

type jobRunner struct {
	client            *docker.Client
	eventListener     DockerEventListener
	stopEventListener StopEventListener
	eventChan         chan *docker.APIEvents
	stopChan          chan JobID
	cmdChan           chan interface{}
	cmdIndex          int
	prevImage         *docker.Image
	currContainer     *docker.Container
	job               *Job
	jobUpdater        JobUpdater
}

func newJobRunner(job *Job, client *docker.Client, eventListener DockerEventListener, jobUpdater JobUpdater, stopEventListener StopEventListener) (*jobRunner, error) {
	jr := &jobRunner{
		client:            client,
		job:               job,
		eventListener:     eventListener,
		jobUpdater:        jobUpdater,
		stopEventListener: stopEventListener,
	}

	// register an event listener
	jr.eventChan = make(chan *docker.APIEvents)
	jr.eventListener.RegisterListener(jr.eventChan)
	jr.stopChan = make(chan JobID)
	jr.stopEventListener.RegisterListener(jr.stopChan)
	jr.cmdChan = make(chan interface{}, 2)
	jr.cmdChan <- true
	jr.cmdIndex = 0
	jr.prevImage = &docker.Image{
		ID: job.ImageName,
	}
	log.Debug(jr.prevImage)
	return jr, nil
}

func (jr *jobRunner) runJob() error {
	defer jr.cleanup()
	// TODO: explicitly handle job update statuses?
	// maybe just fail the job?
	jr.jobUpdater.UpdateStatus(jr.job, JobStatusRunning)

	jr.pullImage()
	for {
		select {
		// TODO: think about how to filter these events
		// on a busy Docker daemon there could be lots of them
		case event, ok := <-jr.eventChan:
			if !ok {
				// channel is closed
				err := fmt.Errorf("Event channel closed")
				log.Warn(err)
				return err
			}
			log.Debugf("Received event %+v", event)
			if err := jr.handleEvent(event); err != nil {
				return err
			}
		case _, ok := <-jr.cmdChan:
			if !ok {
				// no more commands to run
				return nil
			}
			log.Debugf("Running next command")
			if err := jr.runNextCmd(); err != nil {
				log.Errorf("Error running command %s", err)
				return err
			}
		case ID := <-jr.stopChan:
			log.Debugf("Received stop event")
			jr.handleStopRequest(ID)
		}
	}
}

func (jr *jobRunner) handleStopRequest(jobID JobID) {
	if jr.job.ID != jobID {
		return
	}
	log.Infof("Stoppping job %d", jr.job.ID)
	if err := jr.client.StopContainer(jr.currContainer.ID, 5); err != nil {
		log.Errorf("Error stoppping job %d: %s", jr.job.ID, err)
	}
	log.Debugf("Setting status stopped for job %d", jobID)
	jr.jobUpdater.UpdateStatus(jr.job, JobStatusStopped)
}

func (jr *jobRunner) pullImage() error {
	repo, tag := docker.ParseRepositoryTag(jr.job.ImageName)
	opts := docker.PullImageOptions{
		Repository: repo,
		Tag:        tag,
	}
	log.Debugf("Pulling image %s", jr.job.ImageName)
	if err := jr.client.PullImage(opts, docker.AuthConfiguration{}); err != nil {
		log.Errorf("Error pulling image %s: %s", jr.job.ImageName, err)
		jr.jobUpdater.UpdateStatus(jr.job, JobStatusError)
		return err
	}
	log.Debugf("Done pulling image %d", jr.job.ImageName)
	return nil
}

func (jr *jobRunner) cleanup() {
	go func() {
		// read the events out of this channel
		// to ensure no goroutines leak while
		// trying to send on the channel
		for _ = range jr.eventChan {
			// TODO: this never stops flushing...
			// log.Debugf("Flushing Docker event channel")
		}
		// log.Debugf("Done flushing Docker event channel")
	}()
	go func() {
		// read the events out of this channel
		// to ensure no goroutines leak while
		// trying to send on the channel
		for _ = range jr.eventChan {
			// TODO: this never stops flushing...
			// log.Debugf("Flushing stop event channel")
		}
		// log.Debugf("Done flushing stop event channel")
	}()
	log.Debugf("Removing event stop listener")
	jr.stopEventListener.UnregisterListener(jr.stopChan)
	log.Debugf("Stop event listener removed")
}

func (jr *jobRunner) handleEvent(event *docker.APIEvents) error {
	if event == nil {
		return nil
	}
	if jr.currContainer == nil {
		// nothing running yet
		return nil
	}
	if event.ID != jr.currContainer.ID {
		// this event is for a container we don't care about
		return nil
	}

	switch event.Status {
	case "create":
		log.Debugf("Received create status for %s", event.ID)
		return nil

	case "start":
		log.Debugf("Received start status for %s", event.ID)
		jr.handleStartEvent(event)
		return nil

	case "commit":
		log.Debugf("Received commit status for %s", event.ID)
		return nil

	case "die":
		log.Debugf("Received die status for %s", event.ID)
		if err := jr.handleDieEvent(event); err != nil {
			return err
		}
		return nil

	case "stop":
		log.Debugf("Received stop status for %s", event.ID)
		return nil

	case "":
		log.Debugf("Received empty status for %s", event.ID)
		return nil

	default:
		log.Warnf("Received unknown status %s for %s", event.Status, event.ID)
		return nil
	}
}

func (jr *jobRunner) handleStartEvent(event *docker.APIEvents) {
	jr.jobUpdater.UpdateStartTime(jr.job, time.Unix(event.Time, 0))
}

func (jr *jobRunner) handleDieEvent(event *docker.APIEvents) error {
	// the container died, let's see what it returned
	jr.jobUpdater.UpdateEndTime(jr.job, time.Unix(event.Time, 0))
	exitCode, err := jr.client.WaitContainer(jr.currContainer.ID)
	if err != nil {
		log.Errorf("Error waiting for container: %s", err)
		return err
	}
	jr.jobUpdater.AddCmdResult(jr.job, CmdResult(exitCode))
	if exitCode != 0 {
		log.Infof("Container %s exited with non-success code %d", jr.currContainer.ID, exitCode)
		if jr.job.Status != JobStatusStopped {
			// non-zero exit codes only apply to jobs which
			// haven't been forcibly stopped
			log.Debugf("Setting status failed for job %d", jr.job.ID)
			jr.jobUpdater.UpdateStatus(jr.job, JobStatusFailed)
		}
		close(jr.cmdChan)
		return nil
	}

	image, err := jr.client.CommitContainer(docker.CommitContainerOptions{
		Container: jr.currContainer.ID,
	})
	if err != nil {
		log.Errorf("Error committing image: %s", err)
		return err
	}
	log.Debugf("Saving image %s", image.ID)
	jr.jobUpdater.AddImage(jr.job, ImageName(image.ID))
	jr.prevImage = image
	jr.cmdIndex++
	jr.cmdChan <- true
	return nil
}

func (jr *jobRunner) runNextCmd() error {
	// TODO: handle jobs with no explicit commands
	if jr.cmdIndex >= len(jr.job.Cmds) {
		log.Infof("Done running job %d", jr.job.ID)
		jr.jobUpdater.UpdateStatus(jr.job, JobStatusSuccessful)
		close(jr.cmdChan)
		return nil
	}
	config := docker.Config{
		Cmd:   jr.job.Cmds[jr.cmdIndex],
		Image: jr.prevImage.ID,
		Env:   convertEnv(jr.job.Env),
	}

	createOpts := docker.CreateContainerOptions{
		Config: &config,
	}

	container, err := jr.client.CreateContainer(createOpts)
	if err != nil {
		log.Warnf("Failed to create container: %s", err)
		jr.jobUpdater.UpdateStatus(jr.job, JobStatusError)
		close(jr.cmdChan)
		return err
	}

	log.Debugf("New container %+v", container)
	jr.jobUpdater.AddContainer(jr.job, Container(container.ID))

	log.Debugf("%+v", container)

	hostConfig := &docker.HostConfig{}
	err = jr.client.StartContainer(container.ID, hostConfig)
	if err != nil {
		log.Warnf("Failed to start container: %s", err)
		jr.jobUpdater.UpdateStatus(jr.job, JobStatusError)
		close(jr.cmdChan)
		return err
	}
	jr.currContainer = container
	return nil
}

func convertEnv(env map[string]string) []string {
	var converted []string
	for k, v := range env {
		converted = append(converted, fmt.Sprintf("%s=%s", k, v))
	}
	return converted
}
