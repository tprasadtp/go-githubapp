// SPDX-FileCopyrightText: Copyright 2023 Prasad Tengse
// SPDX-License-Identifier: MIT

package internal

import "net/http"

var _ http.RoundTripper = (*RoundTripFunc)(nil)

// RoundTripFunc is an adapter to allow the use of ordinary functions as
// RoundTrippers, similar to [http.HandlerFunc].
type RoundTripFunc func(*http.Request) (*http.Response, error)

// RoundTrip implements the RoundTripper interface by calling f(r).
func (f RoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
