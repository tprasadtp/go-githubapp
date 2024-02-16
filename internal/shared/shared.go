// SPDX-FileCopyrightText: Copyright 2024 Prasad Tengse
// SPDX-License-Identifier: MIT

package shared

import (
	"context"
	"testing"
	"time"
)

func TestingCtx(t *testing.T, timeout time.Duration) (context.Context, context.CancelFunc) {
	// TestingContext returns
	//
	// Ideally we would set per set timeouts, but they are not available yet.
	// See https://github.com/golang/go/issues/48157 for more info.

	if ts, ok := t.Deadline(); ok {
		return context.WithDeadline(context.Background(), ts)
	}

	if timeout < 0 {
		t.Logf("Ignoring invalid timeout value: %s", timeout)
		timeout = time.Second * 30
	}
	return context.WithTimeout(context.Background(), timeout)
}
