// SPDX-FileCopyrightText: Copyright 2023 Prasad Tengse
// SPDX-License-Identifier: MIT

package githubapp

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
)

var (
	_ slog.LogValuer = (*WebHook)(nil)
)

// Errors returned by [VerifyWebHookSignature].
//
//   - [ErrWebHookRequest] is returned when request is invalid, or missing
//     github webhook metadata headers (X-GitHub-Event, X-GitHub-Hook-ID etc).
//   - [ErrWebhookSignature] is returned when signature is invalid, missing
//     or malformed.
const (
	ErrWebHookRequest   = Error("githubapp(webhook): invalid request")
	ErrWebhookSignature = Error("githubapp(webhook): signature is invalid")
)

// WebHook is returned by [VerifyWebHookRequest] upon successful verification of
// the webhook request. It contains all the webhook payloads with additional info
// from headers to detect github app installation.
type WebHook struct {
	// ID is webhook ID received in X-GitHub-Hook-ID header.
	ID string

	// Event is event type like "issues" received in X-GitHub-Event header.
	Event string

	// Payload is payload received in POST.
	Payload []byte

	// Delivery is unique delivery id received in X-GitHub-Delivery header.
	Delivery string

	// Signature is HMAC hex digest of the request body with prefix "sha256=".
	// This is populated from X-Hub-Signature-256 header.
	Signature string

	// Github app installation ID. This can be used by WithInstallationID
	// for building Transport applicable for the installation in the hook event.
	InstallationID uint64

	// InstallationType can be repo|user|org.
	InstallationType string
}

func (w *WebHook) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("id", w.ID),
		slog.String("event_type", w.Event),
		slog.String("delivery_id", w.Delivery),
		slog.String("installation_type", w.InstallationType),
		slog.Uint64("installation_id", w.InstallationID),
	)
}

// VerifyWebHookRequest is a simple function to verify webhook HMAC-SHA256 signature.
//
// This functions assumes that headers are canonical by default and have not been
// modified. Only HMAC-SHA256 signatures is considered for verification.
//
// Typically HMAC secret would be []byte, but as it may be updated via web interface,
// which can only accept strings.
func VerifyWebHookRequest(secret string, req *http.Request) (WebHook, error) {
	if req == nil {
		return WebHook{}, fmt.Errorf("%w: request is nil", ErrWebHookRequest)
	}

	if !strings.EqualFold(req.Method, http.MethodPost) {
		return WebHook{}, fmt.Errorf("%w: unsupported method %s",
			ErrWebHookRequest, req.Method)
	}

	if req.Header == nil {
		return WebHook{}, fmt.Errorf("%w: headers are nil", ErrWebHookRequest)
	}

	// Ensure other X-GitHub-* headers are populated.
	requiredHeaders := [...]string{
		eventHeader,
		hookIDHeader,
		deliveryHeader,
		installationTargetTypeHeader,
		installationTargetIDHeader,
		contentTypeHeader,
	}
	for _, item := range requiredHeaders {
		if req.Header.Get(item) == "" {
			return WebHook{}, fmt.Errorf("%w: missing or empty %s header",
				ErrWebHookRequest, item)
		}
	}

	// Only support content type application/json.
	if req.Header.Get(contentTypeHeader) != "application/json" {
		return WebHook{}, fmt.Errorf("%w: invalid %s header: %s",
			ErrWebHookRequest, contentTypeHeader, req.Header.Get(contentTypeHeader))
	}

	// Ensure X-GitHub-Hook-Installation-Target-ID header is an integer.
	installID, err := strconv.ParseUint(req.Header.Get(installationTargetIDHeader), 10, 64)
	if err != nil {
		return WebHook{}, fmt.Errorf("%w: invalid %s header (%s): %w",
			ErrWebHookRequest, installationTargetIDHeader,
			req.Header.Get(installationTargetIDHeader), err)
	}

	// Ensure X-Hub-Signature-256 header exists and has valid format.
	signature := req.Header.Get(signatureSHA256Header)
	if signature == "" {
		return WebHook{}, fmt.Errorf("%w: missing or empty %s header",
			ErrWebhookSignature, signatureSHA256Header)
	}

	if !strings.HasPrefix(signature, "sha256=") {
		return WebHook{}, fmt.Errorf("%w: missing prefix sha256= from %s header",
			ErrWebhookSignature, signatureSHA256Header)
	}

	// Decode hex encoded signature.
	untrusted, err := hex.DecodeString(strings.TrimPrefix(signature, "sha256="))
	if err != nil {
		return WebHook{}, fmt.Errorf("%w: signature not hex encoded: %w",
			ErrWebhookSignature, err)
	}

	data, err := io.ReadAll(req.Body)
	if err != nil {
		return WebHook{}, fmt.Errorf("githubapp(webhook): failed to read request body: %w", err)
	}

	// Compute HMAC-SHA256.
	hasher := hmac.New(sha256.New, []byte(secret))
	hasher.Write(data)

	trusted := hasher.Sum(nil)

	// Check HMAC signature.
	if hmac.Equal(trusted, untrusted) {
		w := WebHook{
			ID:               req.Header.Get(hookIDHeader),
			Delivery:         req.Header.Get(deliveryHeader),
			Event:            req.Header.Get(eventHeader),
			Signature:        signature,
			InstallationID:   installID,
			InstallationType: req.Header.Get(installationTargetTypeHeader),
			Payload:          data,
		}
		return w, nil
	}

	return WebHook{}, fmt.Errorf("%w: signature mismatch", ErrWebhookSignature)
}
