// SPDX-FileCopyrightText: Copyright 2023 Prasad Tengse
// SPDX-License-Identifier: MIT

package githubapp

// Common headers used by this package.
const (
	apiVersionHeader      = "X-GitHub-Api-Version"
	apiVersionHeaderValue = "2022-11-28"
	acceptHeader          = "Accept"
	acceptHeaderValue     = "application/vnd.github.v3+json"
	uaHeader              = "User-Agent"
	uaHeaderValue         = "github.com/tprasadtp/go-githubapp/v0"
	authzHeader           = "Authorization"
	contentTypeHeader     = "Content-Type"
	contentTypeJSON       = "application/json"
)

// Github webhook headers in canonical form.
const (
	signatureSHA256Header        = "X-Hub-Signature-256"
	eventHeader                  = "X-GitHub-Event"
	hookIDHeader                 = "X-GitHub-Hook-ID"
	deliveryHeader               = "X-GitHub-Delivery"
	installationTargetIDHeader   = "X-GitHub-Hook-Installation-Target-ID"
	installationTargetTypeHeader = "X-GitHub-Hook-Installation-Target-Type"
)
