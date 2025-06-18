package models

import (
	"database/sql"
	"dhis2gw/clients"
	"dhis2gw/config"
	"encoding/json"
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
	"time"
)

var (
	err      error
	Location *time.Location
)

func init() {
	Location, err = time.LoadLocation(config.DHIS2GWConf.Server.TimeZone)
	if err != nil {
		log.Errorln(err)
	}
}

type NullTime struct {
	sql.NullTime
}

// MarshalJSON Implement the json.Marshaller interface
func (nt *NullTime) MarshalJSON() ([]byte, error) {
	if !nt.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(nt.Time)
}

// UnmarshalJSON Implement the json.Unmarshaler interface
func (nt *NullTime) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		nt.Valid = false
		return nil
	}
	err := json.Unmarshal(data, &nt.Time)
	nt.Valid = err == nil
	return err
}

type Schedule struct {
	ID              int64           `db:"id" json:"id,omitempty"`
	ScheduleType    string          `db:"sched_type" json:"scheduleType"`
	Params          json.RawMessage `db:"params" json:"params,omitempty"` // Use json.RawMessage for JSON fields
	ScheduleContent string          `db:"sched_content" json:"scheduleContent,omitempty"`
	ScheduleURL     string          `db:"sched_url" json:"scheduleURL,omitempty"`
	Command         string          `db:"command" json:"command,omitempty"`
	CommandArgs     string          `db:"command_args" json:"commandArgs,omitempty"`
	FirstRunAt      NullTime        `db:"first_run_at" json:"firstRunAt,omitempty"`
	Repeat          string          `db:"repeat" json:"repeat,omitempty"`
	RepeatInterval  int             `db:"repeat_interval" json:"repeat_interval,"`
	CronExpression  string          `db:"cron_expression" json:"cronExpression,omitempty"`
	LastRunAt       NullTime        `db:"last_run_at" json:"lastRunAt,omitempty"` // Use pointer for nullable fields
	NextRunAt       time.Time       `db:"next_run_at" json:"nextRunAt,omitempty"`
	Status          string          `db:"status" json:"status,omitempty"`
	IsActive        bool            `db:"is_active" json:"isActive,omitempty"`
	RequestID       *RequestID      `db:"request_id" json:"request_id,omitempty"`
	ServerID        *ServerID       `db:"server_id" json:"serverID,omitempty"`
	ServerInCC      *bool           `db:"server_in_cc" json:"ServerInCC,omitempty"`
	AsyncJobType    string          `db:"async_job_type" json:"asyncJobType,omitempty"`
	AsyncJobID      string          `db:"async_jobid" json:"asyncJobID,omitempty"`
	CreatedBy       *int64          `db:"created_by" json:"createdBy,omitempty"` // Use pointer for nullable fields
	Created         time.Time       `db:"created" json:"created,omitempty"`
	Updated         time.Time       `db:"updated" json:"updated,omitempty"`
}

const createScheduleSQL = `INSERT INTO 
	schedules (sched_type, params, sched_url, sched_content, command, command_args,
		repeat, repeat_interval, cron_expression, next_run_at, status, is_active,
		async_job_type, async_jobid, request_id, server_id, server_in_cc,
		created_by, created, updated) 
	VALUES (
		:sched_type, :params, :sched_url, :sched_content, :command, :command_args,
		:repeat, :repeat_interval, :cron_expression, :next_run_at, :status, :is_active,
		:async_job_type, :async_jobid, :request_id, :server_id, :server_in_cc,
		:created_by, :created, :updated
	) RETURNING id`

// CreateSchedule inserts a new schedule into the database
func CreateSchedule(db *sqlx.DB, schedule Schedule) (int64, error) {
	rows, err := db.NamedQuery(createScheduleSQL, schedule)
	if err != nil {
		log.WithFields(
			log.Fields{"schedule": schedule, "Error": err}).
			Error("Failed to create schedule")
		return 0, err
	}
	for rows.Next() {
		var schedId int64
		_ = rows.Scan(&schedId)
		schedule.ID = schedId
	}
	_ = rows.Close()
	// id, _ = res.LastInsertId()
	return schedule.ID, nil
}

// CreateScheduleTx creates a new schedule in a transaction
func CreateScheduleTx(tx *sqlx.Tx, schedule Schedule) (int64, error) {
	// var id int64
	rows, err := tx.NamedQuery(createScheduleSQL, schedule)
	if err != nil {
		log.WithFields(
			log.Fields{"schedule": schedule, "Error": err}).
			Error("Failed to create schedule")
		return 0, err
	}
	for rows.Next() {
		var schedId int64
		_ = rows.Scan(&schedId)
		schedule.ID = schedId
	}
	_ = rows.Close()
	// id, _ = res.LastInsertId()
	return schedule.ID, nil
}

