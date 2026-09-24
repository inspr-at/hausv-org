package ai

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestValidateDestination(t *testing.T) {
	accepted := []string{
		"",
		"https://audit.example.invalid/v1",
		"https://example.com/v1",
		"HTTPS://Example.COM/v1",
		"https://192.168.1.1/v1",
		"http://127.0.0.1:11434/v1",
		"http://127.0.0.2/v1",
		"http://10.1.2.3/v1",
		"http://172.16.0.1/v1",
		"http://172.31.255.255/v1",
		"http://192.168.8.10:11434/v1",
		"http://[::1]/v1",
		"http://[fc00::1]/v1",
		"http://[fd12:3456::abcd]/v1",
		"http://[::ffff:10.1.2.3]/v1",
		"http://localhost/v1",
		"http://LOCALHOST:11434/v1",
		"http://mac-studio.local:11434/v1",
		"http://Studio.LOCAL./v1",
		"http://nas.lan/v1",
	}
	for _, raw := range accepted {
		if err := ValidateDestination(raw); err != nil {
			t.Errorf("ValidateDestination(%q) = %v", raw, err)
		}
	}

	rejected := []struct {
		raw  string
		want error
	}{
		{raw: "https://user:pass@example.test/v1", want: ErrDestinationUserinfo},
		{raw: "https://example.test/v1#frag", want: ErrDestinationFragment},
		{raw: "https://example.test/v1#", want: ErrDestinationFragment},
		{raw: "ftp://example.test/v1", want: ErrDestinationScheme},
		{raw: "file:///tmp/x", want: ErrDestinationScheme},
		{raw: "http:///v1", want: ErrDestinationHost},
		{raw: "http://8.8.8.8/v1", want: ErrDestinationHTTP},
		{raw: "http://example.com/v1", want: ErrDestinationHTTP},
		{raw: "http://172.15.5.5/v1", want: ErrDestinationHTTP},
		{raw: "http://172.32.0.1/v1", want: ErrDestinationHTTP},
		{raw: "http://169.254.169.254/latest", want: ErrDestinationHTTP},
		{raw: "http://[fe80::1]/v1", want: ErrDestinationHTTP},
		{raw: "http://[::ffff:8.8.8.8]/v1", want: ErrDestinationHTTP},
		{raw: "http://evil.local.com/v1", want: ErrDestinationHTTP},
		{raw: "http://localhost.evil.com/v1", want: ErrDestinationHTTP},
		{raw: "http://2130706433/v1", want: ErrDestinationHTTP},
		{raw: "http://host:abc/v1", want: ErrDestinationHost},
	}
	for _, test := range rejected {
		err := ValidateDestination(test.raw)
		if err == nil {
			t.Errorf("ValidateDestination(%q) accepted", test.raw)
			continue
		}
		if !errors.Is(err, test.want) {
			t.Errorf("ValidateDestination(%q) = %v, want %v", test.raw, err, test.want)
		}
	}
}

func TestOriginNormalisesSchemeHostAndPort(t *testing.T) {
	left, ok := Origin("https://OpenRouter.ai/api/v1")
	if !ok || left != "https://openrouter.ai:443" {
		t.Fatalf("origin = %q, ok=%v", left, ok)
	}
	if !SameOrigin("https://OpenRouter.ai/api/v1", "https://openrouter.ai:443/other") {
		t.Fatal("default https port did not match")
	}
	if !SameOrigin("http://studio.local./v1", "http://Studio.local:80/x") {
		t.Fatal("trailing dot or case did not match")
	}
	if !SameOrigin("https://[::1]/v1", "https://[::1]:443/x") {
		t.Fatal("ipv6 default port did not match")
	}
	if SameOrigin("https://openrouter.ai/v1", "https://openrouter.ai:8443/v1") {
		t.Fatal("different ports matched")
	}
	if SameOrigin("http://openrouter.ai/v1", "https://openrouter.ai/v1") {
		t.Fatal("different schemes matched")
	}
	if _, ok := Origin("http://host:abc/v1"); ok {
		t.Fatal("invalid port produced an origin")
	}
}

