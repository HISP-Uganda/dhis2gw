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
