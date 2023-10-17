// SPDX-FileCopyrightText: Copyright 2023 Prasad Tengse
// SPDX-License-Identifier: MIT

package api_test

import (
	"net/url"
	"testing"

	"github.com/tprasadtp/go-githubapp/internal/api"
)

func TestDefaultEndpoint(t *testing.T) {
	_, err := url.Parse(api.DefaultEndpoint)
	if err != nil {
		t.Errorf("DefaultEndpoint URL(%s) is invalid: %s", api.DefaultEndpoint, err)
	}
}