func TestNewFromEnvEnforcesOrganisationDestinationOnlyWhenAsked(t *testing.T) {
	operator, err := NewFromEnv(envMap(map[string]string{
		"AI_BASE_URL": "http://8.8.8.8/v1",
		"AI_MODEL":    "demo",
	}))
	if err != nil || operator == nil {
		t.Fatalf("operator destination rejected: %v", err)
	}
	_, err = NewFromEnv(envMap(map[string]string{
		"AI_BASE_URL":                   "http://8.8.8.8/v1",
		"AI_MODEL":                      "demo",
		"AI_ENFORCE_DESTINATION_POLICY": "1",
	}))
	if err == nil || !errors.Is(err, ErrDestinationHTTP) {
		t.Fatalf("organisation policy error = %v", err)
	}
}

func TestNewFromEnvSendsProcessKeyOnlyToItsOwnClient(t *testing.T) {
	const key = "audit-synthetic-nonsecret"
	previous := http.DefaultClient
	t.Cleanup(func() { http.DefaultClient = previous })
	var auth string
	var calls int
	http.DefaultClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		auth = r.Header.Get("Authorization")
		return &http.Response{StatusCode: http.StatusBadRequest, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("audit rejection")), Request: r}, nil
	})}

	suggester, err := NewFromEnv(envMap(map[string]string{
		"AI_BASE_URL": "https://openrouter.ai/api/v1",
		"AI_MODEL":    "demo",
		"AI_API_KEY":  key,
	}))
	if err != nil || suggester == nil {
		t.Fatal(err)
	}
	_, _ = suggester.Suggest(context.Background(), testInput())
	if calls != 1 || auth != "Bearer "+key {
		t.Fatalf("calls=%d authorization=%q", calls, auth)
	}
}

func TestCrossOriginRedirectIsRefused(t *testing.T) {
	const key = "audit-synthetic-nonsecret"
	previous := http.DefaultClient
	t.Cleanup(func() { http.DefaultClient = previous })
	var calls int
	var hosts []string
	http.DefaultClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		hosts = append(hosts, r.URL.Host)
		header := make(http.Header)
		header.Set("Location", "https://audit.example.invalid/v1/chat/completions")
		return &http.Response{StatusCode: http.StatusTemporaryRedirect, Header: header, Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
	})}

	suggester, err := NewFromEnv(envMap(map[string]string{
		"AI_BASE_URL": "https://openrouter.ai/api/v1",
		"AI_MODEL":    "demo",
		"AI_API_KEY":  key,
	}))
	if err != nil {
		t.Fatal(err)
	}
	_, err = suggester.Suggest(context.Background(), testInput())
	if err == nil || !strings.Contains(err.Error(), "cross-origin redirect refused") {
		t.Fatalf("error = %v", err)
	}
	if calls != 1 || len(hosts) != 1 || hosts[0] != "openrouter.ai" {
		t.Fatalf("redirect left the origin: calls=%d hosts=%v", calls, hosts)
	}
}

func TestSameOriginRedirectKeepsTheCredential(t *testing.T) {
	const key = "audit-synthetic-nonsecret"
	previous := http.DefaultClient
	t.Cleanup(func() { http.DefaultClient = previous })
	var calls int
	var auths []string
	http.DefaultClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		auths = append(auths, r.Header.Get("Authorization"))
		if calls == 1 {
			header := make(http.Header)
			header.Set("Location", "https://openrouter.ai/api/v1/chat/completions?continued=1")
			return &http.Response{StatusCode: http.StatusTemporaryRedirect, Header: header, Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
		}
		payload, err := json.Marshal(map[string]any{
			"choices": []any{map[string]any{"message": map[string]string{"content": validAnswerJSON()}}},
		})
		if err != nil {
			return nil, err
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(payload))), Request: r}, nil
	})}

	suggester, err := NewFromEnv(envMap(map[string]string{
		"AI_BASE_URL": "https://openrouter.ai/api/v1",
		"AI_MODEL":    "demo",
		"AI_API_KEY":  key,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := suggester.Suggest(context.Background(), testInput()); err != nil {
		t.Fatal(err)
	}
	if calls != 2 || auths[0] != "Bearer "+key || auths[1] != "Bearer "+key {
		t.Fatalf("calls=%d auths=%q", calls, auths)
	}
}
