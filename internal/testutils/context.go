// SPDX-FileCopyrightText: Copyright 2024 Prasad Tengse
// SPDX-License-Identifier: MIT

package testutils

import (
	"context"
	"testing"
	"time"
)

// TestingContext returns context suitable for testing. Context's
// timeout is based on test's own timeout and test's own timeout
// whichever is lower. In case test lacks a timeout, given value
// is used as fallback. It is responsibility of the caller to invoke
// the returned cancel function to avoid leaking context.
func TestingContext(t *testing.T, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		t.Fatalf("TestingContext: timeout must be positive")
	}
	// Ideally we would set per set timeouts, but they are not available yet.
	// See https://github.com/golang/go/issues/48157 for more info.
	if ts, ok := t.Deadline(); ok {
		// If timeout is shorter than test's own timeout, prefer it.
		if v := time.Now().Add(timeout); v.Before(ts) {
			return context.WithDeadline(context.Background(), v)
		}
		return context.WithDeadline(context.Background(), ts)
	}
	return context.WithTimeout(context.Background(), timeout)
}
