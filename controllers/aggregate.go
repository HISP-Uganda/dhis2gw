package controllers

import (
	"dhis2gw/config"
	"dhis2gw/db"
	"dhis2gw/joblog"
	"dhis2gw/models"
	"dhis2gw/tasks"
	"dhis2gw/utils"
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
	"net/http"
)

//go:embed schemas/simplified_aggregate_request.json
var aggregateRequestSchema string

type AggregateController struct{}

// CreateRequest godoc
// @Summary Submit aggregate data request
// @Description Accepts a JSON payload for an aggregate DHIS2 submission. Requires `Authorization: Token <token>` header.
// @Tags aggregate
// @Accept json
// @Produce json
// @Security BasicAuth
// @Security TokenAuth
// @Param request body models.AggregateRequest true "Aggregate submission payload"
// @Success 200 {object} models.AggregateResponse
// @Failure 400 {object} models.ErrorResponse "Invalid JSON or schema validation failed"
// @Failure 500 {object} models.ErrorResponse "Server-side error"
// @Router /aggregate [post]
func (a *AggregateController) CreateRequest(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON: " + err.Error()})
		return
	}

	valid, errors, err := utils.ValidateJSONAgainstSchemaString(aggregateRequestSchema, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Schema validation error: " + err.Error()})
		return
	}

	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  "Request does not match required schema",
			"detail": errors,
		})
		return
	}

	var request models.AggregateRequest
	jsonBytes, _ := json.Marshal(req)
	if err := json.Unmarshal(jsonBytes, &request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Could not parse validated data: " + err.Error()})
		return
	}

	db := c.MustGet("dbConn").(*sqlx.DB)
	asynqClient := c.MustGet("asynqClient").(*asynq.Client)

	// Now we have a valid AggregateRequest, we can process it
	jl, err := joblog.New(db, request)
	if err != nil {
		log.Errorf("Could not create job log: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to log submission"})
		return
	}

	// 3. Enqueue a background job (pass JobLog ID in payload)
	taskPayload := tasks.AggregateTaskPayload{
		LogID:   jl.ID,
		Payload: request,
	}
	task, err := tasks.NewAggregateTask(taskPayload)
	if err != nil {
		log.Errorf("Could not create aggregate task: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create task"})
		return
	}
	taskInfo, err := asynqClient.Enqueue(task)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to enqueue job"})
		return
	}

	// 4. Update job log with the Asynq Task ID
	_ = jl.UpdateTaskID(taskInfo.ID) // handle error as needed

	c.JSON(http.StatusOK, gin.H{
		"message":       "Aggregate request queued for processing",
		"payload":       request.ToDHIS2AggregatePayload(),
		"submission_id": jl.ID,
		"task_id":       taskInfo.ID,
	})
}

// ReEnqueueAggregateTask godoc
// @Summary Re-enqueue a failed aggregate task
// @Description Re-enqueues a task from the dead or retry queue by its ID. Requires `Authorization: Token
// @Tags aggregate
// @Security BasicAuth
// @Security TokenAuth
// @Param task_id path string true "Task ID to re-enqueue"
// @Param queue query string false "Queue to re-enqueue from (default: dead)"
// @Success 200 {object} models.TaskReEnqueueResponse
// @Failure 404 {object} models.ErrorResponse "Task not found"
// @Failure 500 {object} models.ErrorResponse "Server-side error"
// @Router /aggregate/reenqueue/{task_id} [post]
func (a *AggregateController) ReEnqueueAggregateTask(c *gin.Context) {
	taskID := c.Param("task_id")
	queue := c.DefaultQuery("queue", "default")
	asyncClient := c.MustGet("asynqClient").(*asynq.Client)
	inspector := asynq.NewInspector(asynq.RedisClientOpt{Addr: config.DHIS2GWConf.Server.RedisAddress})

	// Get the task from the dead/retry queue
	info, err := inspector.GetTaskInfo(queue, taskID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get task info in queue: " + err.Error()})
		return
	}

	jl, err := joblog.GetByTaskID(db.GetDB(), taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job log not found for task ID: " + taskID})
		return
	}

	// Create a new Task from the old type & payload
	task := asynq.NewTask(info.Type, info.Payload, asynq.MaxRetry(3))
	// Enqueue as a new task (you can add options, like target queue or delay, here)
	taskInfo, errEnqueue := asyncClient.Enqueue(task)
	if errEnqueue != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to re-enqueue task: " + err.Error()})
		return
	}

	if jl != nil {
		_ = jl.UpdateTaskID(taskInfo.ID)
	}
	_ = inspector.DeleteTask(info.Queue, taskID) // Remove from dead/retry queue
	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Re-enqueued task %s (type: %s) from %s queue", taskID, info.Type, info.Queue),
	})
	return
}

type BatchReEnqueueRequest struct {
	Queue   string   `json:"queue"`    // e.g., "dead" or "retry"
	TaskIDs []string `json:"task_ids"` // task IDs to re-enqueue
}

// BatchReEnqueueAggregateTasksByIDs godoc
// @Summary Re-enqueue multiple aggregate tasks by IDs
// @Description Re-enqueues multiple tasks from the dead or retry queue by their IDs. Requires `Authorization
// @Tags aggregate
// @Security BasicAuth
// @Security TokenAuth
// @Param request body BatchReEnqueueRequest true "Batch re-enqueue request"
// @Success 200 {object} models.BatchReEnqueueResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request body"
// @Failure 404 {object} models.ErrorResponse "Task not found"
// @Failure 500 {object} models.ErrorResponse "Server-side error"
// @Router /aggregate/reenqueue/batch [post]
func (a *AggregateController) BatchReEnqueueAggregateTasksByIDs(c *gin.Context) {
	var req BatchReEnqueueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if req.Queue == "" {
		req.Queue = "dead"
	}

	asyncClient := c.MustGet("asynqClient").(*asynq.Client)
	inspector := asynq.NewInspector(asynq.RedisClientOpt{Addr: config.DHIS2GWConf.Server.RedisAddress})

	reEnqueued := 0
	failed := 0
	errors := []string{}

	for _, taskID := range req.TaskIDs {
		info, err := inspector.GetTaskInfo(req.Queue, taskID)
		if err != nil {
			failed++
			errors = append(errors, fmt.Sprintf("Task %s: %v", taskID, err))
			continue
		}

		task := asynq.NewTask(info.Type, info.Payload)
		taskInfo, err := asyncClient.Enqueue(task, asynq.MaxRetry(3))
		if err != nil {
			failed++
			errors = append(errors, fmt.Sprintf("Task %s: failed to enqueue: %v", taskID, err))
			continue
		}

		// Optionally update your job log
		jl, err := joblog.GetByTaskID(db.GetDB(), taskID)
		if err == nil && jl != nil {
			_ = jl.UpdateTaskID(taskInfo.ID)
		}

		// Optionally, delete original from dead queue:
		// _ = inspector.DeleteTask(req.Queue, taskID)

		reEnqueued++
	}

	c.JSON(http.StatusOK, gin.H{
		"queue":      req.Queue,
		"reEnqueued": reEnqueued,
		"failed":     failed,
		"errors":     errors,
	})
}
