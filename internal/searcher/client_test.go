// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package searcher

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestExecGetRetries(t *testing.T) {
	origBackoff := retryBackoff
	retryBackoff = func(int) time.Duration { return 0 }
	t.Cleanup(func() { retryBackoff = origBackoff })

	type result struct{ Name string }
	tests := []struct {
		name         string
		statuses     []int // status per attempt; the last one repeats
		wantRequests int32
		wantErr      error // nil means success
		wantAnyErr   bool
	}{
		{name: "success", statuses: []int{200}, wantRequests: 1},
		{name: "recovers after 5xx", statuses: []int{500, 503, 200}, wantRequests: 3},
		{name: "recovers after 429", statuses: []int{429, 200}, wantRequests: 2},
		{name: "gives up after retries", statuses: []int{502}, wantRequests: maxRetries + 1, wantErr: ErrUnavailable},
		{name: "404 is not retried", statuses: []int{404}, wantRequests: 1, wantErr: ErrNotFound},
		{name: "400 is not retried", statuses: []int{400}, wantRequests: 1, wantAnyErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var requests atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				n := int(requests.Add(1))
				status := test.statuses[min(n, len(test.statuses))-1]
				w.WriteHeader(status)
				if status == 200 {
					_, _ = w.Write([]byte(`{"Name":"hello"}`))
				}
			}))
			defer srv.Close()

			got, err := execGet[result](context.Background(), srv.URL)
			if n := requests.Load(); n != test.wantRequests {
				t.Errorf("got %d requests, want %d", n, test.wantRequests)
			}
			switch {
			case test.wantErr != nil:
				if !errors.Is(err, test.wantErr) {
					t.Errorf("got error %v, want %v", err, test.wantErr)
				}
			case test.wantAnyErr:
				if err == nil || errors.Is(err, ErrUnavailable) {
					t.Errorf("got error %v, want a non-retryable error", err)
				}
			default:
				if err != nil {
					t.Fatalf("got error %v", err)
				}
				if got.Name != "hello" {
					t.Errorf("got %+v", got)
				}
			}
		})
	}
}

func TestExecGetNetworkErrorIsUnavailable(t *testing.T) {
	origBackoff := retryBackoff
	retryBackoff = func(int) time.Duration { return 0 }
	t.Cleanup(func() { retryBackoff = origBackoff })

	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close() // Nothing is listening anymore.

	_, err := execGet[struct{}](context.Background(), url)
	if !errors.Is(err, ErrUnavailable) {
		t.Errorf("got error %v, want ErrUnavailable", err)
	}
}

func TestExecGetStopsRetryingWhenCanceled(t *testing.T) {
	var requests atomic.Int32
	ctx, cancel := context.WithCancel(context.Background())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		cancel()
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := execGet[struct{}](ctx, srv.URL)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("got error %v, want context.Canceled", err)
	}
	if n := requests.Load(); n != 1 {
		t.Errorf("got %d requests, want 1", n)
	}
}
