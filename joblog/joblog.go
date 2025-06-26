package joblog

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
	"strings"
	"time"
)

type JobLog struct {
	ID          int64           `db:"id" json:"id"`
	Submitted   time.Time       `db:"submitted_at" json:"submitted_at"`
	Payload     json.RawMessage `db:"payload" swaggertype:"object" json:"payload"`
	Status      string          `db:"status" json:"status"`
	RetryCount  int             `db:"retry_count" json:"retry_count"`
	LastAttempt sql.NullTime    `db:"last_attempt_at" json:"last_attempt"`
	TaskID      sql.NullString  `db:"task_id" json:"task_id"`
	Response    sql.NullString  `db:"response" json:"response"`
	Errors      sql.NullString  `db:"errors" json:"errors"` // Optional field for storing error messages

	db *sqlx.DB `json:"-"` // not persisted, for method receivers
}

// JobLogSwagger is for Swagger documentation only
type JobLogSwagger struct {
	ID          int64                  `json:"id" example:"123"`
	Submitted   time.Time              `json:"submitted_at" example:"2024-06-24T08:00:00Z"`
	Payload     map[string]interface{} `json:"payload" swaggertype:"object"`
	Status      string                 `json:"status" example:"SUCCESS"`
	RetryCount  int                    `json:"retry_count" example:"0"`
	LastAttempt *time.Time             `json:"last_attempt_at,omitempty" example:"2024-06-24T09:00:00Z"`
	TaskID      *string                `json:"task_id,omitempty" example:"abc-123"`
	Response    *string                `json:"response,omitempty" example:"OK"`
	Errors      *string                `json:"errors,omitempty" example:""`
}

type JobLogFilter struct {
	Status        *string    // Filter by status (e.g., "FAILED", "SUCCESS")
	TaskID        *string    // Filter by TaskID
	JobID         *int64     // Filter by ID
	SubmittedAt   *time.Time // Filter by submission date (exact or range)
	SubmittedFrom *time.Time // Range: submitted after this date
	SubmittedTo   *time.Time // Range: submitted before this date
	Page          int        // Page number (1-based)
	PageSize      int        // Items per page
}

// New creates a new JobLog with attached db handle.
func New(db *sqlx.DB, payload interface{}) (*JobLog, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	var jl JobLog
	query := `
		INSERT INTO submission_log (payload, status)
		VALUES ($1, 'queued')
		RETURNING id, submitted_at, payload, status, retry_count, last_attempt_at, task_id, response`
	err = db.Get(&jl, query, raw)
	if err != nil {
		return nil, err
	}
	jl.db = db
	return &jl, nil
}

// Load finds a JobLog by ID.
func Load(db *sqlx.DB, id int64) (*JobLog, error) {
	var jl JobLog
	err := db.Get(&jl, `
		SELECT id, submitted_at, payload, status, retry_count, last_attempt_at, task_id, response, errors
		FROM submission_log WHERE id = $1`, id)
	if err != nil {
		return nil, err
	}
	jl.db = db
	return &jl, nil
}

// UpdateTaskID updates the task ID for this job log.
func (jl *JobLog) UpdateTaskID(taskID string) error {
	_, err := jl.db.Exec(`UPDATE submission_log SET task_id = $1 WHERE id = $2`, taskID, jl.ID)
	if err == nil {
		jl.TaskID = sql.NullString{String: taskID, Valid: true}
	}
	return err
}

// UpdateStatusAndResponse updates the job log status and DHIS2 response.
func (jl *JobLog) UpdateStatusAndResponse(status, response string) error {
	_, err := jl.db.Exec(
		`UPDATE submission_log SET status = $1, response = $2, last_attempt_at = NOW() WHERE id = $3`,
		status, response, jl.ID,
	)
	if err == nil {
		jl.Status = status
		jl.Response = sql.NullString{String: response, Valid: true}
	}
	return err
}

// UpdateStatusAndErrors updates the job log status and error messages.
func (jl *JobLog) UpdateStatusAndErrors(status, errors string) error {
	_, err := jl.db.Exec(
		`UPDATE submission_log SET status = $1, errors = $2, last_attempt_at = NOW() WHERE id = $3`,
		status, errors, jl.ID,
	)
	if err == nil {
		jl.Status = status
		jl.Errors = sql.NullString{String: errors, Valid: true}
	}
	return err
}

// UpdateErrors updates the job log with error messages.
func (jl *JobLog) UpdateErrors(errors string) error {
	_, err := jl.db.Exec(
		`UPDATE submission_log SET errors = $1, last_attempt_at = NOW() WHERE id = $2`,
		errors, jl.ID,
	)
	if err == nil {
		jl.Errors = sql.NullString{String: errors, Valid: true}
	}
	return err
}

