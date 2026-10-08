package http

import (
	"encoding/json"
	apperror "evsched/errors"
	apprequest "evsched/http/request"
	"evsched/http/response"
	"evsched/job"
	"evsched/logging"
	"fmt"
	"net/http"
)

type JobHandler struct {
	service job.JobService
}

func NewJobHandler(service job.JobService) *JobHandler {
	return &JobHandler{service: service}
}

func (h JobHandler) AddJob(writer http.ResponseWriter, request *http.Request) {

	logger := logging.FromContext(request.Context())

	var req apprequest.CreateJobRequest
	if err := json.NewDecoder(request.Body).Decode(&req); err != nil {
		logger.Error("add job - request body parse json failed", "error", err)
		WriteError(writer, apperror.InvalidInput(fmt.Sprintf("invalid request body: %s", err.Error())))
		return
	}

	if err := req.Validate(); err != nil {
		logger.Error("add job - validate failed", "error", err)
		WriteError(writer, apperror.InvalidInput(fmt.Sprintf("invalid request body: %s", err.Error())))
		return
	}

	jobCreated, err := h.service.AddJob(request.Context(), req.JobType, req.SourceID, req.JobData, req.IsCron, req.Cron, req.ScheduledAt)
	if err != nil {
		logger.Error("add job - create job failed", "error", err)
		WriteError(writer, err)
		return
	}

	jobResponse := response.JobResponse{
		JobID:       jobCreated.JobID,
		ScheduledAt: jobCreated.ScheduledAt,
	}
	if err = WriteSuccessDetail(writer, http.StatusCreated, "create job success", jobResponse); err != nil {
		WriteError(writer, err)
	}
}

func (h JobHandler) PauseJob(writer http.ResponseWriter, request *http.Request) {

	logger := logging.FromContext(request.Context())

	jobID := request.PathValue("jobID")
	if jobID == "" {
		WriteError(writer, apperror.InvalidInput("jobID is required"))
		return
	}

	err := h.service.Pause(request.Context(), jobID)
	if err != nil {
		logger.Error("pause job failed", "error", err)
		WriteError(writer, err)
		return
	}

	if err = WriteSuccessDetail(writer, http.StatusOK, "pause job success", ""); err != nil {
		WriteError(writer, err)
	}

}

func (h JobHandler) ResumeJob(writer http.ResponseWriter, request *http.Request) {
	logger := logging.FromContext(request.Context())

	jobID := request.PathValue("jobID")
	if jobID == "" {
		WriteError(writer, apperror.InvalidInput("jobID is required"))
		return
	}

	err := h.service.Resume(request.Context(), jobID)
	if err != nil {
		logger.Error("resume job failed", "error", err)
		WriteError(writer, err)
		return
	}

	if err = WriteSuccessDetail(writer, http.StatusOK, "resume job success", ""); err != nil {
		WriteError(writer, err)
	}
}

func (h JobHandler) DeleteJob(writer http.ResponseWriter, request *http.Request) {
	logger := logging.FromContext(request.Context())

	jobID := request.PathValue("jobID")
	if jobID == "" {
		WriteError(writer, apperror.InvalidInput("jobID is required"))
		return
	}

	err := h.service.Delete(request.Context(), jobID)
	if err != nil {
		logger.Error("delete job failed", "error", err)
		WriteError(writer, err)
		return
	}

	if err = WriteSuccessDetail(writer, http.StatusOK, "delete job success", ""); err != nil {
		WriteError(writer, err)
	}
}

func (h JobHandler) GetJob(writer http.ResponseWriter, request *http.Request) {
	logger := logging.FromContext(request.Context())

	jobID := request.PathValue("jobID")
	if jobID == "" {
		WriteError(writer, apperror.InvalidInput("jobID is required"))
		return
	}

	job, err := h.service.Get(request.Context(), jobID)
	if err != nil {
		logger.Error("get job failed", "error", err)
		WriteError(writer, err)
		return
	}

	if err = WriteSuccessDetail(writer, http.StatusOK, "", job); err != nil {
		WriteError(writer, err)
	}
}

func (h JobHandler) GetJobScheduledTime(writer http.ResponseWriter, request *http.Request) {
	logger := logging.FromContext(request.Context())

	jobID := request.PathValue("jobID")
	if jobID == "" {
		WriteError(writer, apperror.InvalidInput("jobID is required"))
		return
	}

	scheduledTime, err := h.service.GetNextRunTime(request.Context(), jobID)
	if err != nil {
		logger.Error("get job scheduled time failed", "error", err)
		WriteError(writer, err)
		return
	}

	if err = WriteSuccessDetail(writer, http.StatusOK, "", scheduledTime); err != nil {
		WriteError(writer, err)
	}
}
