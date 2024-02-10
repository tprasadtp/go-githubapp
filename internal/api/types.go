// SPDX-FileCopyrightText: Copyright 2023 Prasad Tengse
// SPDX-License-Identifier: MIT

package api

// Repository represents a GitHub repository. This is incomplete!
type Repository struct {
	ID    *int64  `json:"id,omitempty"`
	Owner *User   `json:"owner,omitempty"`
	Name  *string `json:"name,omitempty"`
}

// User represents a GitHub user. This is incomplete!
type User struct {
	Login *string `json:"login,omitempty"`
	ID    *int64  `json:"id,omitempty"`
}

// InstallationTokenRequest is payload for installation token request.
//
// https://docs.github.com/en/rest/apps/apps?apiVersion=2022-11-28#create-an-installation-access-token-for-an-app
type InstallationTokenRequest struct {
	Repositories []string          `json:"repositories,omitempty"`
	Permissions  map[string]string `json:"permissions,omitempty"`
}

// InstallationTokenResponse is returned by the API for [InstallationTokenRequest].
//
// https://docs.github.com/en/rest/apps/apps?apiVersion=2022-11-28#create-an-installation-access-token-for-an-app
type InstallationTokenResponse struct {
	Token        string            `json:"token,omitempty"`
	Exp          *Timestamp        `json:"expires_at,omitempty"`
	Permissions  map[string]string `json:"permissions,omitempty"`
	Repositories []*Repository     `json:"repositories,omitempty"`
}

// Installation represents a GitHub Apps installation.
//
// https://docs.github.com/en/rest/apps/apps?apiVersion=2022-11-28#get-a-repository-installation-for-the-authenticated-app
type Installation struct {
	ID              *int64            `json:"id,omitempty"`
	AppID           *int64            `json:"app_id,omitempty"`
	AppSlug         *string           `json:"app_slug,omitempty"`
	TargetID        *int64            `json:"target_id,omitempty"`
	TargetType      *string           `json:"target_type,omitempty"`
	Account         *User             `json:"account,omitempty"`
	AccessTokensURL *string           `json:"access_tokens_url,omitempty"`
	Permissions     map[string]string `json:"permissions,omitempty"`
	SuspendedAt     *Timestamp        `json:"suspended_at,omitempty"`
}

type ErrorResponse struct {
	Message          string `json:"message,omitempty"` // error message
	DocumentationURL string `json:"documentation_url,omitempty"`
}

// App represents a GitHub App.
//
// https://docs.github.com/en/rest/apps/apps?apiVersion=2022-11-28#get-the-authenticated-app
type App struct {
	ID          *int64            `json:"id,omitempty"`
	Slug        *string           `json:"slug,omitempty"`
	NodeID      *string           `json:"node_id,omitempty"`
	Owner       *User             `json:"owner,omitempty"`
	Name        *string           `json:"name,omitempty"`
	Description *string           `json:"description,omitempty"`
	ExternalURL *string           `json:"external_url,omitempty"`
	Permissions map[string]string `json:"permissions,omitempty"`
	Events      []string          `json:"events,omitempty"`
}

// ListInstallationRepositoriesResponse is a response received by
//
// https://docs.github.com/en/rest/apps/installations?apiVersion=2022-11-28#list-repositories-accessible-to-the-app-installation
type ListInstallationRepositoriesResponse struct {
	TotalCount   int64         `json:"total_count,omitempty"`
	Repositories []*Repository `json:"repositories,omitempty"`
}
