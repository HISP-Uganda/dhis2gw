package joblog

import (
	"database/sql"
	"encoding/json"
	"github.com/jmoiron/sqlx"
	"time"
)

type JobLog struct {
	ID          int64           `db:"id"`
	Submitted   time.Time       `db:"submitted_at"`
	Payload     json.RawMessage `db:"payload"`
	Status      string          `db:"status"`
	RetryCount  int             `db:"retry_count"`
	LastAttempt sql.NullTime    `db:"last_attempt_at"`
	TaskID      sql.NullString  `db:"task_id"`
	Response    sql.NullString  `db:"response"`
	Errors      sql.NullString  `db:"errors"` // Optional field for storing error messages

	db *sqlx.DB // not persisted, for method receivers
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
