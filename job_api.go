package main

import (
	"net/http"
	"strconv"

	log "github.com/Sirupsen/logrus"
	"github.com/emicklei/go-restful"
)

// JobAPI is a jobs api
type JobAPI struct {
	jobService JobService
}

// NewJobAPI creates a new JobAPI
func NewJobAPI(jobService JobService) JobAPI {
	return JobAPI{
		jobService: jobService,
	}
}

// Register registers the job api's routes
func (api JobAPI) Register(container *restful.Container) {
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

	container.Add(ws)
}

func (api JobAPI) findJob(request *restful.Request, response *restful.Response) {
	id, err := strconv.Atoi(request.PathParameter("id"))
	if err != nil {
		response.WriteHeaderAndEntity(http.StatusBadRequest, errorResponse("Invalid job ID"))
		return
	}
	jobID := JobID(id)

	job, err := api.jobService.Find(jobID)
	if err != nil {
		switch err {
		case ErrJobNotFound:
			response.WriteHeaderAndEntity(http.StatusNotFound, errorResponse("No job with that ID"))
			return
		default:
			response.WriteHeaderAndEntity(http.StatusInternalServerError, errorResponse(err.Error()))
			return
		}
	}
	response.WriteHeaderAndEntity(http.StatusOK, job)
}

func (api JobAPI) createJob(request *restful.Request, response *restful.Response) {
	job := &Job{}
	err := request.ReadEntity(job)
	if err != nil {
		response.WriteHeaderAndEntity(http.StatusInternalServerError, errorResponse(err.Error()))
		return
	}

	log.Debugf("Incoming job: %+v", job)

	j, err := api.jobService.Add(*job)
	if err != nil {
		response.WriteHeaderAndEntity(http.StatusInternalServerError, errorResponse(err.Error()))
		return
	}
	response.WriteHeaderAndEntity(http.StatusCreated, j)
}
