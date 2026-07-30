package main

import (
	"bufio"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const goHTTPHeaderReadSlop = 4 << 10

func TestHTTPServerHasBoundedTransportSettings(t *testing.T) {
	handler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	server := newHTTPServer("127.0.0.1:0", handler)

	if server.ReadHeaderTimeout != httpReadHeaderTimeout ||
		server.ReadTimeout != httpReadTimeout ||
		server.WriteTimeout != httpWriteTimeout ||
		server.IdleTimeout != httpIdleTimeout ||
		server.MaxHeaderBytes != httpMaxHeaderBytes {
		t.Fatalf(
			"server bounds = header %s read %s write %s idle %s max %d",
			server.ReadHeaderTimeout,
			server.ReadTimeout,
			server.WriteTimeout,
			server.IdleTimeout,
			server.MaxHeaderBytes,
		)
	}
}

func TestHTTPServerReadHeaderTimeoutClosesSlowHeaderBeforeHandler(t *testing.T) {
	var handled atomic.Bool
	limits := shortTestHTTPServerLimits()
	limits.readHeaderTimeout = 60 * time.Millisecond
	_, address := startTestHTTPServer(t, limits, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		handled.Store(true)
		w.WriteHeader(http.StatusNoContent)
	}))

	connection := dialTestHTTPServer(t, address)
	defer connection.Close()
	started := time.Now()
	if _, err := io.WriteString(connection, "GET / HTTP/1.1\r\nHost: localhost\r\nX-Slow: "); err != nil {
		t.Fatal(err)
	}
	if err := connection.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}
	statusLine, err := bufio.NewReader(connection).ReadString('\n')
	if err == nil && strings.Contains(statusLine, " 2") {
		t.Fatalf("slow-header request unexpectedly succeeded: %q", statusLine)
	}
	if elapsed := time.Since(started); elapsed < limits.readHeaderTimeout/2 {
		t.Fatalf("slow-header connection closed before configured timeout: %s", elapsed)
	}
	if handled.Load() {
		t.Fatal("slow-header request reached the application handler")
	}
}

func TestHTTPServerReadTimeoutInterruptsIncompleteBody(t *testing.T) {
	limits := shortTestHTTPServerLimits()
	limits.readTimeout = 80 * time.Millisecond
	readResult := make(chan error, 1)
	_, address := startTestHTTPServer(t, limits, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := io.ReadAll(r.Body)
		readResult <- err
		http.Error(w, "request timed out", http.StatusRequestTimeout)
	}))

	connection := dialTestHTTPServer(t, address)
	defer connection.Close()
	if _, err := io.WriteString(
		connection,
		"POST / HTTP/1.1\r\nHost: localhost\r\nContent-Length: 100\r\nConnection: close\r\n\r\n",
	); err != nil {
		t.Fatal(err)
	}
	if err := connection.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}
	statusLine, err := bufio.NewReader(connection).ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(statusLine, " 408 ") {
		t.Fatalf("incomplete-body status line = %q", statusLine)
	}
	select {
	case readErr := <-readResult:
		var networkError net.Error
		if !errors.As(readErr, &networkError) || !networkError.Timeout() {
			t.Fatalf("body read error = %v, want network timeout", readErr)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("handler body read was not interrupted")
	}
}

func TestHTTPServerWriteTimeoutStopsLateResponse(t *testing.T) {
	limits := shortTestHTTPServerLimits()
	limits.writeTimeout = 50 * time.Millisecond
	flushResult := make(chan error, 1)
	_, address := startTestHTTPServer(t, limits, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(3 * limits.writeTimeout)
		w.Header().Set("Content-Type", "text/plain")
		_, _ = io.WriteString(w, strings.Repeat("late", 4096))
		flushResult <- http.NewResponseController(w).Flush()
	}))

	connection := dialTestHTTPServer(t, address)
	defer connection.Close()
	if _, err := io.WriteString(
		connection,
		"GET / HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n",
	); err != nil {
		t.Fatal(err)
	}
	if err := connection.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}
	statusLine, err := bufio.NewReader(connection).ReadString('\n')
	if err == nil && strings.Contains(statusLine, " 200 ") {
		t.Fatalf("late response escaped WriteTimeout: %q", statusLine)
	}
	select {
	case flushErr := <-flushResult:
		if flushErr == nil {
			t.Fatal("late response flush succeeded after WriteTimeout")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("late handler did not attempt its response")
	}
}

