package resend

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/christianmscott/overwatch/pkg/spec"
)

func TestSendCallsResendAPI(t *testing.T) {
	var gotReq struct {
		From    string   `json:"from"`
		To      []string `json:"to"`
		Subject string   `json:"subject"`
		Text    string   `json:"text"`
	}
	var gotAuth string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &gotReq)
		w.WriteHeader(200)
		w.Write([]byte(`{"id":"test_123"}`))
	}))
	defer srv.Close()

	s := &Sender{
		apiKey:    "re_test_key",
		from:      "Overwatch <alerts@overwatch.dev>",
		recipient: "oncall@acme.io",
	}
	origURL := apiURL
	apiURL = srv.URL
	defer func() { apiURL = origURL }()

	msg := spec.AlertMessage{
		CheckName:      "api-health",
		Status:         spec.StatusDown,
		PreviousStatus: spec.StatusUp,
		Timestamp:      time.Date(2026, 4, 9, 12, 0, 0, 0, time.UTC),
		Detail:         "timeout after 30s",
	}

	if err := s.Send(context.Background(), msg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotAuth != "Bearer re_test_key" {
		t.Errorf("auth header = %q, want %q", gotAuth, "Bearer re_test_key")
	}
	if gotReq.From != "Overwatch <alerts@overwatch.dev>" {
		t.Errorf("from = %q", gotReq.From)
	}
	if len(gotReq.To) != 1 || gotReq.To[0] != "oncall@acme.io" {
		t.Errorf("to = %v", gotReq.To)
	}
	if gotReq.Subject != "[overwatch] api-health is down" {
		t.Errorf("subject = %q", gotReq.Subject)
	}
	if gotReq.Text == "" {
		t.Error("text body is empty")
	}
}

func TestSendReturnsErrorOnHTTPFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(422)
		w.Write([]byte(`{"message":"validation_error"}`))
	}))
	defer srv.Close()

	s := &Sender{apiKey: "re_test", from: "test@test.com", recipient: "x@x.com"}
	origURL := apiURL
	apiURL = srv.URL
	defer func() { apiURL = origURL }()

	err := s.Send(context.Background(), spec.AlertMessage{
		CheckName: "test",
		Status:    spec.StatusDown,
		Timestamp: time.Now(),
	})
	if err == nil {
		t.Fatal("expected error for 422 response")
	}
}
