package cli

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMCPHTTPRequestsInheritRootCancellation(t *testing.T) {
	root, cancelRoot := context.WithCancel(context.Background())
	defer cancelRoot()
	started := make(chan struct{})
	cancelled := make(chan struct{})
	handler := http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		close(started)
		<-request.Context().Done()
		close(cancelled)
	})
	server := configuredMCPHTTPServer(root, "", handler)
	testServer := httptest.NewUnstartedServer(handler)
	testServer.Config = server
	testServer.Start()
	defer testServer.Close()
	shutdownMCPServerOnCancel(root, server)

	requestFinished := make(chan error, 1)
	go func() {
		response, err := testServer.Client().Get(testServer.URL)
		if response != nil {
			_, _ = io.Copy(io.Discard, response.Body)
			_ = response.Body.Close()
		}
		requestFinished <- err
	}()
	deadline, cancelDeadline := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelDeadline()
	awaitMCPContextSignal(t, deadline, started, "request start")
	cancelRoot()
	awaitMCPContextSignal(t, deadline, cancelled, "request cancellation")
	awaitMCPRequestCompletion(t, deadline, requestFinished)
}

func awaitMCPContextSignal(t *testing.T, deadline context.Context, signal <-chan struct{}, want string) {
	t.Helper()
	select {
	case <-signal:
	case <-deadline.Done():
		t.Fatalf("timed out waiting for %s: %v", want, deadline.Err())
	}
}

func awaitMCPRequestCompletion(t *testing.T, deadline context.Context, finished <-chan error) {
	t.Helper()
	select {
	case err := <-finished:
		if err != nil {
			t.Fatalf("in-flight request did not finish cleanly: %v", err)
		}
	case <-deadline.Done():
		t.Fatalf("timed out waiting for request completion: %v", deadline.Err())
	}
}
