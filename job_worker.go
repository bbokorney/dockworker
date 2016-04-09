package main

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/fsouza/go-dockerclient"
)

func (jm jobManager) jobWorker(job Job) {
	log.Debugf("Running job %+v", job)
	job.Status = Running
	jm.updateStatus(job)
	jr, err := newJobRunner(&job, jm.client, jm.eventListner)
	if err != nil {
		log.Errorf("Error creating job runner: %s", err)
		job.Status = Failed
		jm.updateStatus(job)
		return
	}
	err = jr.runJob()
	if err != nil {
		job.Status = Failed
		jm.updateStatus(job)
		return
	}
	job.Status = Successful
	jm.updateStatus(job)
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
}

func newJobRunner(job *Job, client *docker.Client, eventListener EventListener) (*jobRunner, error) {
	jr := &jobRunner{
		client:        client,
		job:           job,
		eventListener: eventListener,
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
		case <-jr.cmdChan:
			log.Debugf("Running next command")
			done, err := jr.runNextCmd()
			if err != nil {
				log.Errorf("Error running command %s", err)
				return err
			}
			if done {
				return nil
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
	if exitCode != 0 {
		// TODO: this is the wrong use of an error
		// a job exiting with an error is not an actual error,
		// so shouldn't be handled as one
		err := fmt.Errorf("Container %s exited with non-success code %d", jr.currContainer.ID, exitCode)
		log.Info(err)
		return err
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

func (jr *jobRunner) runNextCmd() (done bool, err error) {
	// TODO: handle jobs with no explicit commands
	if jr.cmdIndex >= len(jr.job.Cmds) {
		log.Infof("Done running job %d", jr.job.ID)
		return true, nil
	}
	config := docker.Config{
		Cmd:   jr.job.Cmds[jr.cmdIndex],
		Image: jr.prevImage.ID,
	}

	createOpts := docker.CreateContainerOptions{
		Config: &config,
	}

	container, err := jr.client.CreateContainer(createOpts)
	if err != nil {
		log.Errorf("Failed to create container: %s", err)
		return false, err
	}

	log.Debugf("%+v", container)

	hostConfig := &docker.HostConfig{}
	err = jr.client.StartContainer(container.ID, hostConfig)
	if err != nil {
		log.Errorf("Failed to start container: %s", err)
		return false, err
	}
	jr.currContainer = container
	return false, nil
}
