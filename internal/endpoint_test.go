// SPDX-FileCopyrightText: Copyright 2023 Prasad Tengse
// SPDX-License-Identifier: MIT

package internal_test

import (
	"net/url"
	"testing"

	"github.com/tprasadtp/go-githubapp/internal"
)

func TestDefaultEndpoint(t *testing.T) {
	_, err := url.Parse(internal.DefaultEndpoint)
	if err != nil {
		t.Errorf("DefaultEndpoint URL(%s) is invalid: %s", internal.DefaultEndpoint, err)
	}
}
