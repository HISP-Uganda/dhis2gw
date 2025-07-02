package tasks

import (
	"context"
	"dhis2gw/config"
	"dhis2gw/db"
	"dhis2gw/joblog"
	"dhis2gw/models"
	sdk "github.com/HISP-Uganda/go-dhis2-sdk"
	"github.com/goccy/go-json"
	"github.com/hibiken/asynq"
	log "github.com/sirupsen/logrus"
)

const (
	TypeAggregate = "aggregate:send"
)

var dhis2Client *sdk.Client

// SetClient should be called from main.go after initializing the client.
func SetClient(client *sdk.Client) {
	dhis2Client = client
}

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
	if err := aggRequest.Process(ctx); err != nil {
		return err
	}

	return nil
}

func (p *AggregateTaskPayload) Process(ctx context.Context) error {
	payload := p.Payload.ToDHIS2AggregatePayload()

	jl, err := joblog.Load(db.GetDB(), p.LogID)
	if err != nil {
		log.Printf("Failed to load job log: %v", err)
		return err
	}

	if jl.RetryCount > 0 {
		if err := jl.IncrementRetry(); err != nil {
			log.Printf("Failed to increment retry count: %v", err)
		}
	} else {
		dhis2Payload, marshalErr := json.Marshal(payload)
		if marshalErr != nil {
			log.Printf("Failed to marshal DHIS2 payload: %v", marshalErr)
			return marshalErr
		}
		if err := jl.UpdateDhis2Payload(string(dhis2Payload)); err != nil {
			log.Printf("Failed to update submission log with DHIS2 payload: %v", err)
		}
	}

	resp, err := dhis2Client.SendAggregateDataValues(ctx, &payload)
	status := "success"
	dhis2Resp := ""
	errors := ""

	if err != nil {
		log.Error("Error sending aggregate data values to DHIS2: ", err)
		status = "failed"
		errors = err.Error()
	} else {
		log.Info("Successfully sent aggregate data values to DHIS2")
		status = resp.Status
		rp, err := json.Marshal(resp)
		if err != nil {
			log.Error("Error marshalling DHIS2 response: ", err)
			status = "failed"
			errors = err.Error()
		} else {
			dhis2Resp = string(rp)
		}
	}

	_ = jl.UpdateStatusAndErrors(status, errors)

	if config.DHIS2GWConf.API.SaveResponse == "true" && dhis2Resp != "" {
		_ = jl.UpdateResponse(dhis2Resp)
	}

	log.WithFields(log.Fields{"ImportResponse": resp}).Info("Aggregate Import Response")
	return nil
}
