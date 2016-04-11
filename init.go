package dockworker

import (
	log "github.com/Sirupsen/logrus"
	"github.com/emicklei/go-restful"
	"github.com/fsouza/go-dockerclient"
	"github.com/pborman/uuid"
)

// InitWSContainer sets up the program
func InitWSContainer() *restful.Container {
	log.SetLevel(log.DebugLevel)
	jobAPI := initJobAPI()
	wsContainer := restful.NewContainer()
	wsContainer.Filter(globalLogging)
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
	jobUpdater := NewJobUpdater(jobStore)
	jobManager := NewJobManager(jobStore, client, eventListener, jobUpdater)
	jobManager.Start()
	logService := NewLogService(jobStore, client)
	jobService := NewJobService(jobStore, jobManager)
	return NewJobAPI(jobService, logService)
}

func globalLogging(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	reqID := uuid.New()
	log.Infof("%s %s %s", req.Request.Method, req.Request.URL, reqID)
	chain.ProcessFilter(req, resp)
	log.Infof("%d %s %s", resp.StatusCode(), req.Request.URL, reqID)
}