func ListSchedules(db *sqlx.DB) []Schedule {
	query := `SELECT * FROM schedules`
	var schedules []Schedule
	err := db.Select(&schedules, query)
	if err != nil {
		return []Schedule{}
	}
	return schedules
}

// GetSchedule retrieves a schedule from the database by ID
func GetSchedule(db *sqlx.DB, id int64) (Schedule, error) {
	query := `SELECT * FROM schedules WHERE id = $1`
	var schedule Schedule
	err := db.Get(&schedule, query, id)
	if err != nil {
		return Schedule{}, err
	}
	return schedule, nil
}

// GetScheduleForProcessing retrieves a schedule from the database by ID
func GetScheduleForProcessing(db *sqlx.DB, id int64) (Schedule, error) {
	query := `SELECT * FROM schedules WHERE id = $1 FOR UPDATE NOWAIT`
	var schedule Schedule
	err := db.Get(&schedule, query, id)
	if err != nil {
		return Schedule{}, err
	}
	return schedule, nil
}

const updateScheduleSQL = `UPDATE schedules SET
		sched_type = :sched_type, params = :params, sched_content = :sched_content, sched_url = :sched_url, 
		command = :command, command_args = :command_args, first_run_at = :first_run_at, repeat = :repeat,
		repeat_interval = :repeat_interval, cron_expression = :cron_expression,
        next_run_at = :next_run_at, last_run_at = :last_run_at,
		status = :status, is_active = :is_active, updated = :updated
	WHERE id = :id`

// UpdateSchedule updates an existing schedule in the database
func UpdateSchedule(db *sqlx.DB, schedule Schedule) error {
	_, err := db.NamedExec(updateScheduleSQL, schedule)
	return err
}

// UpdateScheduleTx updates an existing schedule in a transaction
func UpdateScheduleTx(tx *sqlx.Tx, schedule Schedule) error {
	_, err := tx.NamedExec(updateScheduleSQL, schedule)
	return err
}

// SetNextRun set the next time the schedule will run
func (s *Schedule) SetNextRun(tx *sqlx.Tx, nextRun time.Time) error {
	s.NextRunAt = nextRun
	_, err := tx.NamedExec(`UPDATE schedules SET next_run_at = :next_run_at WHERE id = :id`, s)
	return err
}

// SetLastRun set the last time the schedule is run
func (s *Schedule) SetLastRun(tx *sqlx.Tx) error {
	s.LastRunAt = NullTime{sql.NullTime{Time: time.Now().In(Location), Valid: true}}
	_, err := tx.NamedExec(`UPDATE schedules SET last_run_at = :last_run_at WHERE id = :id`, s)
	return err
}

// UpdateStatus updates the status of the schedule
func (s *Schedule) UpdateStatus(tx *sqlx.Tx, status string) error {
	s.Status = status
	s.Updated = time.Now().In(Location)
	_, err := tx.NamedExec(`UPDATE schedules SET status = :status, updated = :updated WHERE id = :id`, s)
	return err
}

// Deactivate deactivates the schedule
func (s *Schedule) Deactivate(tx *sqlx.Tx) error {
	s.IsActive = false
	s.Updated = time.Now().In(Location)
	_, err := tx.NamedExec(`UPDATE schedules SET is_active = :is_active, updated = :updated WHERE id = :id`, s)
	return err
}

// UpdateRunDetails updates the schedule run details
func (s *Schedule) UpdateRunDetails(tx *sqlx.Tx, status string, nextRun time.Time) error {
	s.LastRunAt = NullTime{sql.NullTime{Time: time.Now().In(Location), Valid: true}}
	s.Updated = time.Now().In(Location)
	s.Status = status
	s.NextRunAt = nextRun
	_, err := tx.NamedExec(
		`UPDATE schedules SET (status, next_run_at, last_run_at) = (:status, :next_run_at, :last_run_at) 
			WHERE id = :id`, s)
	return err
}

// DeleteSchedule deletes a schedule from the database by ID
func DeleteSchedule(db *sqlx.DB, id int64) error {
	query := `DELETE FROM schedules WHERE id = $1`
	_, err := db.Exec(query, id)
	return err
}

// ScheduleDue is a method on Schedule that returns whether the schedule can now run
func (s *Schedule) ScheduleDue() bool {
	return s.IsActive && (s.NextRunAt.Before(time.Now().In(Location)) || s.NextRunAt.Equal(time.Now().In(Location)))
}

