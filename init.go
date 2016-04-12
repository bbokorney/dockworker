package dockworker

import (
	"os"
	"os/signal"
	"syscall"

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
	stopEventChan := make(chan JobID)
	stopEventListener := NewStopEventListener(stopEventChan)
	stopEventListener.Start()
	jobStore := NewJobStore()
	jobUpdater := NewJobUpdater(jobStore)
	jobManager := NewJobManager(jobStore, client, eventListener, jobUpdater, stopEventListener)
	jobManager.Start()
	logService := NewLogService(jobStore, client)
	jobService := NewJobService(jobStore, jobManager)
	stopService := NewStopService(stopEventChan)
	// TODO: pass in everything which requires cleanup for a shutdown
	signalHandler()
	return NewJobAPI(jobService, logService, stopService)
}

func signalHandler() {
	go func() {
		signalChannel := make(chan os.Signal)
		signal.Notify(signalChannel, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
		sig := <-signalChannel
		log.Infof("Shutting down from signal %s", sig)
		os.Exit(0)
	}()
}

func globalLogging(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	reqID := uuid.New()
	log.Infof("%s %s %s", req.Request.Method, req.Request.URL, reqID)
	chain.ProcessFilter(req, resp)
	log.Infof("%d %s %s", resp.StatusCode(), req.Request.URL, reqID)
}
