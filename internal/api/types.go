// SPDX-FileCopyrightText: Copyright 2023 Prasad Tengse
// SPDX-License-Identifier: MIT

// Package api holds types and methods to serialize and deserialize
// requests to and from GitHub API.
//
// Types are just enough for app endpoints required by library to work
// and should be considered incomplete. Use [github.com/google/go-github/github]
// or [github.com/shurcooL/githubv4] to access the GitHub API with
// [github.com/tprasadtp/go-githubapp.Transport].
package api

import (
	"strconv"
	"time"
)

// Repository represents a GitHub repository.
type Repository struct {
	ID    *int64  `json:"id,omitempty"`
	Owner *User   `json:"owner,omitempty"`
	Name  *string `json:"name,omitempty"`
}

// User represents a GitHub user.
type User struct {
	Login *string `json:"login,omitempty"`
	ID    *int64  `json:"id,omitempty"`
}

type InstallationTokenRequest struct {
	// The names of the repositories that the installation token can access.
	// Providing repository names restricts the access of an installation token to specific repositories.
	Repositories []string `json:"repositories,omitempty"`

	// The permissions granted to the access token.
	// The permissions object includes the permission names and their access type.
	Permissions map[string]string `json:"permissions,omitempty"`
}

type InstallationTokenResponse struct {
	Token        string            `json:"token,omitempty"`
	Exp          *Timestamp        `json:"expires_at,omitempty"`
	Permissions  map[string]string `json:"permissions,omitempty"`
	Repositories []*Repository     `json:"repositories,omitempty"`
}

// Installation represents a GitHub Apps installation.
type Installation struct {
	ID                     *int64            `json:"id,omitempty"`
	AppID                  *int64            `json:"app_id,omitempty"`
	AppSlug                *string           `json:"app_slug,omitempty"`
	TargetID               *int64            `json:"target_id,omitempty"`
	TargetType             *string           `json:"target_type,omitempty"`
	Account                *User             `json:"account,omitempty"`
	AccessTokensURL        *string           `json:"access_tokens_url,omitempty"`
	SingleFileName         *string           `json:"single_file_name,omitempty"`
	RepositorySelection    *string           `json:"repository_selection,omitempty"`
	Events                 []string          `json:"events,omitempty"`
	SingleFilePaths        []string          `json:"single_file_paths,omitempty"`
	Permissions            map[string]string `json:"permissions,omitempty"`
	HasMultipleSingleFiles *bool             `json:"has_multiple_single_files,omitempty"`
	SuspendedAt            *Timestamp        `json:"suspended_at,omitempty"`
}

type ErrorResponse struct {
	Message          string `json:"message,omitempty"` // error message
	DocumentationURL string `json:"documentation_url,omitempty"`
}

// App represents a GitHub App.
type App struct {
	ID                 *int64            `json:"id,omitempty"`
	Slug               *string           `json:"slug,omitempty"`
	NodeID             *string           `json:"node_id,omitempty"`
	Owner              *User             `json:"owner,omitempty"`
	Name               *string           `json:"name,omitempty"`
	Description        *string           `json:"description,omitempty"`
	ExternalURL        *string           `json:"external_url,omitempty"`
	Permissions        map[string]string `json:"permissions,omitempty"`
	Events             []string          `json:"events,omitempty"`
	InstallationsCount *int              `json:"installations_count,omitempty"`
}

// ListInstallationRepositoriesResponse is response received by
// https://docs.github.com/en/rest/apps/installations?apiVersion=2022-11-28#list-repositories-accessible-to-the-app-installation
type ListInstallationRepositoriesResponse struct {
	TotalCount   int64         `json:"total_count,omitempty"`
	Repositories []*Repository `json:"repositories,omitempty"`
}

// Timestamp represents a time that can be un-marshalled from a JSON string
// formatted as either an RFC3339 or Unix timestamp. This is necessary for some
// fields since the GitHub API is inconsistent in how it represents times.
type Timestamp struct {
	time.Time
}

func (t Timestamp) String() string {
	return t.Time.String()
}

// UnmarshalJSON implements the json.Unmarshaler interface.
// Time is expected in RFC3339 or Unix format.
//
//nolint:nonamedreturns,nakedret // ignore
func (t *Timestamp) UnmarshalJSON(data []byte) (err error) {
	str := string(data)
	i, err := strconv.ParseInt(str, 10, 64)
	if err == nil {
		t.Time = time.Unix(i, 0)
		if t.Time.Year() > 3000 {
			t.Time = time.Unix(0, i*1e6)
		}
	} else {
		t.Time, err = time.Parse(`"`+time.RFC3339+`"`, str)
	}
	return
}

// Equal reports whether t and u are equal based on time.Equal.
func (t Timestamp) Equal(u Timestamp) bool {
	return t.Time.Equal(u.Time)
}
