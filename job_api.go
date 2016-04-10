package main

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	log "github.com/Sirupsen/logrus"
	"github.com/emicklei/go-restful"
)

// JobAPI is a jobs api
type JobAPI struct {
	jobService JobService
	logService LogService
}

// NewJobAPI creates a new JobAPI
func NewJobAPI(jobService JobService, logService LogService) JobAPI {
	return JobAPI{
		jobService: jobService,
		logService: logService,
	}
}

// Register registers the job api's routes
func (api JobAPI) Register(container *restful.Container) {
	// TODO: Pretty print responses flag
	ws := new(restful.WebService)
	ws.Path("/jobs").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/{id}").To(api.findJob).
		Operation("findJob").
		Param(ws.PathParameter("id", "id of job").DataType("int")).
		Writes(Job{}))

	ws.Route(ws.POST("").To(api.createJob).
		Operation("createJob").
		Reads(Job{}))

	ws.Route(ws.GET("/{id}/logs").To(api.logs).
		Operation("logs").
		Param(ws.PathParameter("id", "id of job").DataType("int")).
		Produces("text/plain"))

	container.Add(ws)
}

func (api JobAPI) findJob(request *restful.Request, response *restful.Response) {
	id, err := strconv.Atoi(request.PathParameter("id"))
	if err != nil {
		logAndRespondError(response, http.StatusInternalServerError, ErrInvalidJobID)
		return
	}
	jobID := JobID(id)

	job, err := api.jobService.Find(jobID)
	if err != nil {
		switch err {
		case ErrJobNotFound:
			logAndRespondError(response, http.StatusNotFound, ErrJobNotFound)
			return
		default:
			logAndRespondError(response, http.StatusNotFound, err)
			return
		}
	}
	response.WriteHeaderAndEntity(http.StatusOK, job)
}

func (api JobAPI) createJob(request *restful.Request, response *restful.Response) {
	job := &Job{}
	err := request.ReadEntity(job)
	if err != nil {
		if err == io.EOF {
			logAndRespondError(response, http.StatusBadRequest, fmt.Errorf("Invalid JSON"))
			return
		}
		logAndRespondError(response, http.StatusInternalServerError, err)
		return
	}

	log.Debugf("Incoming job: %+v", job)

	j, err := api.jobService.Add(*job)
	if err != nil {
		logAndRespondError(response, http.StatusInternalServerError, err)
		return
	}
	response.WriteHeaderAndEntity(http.StatusCreated, j)
}

func (api JobAPI) logs(request *restful.Request, response *restful.Response) {
	id, err := strconv.Atoi(request.PathParameter("id"))
	if err != nil {
		logAndRespondError(response, http.StatusInternalServerError, ErrInvalidJobID)
		return
	}
	jobID := JobID(id)

	job, err := api.jobService.Find(jobID)
	if err != nil {
		switch err {
		case ErrJobNotFound:
			logAndRespondError(response, http.StatusNotFound, ErrJobNotFound)
			return
		default:
			logAndRespondError(response, http.StatusInternalServerError, err)
			return
		}
	}

	// get the logs
	if err := api.logService.GetLogs(job, response.ResponseWriter); err != nil {
		logAndRespondError(response, http.StatusInternalServerError, err)
		return
	}
}

func logAndRespondError(response *restful.Response, status int, err error) {
	log.Infof("Error response %d %s", status, err)
	response.WriteHeaderAndEntity(status, errorResponse(err.Error()))
}