func CreateAsyncJobSchedule(
	tx *sqlx.Tx,
	request RequestID,
	server ServerID,
	serverInCC bool,
	jobType string,
	jobID string,
) (int64, error) {
	repeatInterval := config.DHIS2GWConf.Server.Dhis2JobStatusCheckInterval
	var schedule = Schedule{
		Params:         []byte("{}"),
		ScheduleType:   "dhis2_async_job_check",
		Repeat:         "interval",
		RepeatInterval: repeatInterval,
		NextRunAt:      time.Now().Add(time.Second * time.Duration(repeatInterval)),
		Status:         "ready",
		IsActive:       true,
		RequestID:      &request,
		ServerID:       &server,
		ServerInCC:     &serverInCC,
		AsyncJobType:   jobType,
		AsyncJobID:     jobID,
		CreatedBy:      nil,
		Created:        time.Now().In(Location),
		Updated:        time.Now().In(Location),
	}
	return CreateScheduleTx(tx, schedule)

}

func CheckDhis2AsyncJobTaskSummary(tx *sqlx.Tx, schedule Schedule) (*AsyncJobImportSummary, error) {
	//if server, ok := ServerMap[fmt.Sprintf("%d", schedule.ServerID)]; ok {
	//
	//}
	var taskSummary *AsyncJobImportSummary
	if *schedule.ServerID > 0 {
		server := GetServerByID(int64(*schedule.ServerID))
		srv := clients.Server{
			BaseUrl:    server.URL(),
			Username:   server.Username(),
			Password:   server.Password(),
			AuthToken:  server.AuthToken(),
			AuthMethod: server.AuthMethod(),
		}
		client, err := srv.NewDhis2Client()
		if err != nil {
			log.WithFields(log.Fields{
				"server_id": *schedule.ServerID,
				"error":     err,
			}).Error("Could not get client for server!")
			return taskSummary, err
		}
		resource := fmt.Sprintf(
			"system/taskSummaries/%s/%s",
			schedule.AsyncJobType,
			schedule.AsyncJobID,
		)
		params := map[string]string{}
		resp, err := client.GetResource(resource, params)
		if err != nil {
			log.WithFields(log.Fields{
				"server_id":   *schedule.ServerID,
				"schedule_id": schedule.ID,
				"job_id":      schedule.AsyncJobID,
				"error":       err,
			}).Error("Could not get task summary for async job!")
			return taskSummary, err
		}
		// var taskSummary AsyncJobImportSummary
		err = json.Unmarshal(resp.Body(), &taskSummary)
		if err != nil {
			log.WithError(err).Error("Failed to unmarshall response taskSummary!")
			return taskSummary, err
		}
		//  log.WithField("TaskSummary", taskSummary).Info("TaskSummary")

	}
	return taskSummary, nil
}

// CheckDhis2AsyncJobStatus checks the status of an async job
func CheckDhis2AsyncJobStatus(schedule Schedule) (bool, bool, error) {
	if *schedule.ServerID > 0 {
		server := GetServerByID(int64(*schedule.ServerID))
		srv := clients.Server{
			BaseUrl:    server.URL(),
			Username:   server.Username(),
			Password:   server.Password(),
			AuthToken:  server.AuthToken(),
			AuthMethod: server.AuthMethod(),
		}
		client, err := srv.NewDhis2Client()
		if err != nil {
			log.WithFields(log.Fields{
				"server_id": *schedule.ServerID,
				"error":     err,
			}).Error("Could not get client for server!")
			return false, false, err
		}
		resource := fmt.Sprintf(
			"system/tasks/%s/%s",
			schedule.AsyncJobType,
			schedule.AsyncJobID,
		)
		params := map[string]string{}
		resp, err := client.GetResource(resource, params)
		if err != nil {
			log.WithFields(log.Fields{
				"server_id":   *schedule.ServerID,
				"schedule_id": schedule.ID,
				"job_id":      schedule.AsyncJobID,
				"error":       err,
			}).Error("Could not get task status for async job!")
			return false, false, err
		}
		var taskStatus []AsyncJobStatus
		err = json.Unmarshal(resp.Body(), &taskStatus)
		if err != nil {
			log.WithError(err).Error("Failed to unmarshall response taskStatus!")
			return false, false, err
		}
		// log.Infof("Task status for async job: %v", taskStatus)
		// use lo to find if any AsyncJobStatus in taskStatus slice has completed == true and return true, nil
		return len(lo.Filter(taskStatus, func(item AsyncJobStatus, _ int) bool {
			return item.Completed
		})) > 0, len(taskStatus) > 0, nil

	}

	return false, false, nil
}
