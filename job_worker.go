package main

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/fsouza/go-dockerclient"
)

func (jm jobManager) jobWorker(job Job) {
	log.Debugf("Running job %+v", job)
	jr, err := newJobRunner(&job, jm.client, jm.eventListner, jm.jobUpdater)
	if err != nil {
		log.Errorf("Error creating job runner: %s", err)
		jm.jobUpdater.UpdateStatus(&job, JobStatusFailed)
		return
	}
	jr.runJob()
}

type jobRunner struct {
	client        *docker.Client
	eventListener EventListener
	eventChan     chan *docker.APIEvents
	cmdChan       chan interface{}
	cmdIndex      int
	prevImage     *docker.Image
	currContainer *docker.Container
	job           *Job
	jobUpdater    JobUpdater
}

func newJobRunner(job *Job, client *docker.Client, eventListener EventListener, jobUpdater JobUpdater) (*jobRunner, error) {
	jr := &jobRunner{
		client:        client,
		job:           job,
		eventListener: eventListener,
		jobUpdater:    jobUpdater,
	}

	// register an event listener
	jr.eventChan = make(chan *docker.APIEvents)
	jr.eventListener.RegisterListener(jr.eventChan)
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
		}
	}
}

func (jr *jobRunner) cleanup() {
	go func() {
		// read the events out of this channel
		// to ensure no goroutines leak while
		// trying to send on the channel
		for _ = range jr.eventChan {
			log.Debugf("Flushing event channel")
		}
	}()
	log.Debugf("Removing event listener")
	jr.eventListener.UnregisterListener(jr.eventChan)
	log.Debugf("Event listener removed")
}

func (jr *jobRunner) handleEvent(event *docker.APIEvents) error {
	if event == nil {
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
		return nil

	case "commit":
		log.Debugf("Received commit status for %s", event.ID)
		return nil

	case "die":
		log.Debugf("Received die status for %s", event.ID)
		if err := jr.handleDieEvent(); err != nil {
			return err
		}
		return nil

	case "":
		log.Debugf("Received empty status for %s", event.ID)
		return nil

	default:
		log.Warnf("Received unknown status %s for %s", event.Status, event.ID)
		return nil
	}
}

func (jr *jobRunner) handleDieEvent() error {
	// the container died, let's see what it returned
	exitCode, err := jr.client.WaitContainer(jr.currContainer.ID)
	if err != nil {
		log.Errorf("Error waiting for container: %s", err)
		return err
	}
	jr.jobUpdater.AddCmdResult(jr.job, CmdResult(exitCode))
	if exitCode != 0 {
		log.Infof("Container %s exited with non-success code %d", jr.currContainer.ID, exitCode)
		jr.jobUpdater.UpdateStatus(jr.job, JobStatusFailed)
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
	log.Debugf("%+v", image)
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
