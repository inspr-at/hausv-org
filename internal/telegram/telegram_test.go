package telegram

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestGetUpdatesDecodesAndHidesToken(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		var payload map[string]any
		_ = json.NewDecoder(r.Body).Decode(&payload)
		if payload["offset"].(float64) != 42 {
			t.Errorf("offset = %v", payload["offset"])
		}
		_, _ = w.Write([]byte(`{"ok":true,"result":[{"update_id":43,"message":{"chat":{"id":7},"text":"/pp20status","from":{"id":1,"first_name":"J"}}}]}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "SECRET-TOKEN")
	updates, err := c.GetUpdates(context.Background(), 42, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if len(updates) != 1 || updates[0].UpdateID != 43 || updates[0].Message.Chat.ID != 7 {
		t.Fatalf("updates = %+v", updates)
	}
	if !strings.Contains(gotPath, "SECRET-TOKEN") {
		t.Fatalf("token must be in the request path, got %s", gotPath)
	}
}

func TestSendMessageRetriesOn429(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"ok":false,"description":"Too Many Requests","parameters":{"retry_after":0}}`))
			return
		}
		_, _ = w.Write([]byte(`{"ok":true,"result":{}}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "tok")
	if err := c.SendMessage(context.Background(), 7, "hi"); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 {
		t.Fatalf("calls = %d, want 2", calls.Load())
	}
}

func TestErrorsNeverContainToken(t *testing.T) {
	c := New("http://127.0.0.1:1", "SECRET-TOKEN")
	err := c.SendMessage(context.Background(), 7, "hi")
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), "SECRET-TOKEN") {
		t.Fatalf("token leaked into error: %v", err)
	}
}

func TestUnconfiguredClientRefuses(t *testing.T) {
	c := New("", "")
	if c.Configured() {
		t.Fatal("empty token must not be configured")
	}
	if _, err := c.GetUpdates(context.Background(), 0, time.Second); err == nil {
		t.Fatal("expected error")
	}
}
