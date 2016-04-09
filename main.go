package main

import (
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/emicklei/go-restful"
	"github.com/fsouza/go-dockerclient"
)

// TODO:
// * handle shutdown signal, stop workers

func main() {
	wsContainer := initWSContainer()
	log.Info("Starting up...")
	log.Fatal(http.ListenAndServe(":4321", wsContainer))
}

func initWSContainer() *restful.Container {
	log.SetLevel(log.DebugLevel)
	jobAPI := initJobAPI()
	wsContainer := restful.NewContainer()
	jobAPI.Register(wsContainer)
	return wsContainer
}

func initJobAPI() JobAPI {
	client, err := docker.NewClientFromEnv()
	if err != nil {
		log.Fatalf("Error creating client %s", err)
	}
	log.Debugf("%+v", client)
	err = client.Ping()
	if err != nil {
		log.Fatalf("Failed to ping Docker daemon: %s", err)
	}
	eventListener := NewEventListener(client)
	err = eventListener.Start()
	if err != nil {
		log.Fatalf("Failed to start event listener: %s", err)
	}
	jobStore := NewJobStore()
	jobManager := NewJobManager(jobStore, client, eventListener)
	jobManager.Start()
	jobService := NewJobService(jobStore, jobManager)
	return NewJobAPI(jobService)
}
