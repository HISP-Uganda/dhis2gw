package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"dhis2gw/internal/config"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/HISP-Uganda/go-dhis2-sdk/dhis2/schema"
	log "github.com/sirupsen/logrus"
)

type ExtendedDataValue struct {
	schema.DataValue // embed the original type

	CategoryCombo  *string `json:"categoryCombo,omitempty"`
	CategoryOption *string `json:"categoryOption,omitempty"`
}

type Comment struct {
	Explanation string   `json:"explanation"`
	Attachment  []string `json:"attachment"`
}

func SendDataValue(context context.Context, dataValue ExtendedDataValue) error {
	if dataValue.CategoryCombo == nil {
		return errors.New("CategoryCombo cannot be nil")
	}
	params := map[string]string{
		"de": *dataValue.DataElement,
		"pe": *dataValue.Period,
		"ou": *dataValue.OrgUnit,
		"co": *dataValue.CategoryOptionCombo, // default
		"cc": *dataValue.CategoryCombo,
		"cp": *dataValue.CategoryOption,
		// "value": *dataValue.Value,
	}
	if dataValue.Value != nil && *dataValue.Value != "" {
		params["value"] = *dataValue.Value
	}
	if dataValue.Comment != nil && *dataValue.Comment != "" {
		encoded, _ := EncodeComment(*dataValue.Comment, []string{})
		params["comment"] = encoded
	}
	log.Infof("Sending data value through SukumadPro: %v", params)
	return submitMiddlewareRequest(context, params)
}

type middlewareRequest struct {
	SourceSystem            string `json:"sourceSystem"`
	DestinationServerUID    string `json:"destinationServerUid"`
	CorrelationID           string `json:"correlationId"`
	IdempotencyKey          string `json:"idempotencyKey"`
	PayloadFormat           string `json:"payloadFormat"`
	SubmissionBinding       string `json:"submissionBinding"`
	ResponseBodyPersistence string `json:"responseBodyPersistence"`
	URLSuffix               string `json:"urlSuffix"`
	Payload                 any    `json:"payload"`
}

const defaultResponseBodyPersistence = "save"

func submitMiddlewareRequest(ctx context.Context, payload any) error {
	cfg := config.MustGet().Config.PBS.Middleware
	if strings.TrimSpace(cfg.URL) == "" {
		return errors.New("PBS middleware URL is not configured")
	}
	if strings.TrimSpace(cfg.APIToken) == "" {
		return errors.New("PBS middleware API token is not configured")
	}
	if strings.TrimSpace(cfg.SourceSystem) == "" {
		return errors.New("PBS middleware source system is not configured")
	}
	if strings.TrimSpace(cfg.DestinationServerUID) == "" {
		return errors.New("PBS middleware destination server UID is not configured")
	}
	if strings.TrimSpace(cfg.URLSuffix) == "" {
		return errors.New("PBS middleware URL suffix is not configured")
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal middleware payload: %w", err)
	}
	hashInput := struct {
		SourceSystem         string          `json:"sourceSystem"`
		DestinationServerUID string          `json:"destinationServerUid"`
		URLSuffix            string          `json:"urlSuffix"`
		Payload              json.RawMessage `json:"payload"`
	}{cfg.SourceSystem, cfg.DestinationServerUID, cfg.URLSuffix, payloadJSON}
	hashJSON, err := json.Marshal(hashInput)
	if err != nil {
		return fmt.Errorf("marshal middleware idempotency input: %w", err)
	}
	hash := sha256.Sum256(hashJSON)
	correlationID, err := newCorrelationID()
	if err != nil {
		return fmt.Errorf("generate middleware correlation ID: %w", err)
	}
	body := middlewareRequest{
		SourceSystem: cfg.SourceSystem, DestinationServerUID: cfg.DestinationServerUID,
		CorrelationID: correlationID, IdempotencyKey: "pbs-sync-" + hex.EncodeToString(hash[:]),
		PayloadFormat: "json", SubmissionBinding: "query", ResponseBodyPersistence: defaultResponseBodyPersistence, URLSuffix: cfg.URLSuffix, Payload: payload,
	}
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal middleware request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.URL, strings.NewReader(string(bodyJSON)))
	if err != nil {
		return fmt.Errorf("create middleware request: %w", err)
	}
	req.Header.Set("X-API-Token", cfg.APIToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return fmt.Errorf("submit request to SukumadPro: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		responseBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return fmt.Errorf("SukumadPro returned error: status=%d body=%s", resp.StatusCode, responseBody)
	}
	return nil
}

func newCorrelationID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%s-%s-%s-%s-%s", hex.EncodeToString(b[0:4]), hex.EncodeToString(b[4:6]), hex.EncodeToString(b[6:8]), hex.EncodeToString(b[8:10]), hex.EncodeToString(b[10:16])), nil
}

func EncodeComment(explanation string, attachments []string) (string, error) {
	p := Comment{
		Explanation: explanation,
		Attachment:  attachments,
	}

	j, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	return string(j), nil
}

func PushIndividualDataValues(ctx context.Context, dataValues []ExtendedDataValue) error {
	if len(dataValues) == 0 {
		return nil
	}
	err := SendWithWorkers(ctx, dataValues, 5, SendDataValue)
	if err != nil {
		log.WithError(err).Error("SendWithWorkers: failed to submit data value through SukumadPro")
		return err
	}

	return nil
}

func SendWithWorkers[T any](
	ctx context.Context,
	items []T,
	workers int,
	sender func(context.Context, T) error,
) error {
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup
	errCh := make(chan error, len(items))

loop:
	for i, item := range items {
		if err := ctx.Err(); err != nil {
			break
		}

		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			break loop
		}

		wg.Add(1)
		go func(i int, item T) {
			defer wg.Done()
			defer func() { <-sem }()

			if err := ctx.Err(); err != nil {
				errCh <- fmt.Errorf("index %d: %w", i, err)
				return
			}

			if err := sender(ctx, item); err != nil {
				errCh <- fmt.Errorf("index %d: %w", i, err)
			}
		}(i, item)
	}

	wg.Wait()
	close(errCh)

	if len(errCh) > 0 {
		return <-errCh // return first error
	}

	return ctx.Err()
}
