package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReadAPIError_openAIStyle(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, `{"error":{"message":"model not found"}}`)
	}))
	defer srv.Close()

	client := &Client{
		cli: srv.Client(),
		url: srv.URL,
	}

	_, err := client.CallLLMStream(context.Background(), map[string]any{
		"model": "test",
		"messages": []map[string]any{
			{"role": "user", "content": "hi"},
		},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "model not found") {
		t.Fatalf("error = %q, want model not found", err.Error())
	}
}

func TestSSEData(t *testing.T) {
	data, ok := sseData("data: {\"id\":\"1\"}")
	if !ok || data != `{"id":"1"}` {
		t.Fatalf("sseData = %q, %v", data, ok)
	}

	_, ok = sseData("data: [DONE]")
	if ok {
		t.Fatal("expected [DONE] to be skipped")
	}

	_, ok = sseData("not sse")
	if ok {
		t.Fatal("expected non-sse line to be skipped")
	}
}