func TestHTTPServerIdleTimeoutClosesKeepAliveConnection(t *testing.T) {
	limits := shortTestHTTPServerLimits()
	limits.idleTimeout = 60 * time.Millisecond
	var handled atomic.Int32
	_, address := startTestHTTPServer(t, limits, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		handled.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))

	connection := dialTestHTTPServer(t, address)
	defer connection.Close()
	reader := bufio.NewReader(connection)
	request := "GET / HTTP/1.1\r\nHost: localhost\r\nConnection: keep-alive\r\n\r\n"
	if _, err := io.WriteString(connection, request); err != nil {
		t.Fatal(err)
	}
	first, err := http.ReadResponse(reader, &http.Request{Method: http.MethodGet})
	if err != nil {
		t.Fatal(err)
	}
	_ = first.Body.Close()
	if first.StatusCode != http.StatusNoContent {
		t.Fatalf("first status = %d", first.StatusCode)
	}

	time.Sleep(3 * limits.idleTimeout)
	if err := connection.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}
	_, writeErr := io.WriteString(connection, request)
	if writeErr == nil {
		if second, readErr := http.ReadResponse(reader, &http.Request{Method: http.MethodGet}); readErr == nil {
			_ = second.Body.Close()
			t.Fatalf("idle connection accepted a second request: %s", second.Status)
		}
	}
	if got := handled.Load(); got != 1 {
		t.Fatalf("handled requests = %d, want 1", got)
	}
}

func TestHTTPServerRejectsHeadersBeyondEffectiveLimitBeforeHandler(t *testing.T) {
	var handled atomic.Bool
	limits := shortTestHTTPServerLimits()
	limits.maxHeaderBytes = 1 << 10
	_, address := startTestHTTPServer(t, limits, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		handled.Store(true)
		w.WriteHeader(http.StatusNoContent)
	}))

	connection := dialTestHTTPServer(t, address)
	defer connection.Close()
	if err := connection.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}
	// Go reserves 4 KiB beyond MaxHeaderBytes for the buffered reader. Sending
	// more than configured limit + that parser allowance guarantees rejection.
	request := "GET / HTTP/1.1\r\nHost: localhost\r\nX-Oversized: " +
		strings.Repeat("a", limits.maxHeaderBytes+goHTTPHeaderReadSlop+1) +
		"\r\n\r\n"
	if _, err := io.WriteString(connection, request); err != nil {
		t.Fatal(err)
	}
	statusLine, err := bufio.NewReader(connection).ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(statusLine, " 431 ") {
		t.Fatalf("oversized-header status line = %q", statusLine)
	}
	if handled.Load() {
		t.Fatal("oversized request reached the application handler")
	}
}

func shortTestHTTPServerLimits() httpServerLimits {
	return httpServerLimits{
		readHeaderTimeout: 2 * time.Second,
		readTimeout:       2 * time.Second,
		writeTimeout:      2 * time.Second,
		idleTimeout:       2 * time.Second,
		maxHeaderBytes:    8 << 10,
	}
}

func startTestHTTPServer(
	t *testing.T,
	limits httpServerLimits,
	handler http.Handler,
) (*http.Server, string) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := newHTTPServerWithLimits("", handler, limits)
	go func() {
		_ = server.Serve(listener)
	}()
	t.Cleanup(func() {
		_ = server.Close()
		_ = listener.Close()
	})
	return server, listener.Addr().String()
}

func dialTestHTTPServer(t *testing.T, address string) net.Conn {
	t.Helper()
	connection, err := net.DialTimeout("tcp", address, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	return connection
}
