package controllers

import (
	"dhis2gw/joblog"
	"dhis2gw/models"
	"dhis2gw/utils/dbutils"
	"fmt"
	"github.com/HISP-Uganda/go-dhis2-sdk/utils"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"math"
	"net/http"
	"strconv"
	"time"
)

type LogsController struct{}

type JobLogPaginatedResponse models.PaginatedResponse[joblog.JobLogSwagger]

// GetLogsHandler godoc
// @Summary Get job logs
// @Description Returns a paginated list of job logs with optional filters like status, task ID, job ID, and submission date range.
// @Tags logs
// @Produce json
// @Security BasicAuth
// @Security TokenAuth
// @Param        status        query     string  false  "Filter by status"
// @Param        task_id       query     string  false  "Filter by task id"
// @Param        job_id        query     integer false  "Filter by job id"
// @Param        submitted_at  query     string  false  "Filter by exact submitted_at (RFC3339)"
// @Param        submitted_from query    string  false  "Submitted after (RFC3339)"
// @Param        submitted_to  query     string  false  "Submitted before (RFC3339)"
// @Param        page          query     int     false  "Page number (default 1)"
// @Param        page_size     query     int     false  "Items per page (default 20)"
// @Success 200 {object}  JobLogPaginatedResponse
// @Failure 400 {object} models.ErrorResponse "Invalid query parameters"
// @Failure 500 {object} models.ErrorResponse "Server-side error"
// @Router /logs [get]
func (l *LogsController) GetLogsHandler(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Parse filters from query params
		var filter joblog.JobLogFilter

		if status := c.Query("status"); status != "" {
			filter.Status = &status
		}
		if taskID := c.Query("task_id"); taskID != "" {
			filter.TaskID = &taskID
		}
		if jobID := c.Query("job_id"); jobID != "" {
			if id, err := strconv.ParseInt(jobID, 10, 64); err == nil {
				filter.JobID = &id
			}
		}
		if submitted := c.Query("submitted_at"); submitted != "" {
			if t, err := time.Parse("2006-01-02", submitted); err == nil {
				filter.SubmittedAt = t
			}
		}
		if submittedFrom := c.Query("submitted_from"); submittedFrom != "" {
			if t2, err := time.Parse("2006-01-02", submittedFrom); err == nil {
				filter.SubmittedFrom = t2
			}
		}
		if submittedTo := c.Query("submitted_to"); submittedTo != "" {
			if t, err := time.Parse("2006-01-02", submittedTo); err == nil {
				filter.SubmittedTo = t
			}
		}
		// Pagination params
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
		filter.Page = page
		filter.PageSize = pageSize

		// Get logs and total count
		logs, total, err := joblog.GetLogs(db, &filter)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

		response := models.PaginatedResponse[joblog.JobLog]{
			Items:      logs,
			Total:      int64(total),
			Page:       filter.Page,
			TotalPages: totalPages,
			PageSize:   filter.PageSize,
		}

		c.JSON(http.StatusOK, response)
	}
}

// GetLogByIdHandler godoc
// @Summary      Get job log by ID
// @Description  Get a specific job log entry by its database ID.
// @Tags         logs
// @Produce      json
// @Security     BasicAuth
// @Security     TokenAuth
// @Param        id   path      int  true   "Log ID"
// @Success      200  {object}  joblog.JobLogSwagger "Job log entry"
// @Failure      404  {object} models.ErrorResponse "Log not found"
// @Failure      500  {object}  models.ErrorResponse "Server-side error"
// @Router       /logs/{id} [get]
func (l *LogsController) GetLogByIdHandler(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid log ID"})
			return
		}

		var log joblog.JobLog
		err = db.Get(&log, "SELECT * FROM submission_log WHERE id = $1", id)
		if err != nil {
			// sqlx returns error if not found or DB error
			c.JSON(http.StatusNotFound, gin.H{"error": "Log not found"})
			return
		}
		payload, _ := dbutils.RawMessageToMap(log.Payload)
		jl := joblog.JobLogSwagger{
			ID:         log.ID,
			TaskID:     utils.StringPtr(log.TaskID.String),
			Status:     log.Status,
			Submitted:  log.Submitted,
			Payload:    payload,
			RetryCount: log.RetryCount,
			Response:   utils.StringPtr(log.Response.String),
			Errors:     utils.StringPtr(log.Errors.String),
		}

		c.JSON(http.StatusOK, jl)
	}
}

