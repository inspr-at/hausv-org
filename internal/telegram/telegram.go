// Package telegram is a minimal stdlib Bot API client: long-poll getUpdates
// plus sendMessage. No webhook, no framework — the bot needs exactly two
// verbs. The token never appears in errors or logs.
package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	token   string
	httpc   *http.Client
}

// New builds a client. baseURL is https://api.telegram.org in production and
// an httptest server in tests.
func New(baseURL, token string) *Client {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = "https://api.telegram.org"
	}
	return &Client{
		baseURL: baseURL,
		token:   strings.TrimSpace(token),
		// Long-poll requests block up to the poll timeout server-side; give
		// the transport generous headroom on top.
		httpc: &http.Client{Timeout: 90 * time.Second},
	}
}

func (c *Client) Configured() bool { return c != nil && c.token != "" }

type Update struct {
	UpdateID int64    `json:"update_id"`
	Message  *Message `json:"message"`
}

type Message struct {
	From *User  `json:"from"`
	Chat Chat   `json:"chat"`
	Text string `json:"text"`
	Date int64  `json:"date"`
}

type Chat struct {
	ID int64 `json:"id"`
}

type User struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
}

type apiResponse struct {
	OK          bool            `json:"ok"`
	Result      json.RawMessage `json:"result"`
	Description string          `json:"description"`
	Parameters  *struct {
		RetryAfter int `json:"retry_after"`
	} `json:"parameters"`
}

// GetUpdates long-polls for new updates starting at offset.
func (c *Client) GetUpdates(ctx context.Context, offset int64, timeout time.Duration) ([]Update, error) {
	if !c.Configured() {
		return nil, errors.New("telegram not configured")
	}
	seconds := int(timeout / time.Second)
	if seconds < 0 {
		seconds = 0
	}
	payload := map[string]any{
		"offset":          offset,
		"timeout":         seconds,
		"allowed_updates": []string{"message"},
	}
	raw, _, err := c.call(ctx, "getUpdates", payload)
	if err != nil {
		return nil, err
	}
	var updates []Update
	if err := json.Unmarshal(raw, &updates); err != nil {
		return nil, errors.New("telegram returned invalid updates")
	}
	return updates, nil
}

// SendMessage delivers text to one chat, retrying transient failures and
// honoring 429 retry_after.
func (c *Client) SendMessage(ctx context.Context, chatID int64, text string) error {
	if !c.Configured() {
		return errors.New("telegram not configured")
	}
	payload := map[string]any{
		"chat_id":                  chatID,
		"text":                     text,
		"disable_web_page_preview": true,
	}
	backoff := time.Second
	for attempt := 0; attempt < 3; attempt++ {
		_, retryAfter, err := c.call(ctx, "sendMessage", payload)
		if err == nil {
			return nil
		}
		if attempt == 2 {
			return err
		}
		wait := backoff
		if retryAfter > 0 {
			wait = time.Duration(retryAfter) * time.Second
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
		backoff *= 2
	}
	return errors.New("telegram send failed")
}

func (c *Client) call(ctx context.Context, method string, payload map[string]any) (json.RawMessage, int, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, 0, errors.New("telegram request encode failed")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/bot"+c.token+"/"+method, bytes.NewReader(body))
	if err != nil {
		return nil, 0, errors.New("telegram request build failed")
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpc.Do(req)
	if err != nil {
		// Deliberately generic: the wrapped error could embed the URL and
		// with it the token.
		return nil, 0, errors.New("telegram request failed")
	}
	defer resp.Body.Close()
	var decoded apiResponse
	dec := json.NewDecoder(io.LimitReader(resp.Body, 1<<20))
	if err := dec.Decode(&decoded); err != nil {
		return nil, 0, fmt.Errorf("telegram returned %d with invalid json", resp.StatusCode)
	}
	if !decoded.OK {
		retryAfter := 0
		if decoded.Parameters != nil {
			retryAfter = decoded.Parameters.RetryAfter
		}
		return nil, retryAfter, fmt.Errorf("telegram returned %d: %s", resp.StatusCode, decoded.Description)
	}
	return decoded.Result, 0, nil
}
