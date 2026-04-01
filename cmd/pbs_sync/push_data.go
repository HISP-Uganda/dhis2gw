package main

import (
	"context"
	"dhis2gw/clients"
	"errors"
	"fmt"
	"sync"

	"github.com/HISP-Uganda/go-dhis2-sdk/dhis2/schema"
	"github.com/goccy/go-json"
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
	client := clients.GetDhis2Client()
	if client == nil || client.RestClient == nil {
		return errors.New("DHIS2 client is not initialized")
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
	log.Infof("Sending data value to DHIS2: %v", params)

	resp, err := client.PostResource(
		"/dataValues", nil,
		clients.WithQuery(params),
		clients.WithContext(context),
	)
	if err != nil {
		return fmt.Errorf("failed to send data value to DHIS2: %w", err)
	}

	if resp == nil {
		return errors.New("nil response from DHIS2")
	}

	if resp.IsError() {
		return fmt.Errorf(
			"DHIS2 returned error: status=%d body=%s",
			resp.StatusCode(),
			resp.String(),
		)
	}
	return nil
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

func PushIndividualDataValues(dataValues []ExtendedDataValue) error {
	if len(dataValues) == 0 {
		return nil
	}
	client := clients.GetDhis2Client()
	if client == nil || client.RestClient == nil {
		return errors.New("DHIS2 client is not initialized")
	}

	ctx := context.Background()

	err := SendWithWorkers(ctx, dataValues, 5, SendDataValue)
	if err != nil {
		log.WithError(err).Error("SendWithWorkers: Failed to send data value to DHIS2")
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

	for i, item := range items {
		wg.Add(1)
		sem <- struct{}{}

		go func(i int, item T) {
			defer wg.Done()
			defer func() { <-sem }()

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

	return nil
}
