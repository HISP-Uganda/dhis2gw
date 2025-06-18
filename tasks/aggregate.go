package tasks

import (
	"context"
	"dhis2gw/models"
	"github.com/goccy/go-json"
	"github.com/hibiken/asynq"
)

const (
	TypeAggregate = "aggregate:send"
)

type AggregateTaskPayload struct {
	LogID   int64 `json:"log_id"`
	Payload models.AggregateRequest
}

func NewAggregateTask(aggRequest AggregateTaskPayload) (*asynq.Task, error) {
	// No payload needed for aggregate tasks
	payload, err := json.Marshal(aggRequest)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeAggregate, payload, asynq.MaxRetry(3)), nil
}

func HandleAggregateTask(ctx context.Context, task *asynq.Task) error {
	var aggRequest AggregateTaskPayload
	if err := json.Unmarshal(task.Payload(), &aggRequest); err != nil {
		return err
	}

	// Process the aggregate request
	// log job in DB
	if err := aggRequest.Process(); err != nil {
		return err
	}

	return nil
}

func (p *AggregateTaskPayload) Process() error {
	// Here you would implement the logic to process the aggregate request
	// For example, sending data to DHIS2 or performing some aggregation logic

	// This is a placeholder for actual processing logic
	// You can replace this with your own implementation
	// return p.Payload.SendToDHIS2()
	return nil
}