//func (l *LogsController) ReprocessLogHandler(db *sqlx.DB) gin.HandlerFunc {
//	return func(c *gin.Context) {
//		idStr := c.Param("id")
//		id, err := strconv.ParseInt(idStr, 10, 64)
//		if err != nil {
//			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid log ID"})
//			return
//		}
//
//		jl, err := joblog.GetByID(db, id)
//		if err != nil {
//			c.JSON(http.StatusNotFound, gin.H{"error": "Log not found"})
//			return
//		}
//
//		// Update payload to include reprocessing marker
//		payload, _ := dbutils.RawMessageToMap(jl.Payload)
//		payload["reprocessed"] = true
//		payload["reprocessed_from_id"] = jl.ID
//		newPayload, _ := dbutils.MapToRawMessage(payload)
//
//		// Enqueue reprocessed job
//		if err := joblog.EnqueueReprocessedJob(c.Request.Context(), db, jl.JobType, newPayload); err != nil {
//			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to enqueue reprocessed job"})
//			return
//		}
//
//		c.JSON(http.StatusOK, gin.H{"message": "Job reprocessed successfully", "job_id": jl.ID})
//	}
//}

// DeleteSubmissionLogHandler godoc
// @Summary      Delete a job log by ID
// @Description  Deletes a specific job log entry by its database ID.
// @Tags         logs
// @Produce      json
// @Security     BasicAuth
// @Security     TokenAuth
// @Param        id   path      int  true   "Log ID"
// @Success      200  {object}  models.SuccessResponse "Deletion successful"
// @Failure      404  {object} models.ErrorResponse "Log not found"
// @Failure      500  {object}  models.ErrorResponse "Server-side error"
// @Router       /logs/{id} [delete]
func (l *LogsController) DeleteSubmissionLogHandler(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid log ID"})
			return
		}

		res, err := db.Exec("DELETE FROM submission_log WHERE id = $1", id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete log"})
			return
		}

		rowsAffected, _ := res.RowsAffected()
		if rowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Log not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Deleted %d log(s)", rowsAffected)})
	}
}

// PurgeSubmissionLogsByDateHandler godoc
// @Summary   	Purge submission logs by date
// @Description  Deletes all submission logs older than the specified date.
// @Tags         logs
// @Produce      json
// @Security     BasicAuth
// @Security     TokenAuth
// @Param        date  query     string  true   "Cutoff date in RFC3339 format (e.g., 2024-06-01T00:00:00Z)"
// @Success      200   {object}  models.SuccessResponse "Purge result"
// @Failure      400   {object}  models.ErrorResponse "Invalid date format"
// @Failure      500   {object}  models.ErrorResponse "Server-side error"
// @Router       /logs/purge [delete]
func (l *LogsController) PurgeSubmissionLogsByDateHandler(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		dateStr := c.Query("date")
		if dateStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing 'date' query parameter"})
			return
		}

		// Try to parse the date. Accepts RFC3339 (e.g., "2024-06-01T00:00:00Z")
		cutoff, err := time.Parse(time.RFC3339, dateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid 'date' format, use RFC3339"})
			return
		}

		res, err := db.Exec(`DELETE FROM submission_log WHERE submitted_at < $1`, cutoff)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete logs"})
			return
		}
		rows, _ := res.RowsAffected()
		c.JSON(http.StatusOK, gin.H{
			"message": fmt.Sprintf("Deleted %d logs older than %s", rows, cutoff.Format(time.RFC3339)),
		})
	}
}
