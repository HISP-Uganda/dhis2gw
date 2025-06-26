package controllers

import (
	"dhis2gw/joblog"
	"dhis2gw/models"
	"dhis2gw/tasks"
	"dhis2gw/utils"
	_ "embed"
	"encoding/json"
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
