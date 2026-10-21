package main

import (
	"encoding/json"
	"testing"
)

func TestMiddlewareRequestResponseBodyPersistence(t *testing.T) {
	body, err := json.Marshal(middlewareRequest{
		ResponseBodyPersistence: defaultResponseBodyPersistence,
	})
	if err != nil {
		t.Fatalf("marshal middleware request: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode middleware request: %v", err)
	}
	if payload["responseBodyPersistence"] != "save" {
		t.Fatalf("responseBodyPersistence = %v, want save", payload["responseBodyPersistence"])
	}
}

func TestMiddlewareRequestMetadata(t *testing.T) {
	body, err := json.Marshal(middlewareRequest{
		Metadata: map[string]any{"dataSet": "dataset-uid"},
	})
	if err != nil {
		t.Fatalf("marshal middleware request: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode middleware request: %v", err)
	}
	metadata, ok := payload["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("metadata = %T, want object", payload["metadata"])
	}
	if metadata["dataSet"] != "dataset-uid" {
		t.Fatalf("metadata.dataSet = %v, want dataset-uid", metadata["dataSet"])
	}
}

func TestMiddlewareRequestOmitsEmptyMetadata(t *testing.T) {
	body, err := json.Marshal(middlewareRequest{})
	if err != nil {
		t.Fatalf("marshal middleware request: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode middleware request: %v", err)
	}
	if _, ok := payload["metadata"]; ok {
		t.Fatalf("metadata should be omitted when empty")
	}
}

func TestMetadataForDataSet(t *testing.T) {
	metadata := metadataForDataSet(" dataset-uid ")
	if metadata["dataSet"] != "dataset-uid" {
		t.Fatalf("metadata.dataSet = %v, want dataset-uid", metadata["dataSet"])
	}
	if metadataForDataSet(" ") != nil {
		t.Fatalf("empty data set should omit metadata")
	}
}
