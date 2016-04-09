package main

import (
	"bytes"
	"fmt"
	"strconv"

	log "github.com/Sirupsen/logrus"
	"github.com/fsouza/go-dockerclient"
)

func (jm jobManager) jobWorker(job Job) {
	log.Debugf("Running job %+v", job)
	job.Status = Running
	jm.updateStatus(job)
	err := jm.runJob(&job)
	if err != nil {
		job.Status = Failed
		jm.updateStatus(job)
		return
	}
	job.Status = Successful
	jm.updateStatus(job)
}

func (jm jobManager) runJob(job *Job) error {
	commands := job.Cmds

	var prevImage *docker.Image

	for i, cmd := range commands {
		imageName := job.ImageName
		if prevImage != nil {
			imageName = prevImage.ID
		}
		config := docker.Config{
			Cmd:   cmd,
			Image: imageName,
		}

		createOpts := docker.CreateContainerOptions{
			Config: &config,
		}

		container, err := jm.client.CreateContainer(createOpts)
		if err != nil {
			log.Errorf("Failed to create container: %s", err)
			return err
		}

		log.Debugf("%+v", container)

		hostConfig := &docker.HostConfig{}
		err = jm.client.StartContainer(container.ID, hostConfig)
		if err != nil {
			log.Errorf("Failed to start container: %s", err)
			return err
		}
		log.Debugf("Container started, waiting for it to finish")
		exitCode, err := jm.client.WaitContainer(container.ID)
		if err != nil {
			log.Errorf("Error waiting for container: %s", err)
			return err
		}
		log.Debugf("Container exited with code %d", exitCode)
		log.Debug("Getting logs")
		stdOut := bytes.NewBuffer([]byte{})
		stdErr := bytes.NewBuffer([]byte{})
		err = jm.client.Logs(docker.LogsOptions{
			Container:    container.ID,
			OutputStream: stdOut,
			ErrorStream:  stdErr,
			Stdout:       true,
			Stderr:       true,
		})

		if err != nil {
			log.Errorf("Error getting logs: %s", err)
			return err
		}

		log.Debugf("Stdout: %s", string(stdOut.Bytes()))
		log.Debugf("Stderr: %s", string(stdErr.Bytes()))

		image, err := jm.client.CommitContainer(docker.CommitContainerOptions{
			Container:  container.ID,
			Repository: fmt.Sprintf("dockworker-%d", job.ID),
			Tag:        strconv.Itoa(i),
		})
		if err != nil {
			log.Errorf("Error committing image: %s", err)
			return err
		}
		log.Debugf("%+v", image)
		prevImage = image
	}
	return nil
}