// UpdateResponse updates the job log with a DHIS2 response.
func (jl *JobLog) UpdateResponse(response string) error {
	_, err := jl.db.Exec(
		`UPDATE submission_log SET response = $1, last_attempt_at = NOW() WHERE id = $2`,
		response, jl.ID,
	)
	if err == nil {
		jl.Response = sql.NullString{String: response, Valid: true}
	}
	return err
}

// IncrementRetry increments the retry count and resets the status to "queued".
func (jl *JobLog) IncrementRetry() error {
	_, err := jl.db.Exec(
		`UPDATE submission_log SET retry_count = retry_count + 1, status = 'queued', last_attempt_at = NOW() WHERE id = $1`,
		jl.ID,
	)
	if err == nil {
		jl.RetryCount++
		jl.Status = "queued"
	}
	return err
}

// ListFailed returns all failed jobs for reprocessing.
func ListFailed(db *sqlx.DB) ([]*JobLog, error) {
	var jobs []*JobLog
	err := db.Select(&jobs, `
		SELECT id, submitted_at, payload, status, retry_count, last_attempt_at, task_id, response, errors
		FROM submission_log WHERE status = 'failed'
		ORDER BY submitted_at DESC
	`)
	for _, jl := range jobs {
		jl.db = db
	}
	return jobs, err
}

// GetByID retrieves a JobLog by its ID and attaches the db handle for further operations.
func GetByID(db *sqlx.DB, id int64) (*JobLog, error) {
	var jl JobLog
	err := db.Get(&jl, `
		SELECT id, submitted_at, payload, status, retry_count, last_attempt_at, task_id, response, errors
		FROM submission_log WHERE id = $1`, id)
	if err != nil {
		return nil, err
	}
	jl.db = db
	return &jl, nil
}

// GetByTaskID retrieves a JobLog by its Asynq TaskID and attaches the db handle for further operations.
func GetByTaskID(db *sqlx.DB, taskID string) (*JobLog, error) {
	var jl JobLog
	err := db.Get(&jl, `
		SELECT id, submitted_at, payload, status, retry_count, last_attempt_at, task_id, response, errors
		FROM submission_log WHERE task_id = $1`, taskID)
	if err != nil {
		return nil, err
	}
	jl.db = db
	return &jl, nil
}

// GetLogs retrieves job logs based on the provided filter criteria.
func GetLogs(db *sqlx.DB, filter JobLogFilter) ([]JobLog, int, error) {
	var (
		logs   []JobLog
		args   []interface{}
		where  []string
		query  = `SELECT * FROM submission_log`
		countQ = `SELECT COUNT(*) FROM submission_log`
	)

	// Build WHERE clause dynamically
	if filter.Status != nil {
		where = append(where, fmt.Sprintf("status = $%d", len(args)+1))
		args = append(args, *filter.Status)
	}
	if filter.TaskID != nil {
		where = append(where, fmt.Sprintf("task_id = $%d", len(args)+1))
		args = append(args, *filter.TaskID)
	}
	if filter.JobID != nil {
		where = append(where, fmt.Sprintf("id = $%d", len(args)+1))
		args = append(args, *filter.JobID)
	}
	if filter.SubmittedAt != nil {
		where = append(where, fmt.Sprintf("submitted_at = $%d", len(args)+1))
		args = append(args, *filter.SubmittedAt)
	}
	if filter.SubmittedFrom != nil {
		where = append(where, fmt.Sprintf("submitted_at >= $%d", len(args)+1))
		args = append(args, *filter.SubmittedFrom)
	}
	if filter.SubmittedTo != nil {
		where = append(where, fmt.Sprintf("submitted_at <= $%d", len(args)+1))
		args = append(args, *filter.SubmittedTo)
	}

	if len(where) > 0 {
		cond := " WHERE " + strings.Join(where, " AND ")
		query += cond
		countQ += cond
	}

	// Pagination defaults
	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	// Order by latest submitted
	query += " ORDER BY submitted_at DESC"
	query += fmt.Sprintf(" LIMIT %d OFFSET %d", pageSize, offset)

	log.Info("QUERY: ", query, " ARGS: ", args)
	// Get total count (for pagination UI)
	var total int
	if err := db.Get(&total, countQ, args...); err != nil {
		return nil, 0, err
	}

	// Get page of logs
	if err := db.Select(&logs, query, args...); err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}
