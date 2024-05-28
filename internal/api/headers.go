// SPDX-FileCopyrightText: Copyright 2024 Prasad Tengse
// SPDX-License-Identifier: MIT

package api

// Common headers used by this package.
const (
	VersionHeader      = "X-GitHub-Api-Version"
	VersionHeaderValue = "2022-11-28"
	AcceptHeader       = "Accept"
	AcceptHeaderValue  = "application/vnd.github.v3+json"
	UAHeader           = "User-Agent"
	UAHeaderValue      = "github.com/tprasadtp/go-githubapp/v0"
	AuthzHeader        = "Authorization"
	ContentTypeHeader  = "Content-Type"
	ContentTypeJSON    = "application/json"
)

// GitHub webhook headers in canonical form.
const (
	SignatureSHA256Header        = "X-Hub-Signature-256"
	EventHeader                  = "X-GitHub-Event"
	HookIDHeader                 = "X-GitHub-Hook-ID"
	DeliveryHeader               = "X-GitHub-Delivery"
	InstallationTargetIDHeader   = "X-GitHub-Hook-Installation-Target-ID"
	InstallationTargetTypeHeader = "X-GitHub-Hook-Installation-Target-Type"
)

// AuthzHeaderValue is a convenience function to return Authorization header as value.
// If the token is empty, this returns empty string. Token is assumed to be
// bearer token.
func AuthzHeaderValue(token string) string {
	if token == "" {
		return ""
	}
	return "Bearer " + token
}
