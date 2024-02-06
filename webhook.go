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

	"github.com/tprasadtp/go-githubapp/internal/api"
)

var (
	_ slog.LogValuer = (*WebHook)(nil)
)

const (
	// ErrWebHookMethod is returned by [VerifyWebHookRequest] when a request method
	// is not PUT.
	ErrWebHookMethod = Error("githubapp(webhook): method not supported")

	// ErrWebHookContentType is returned by [VerifyWebHookRequest] when a request
	// content type is not 'application/json'.
	ErrWebHookContentType = Error("githubapp(webhook): unsupported content type")

	// ErrWebHookRequest is returned by [VerifyWebHookRequest] when request is invalid
	// or missing GitHub specific webhook metadata headers (X-GitHub-Event, X-GitHub-Hook-ID etc.).
	ErrWebHookRequest = Error("githubapp(webhook): invalid request")

	// ErrWebhookSignature is returned by [VerifyWebHookRequest] when the signature does not match.
	ErrWebhookSignature = Error("githubapp(webhook): HMAC-SHA256 signature is invalid")
)

// WebHook is returned by [VerifyWebHookRequest] upon successful verification of
// the webhook request. It contains all the webhook payloads with additional info
// from headers to detect GitHub app installation.
type WebHook struct {
	// ID is webhook ID received in X-GitHub-Hook-ID header.
	ID string

	// Event is event type like "issues" received in X-GitHub-Event header.
	Event string

	// Payload is payload received in POST.
	Payload []byte

	// DeliveryID is a unique delivery id received in X-GitHub-DeliveryID header.
	DeliveryID string

	// Signature is HMAC hex digest of the request body with the prefix "sha256=".
	// This is populated from X-Hub-Signature-256 header.
	Signature string

	// GitHub app installation ID. This can be used by WithInstallationID
	// for building Transport applicable for the installation in the hook event.
	InstallationID uint64

	// InstallationType can be repo|user|org.
	InstallationType string
}

func (w *WebHook) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("id", w.ID),
		slog.String("event_type", w.Event),
		slog.String("delivery_id", w.DeliveryID),
		slog.String("installation_type", w.InstallationType),
		slog.Uint64("installation_id", w.InstallationID),
	)
}

// VerifyWebHookRequest is a simple function to verify webhook HMAC-SHA256 signature.
//
// This functions assumes that headers are canonical by default and have not been
// modified. Only HMAC-SHA256 signatures are considered for verification and SHA1
// signature headers are ignored.
//
// Typically, HMAC secret would be []byte, but as it may be updated via web interface,
// which can only accept strings. Returned value is only valid if error is nil.
//
//   - [ErrWebHookRequest] is returned when request is invalid and is missing or malformed
//     headers like 'X-GitHub-Event', 'X-Hub-Signature-256' and more.
//   - [ErrWebHookMethod] is returned when webhook request is not a PUT request.
//   - [ErrWebHookContentType] is returned when content type header is not set to 'application/json'.
//     Though GitHub supports 'application/x-www-form-urlencoded', it is NOT supported by this library.
//   - [ErrWebhookSignature] is returned when signature does not match.
//
// An example HTTP handler which returns appropriate http status code is shown below.
//
//	mux.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
//	    webhook, err := githubapp.VerifyWebHookRequest(secret, r)
//	    if err != nil {
//	        switch {
//	        case errors.Is(err, githubapp.ErrWebhookSignature):
//	            w.WriteHeader(http.StatusUnauthorized)
//	        case errors.Is(err, githubapp.ErrWebHookRequest):
//	            w.WriteHeader(http.StatusBadRequest)
//	        case errors.Is(err, githubapp.ErrWebHookContentType):
//	            w.WriteHeader(http.StatusUnsupportedMediaType)
//	        case errors.Is(err, githubapp.ErrWebHookMethod):
//	            w.WriteHeader(http.StatusMethodNotAllowed)
//	        default:
//	            // This is non-reachable code.
//	            w.WriteHeader(http.StatusNotImplemented)
//	        }
//	        _, _ = w.Write([]byte(err.Error()))
//	        return
//	    }
//
//	    // Do something with webhook, for example, put it in SQS or PubSub.
//	    err = doSomething(r.Context(), webhook)
//	    if err != nil {
//	        w.WriteHeader(http.StatusInternalServerError)
//	        return
//	    }
//
//		// Return HTTP status 2xx.
//	    w.WriteHeader(http.StatusAccepted)
//	})
func VerifyWebHookRequest(secret string, req *http.Request) (WebHook, error) {
	if req == nil {
		return WebHook{}, fmt.Errorf("%w: request is nil", ErrWebHookRequest)
	}

	if !strings.EqualFold(req.Method, http.MethodPost) {
		return WebHook{}, fmt.Errorf("%w: %s", ErrWebHookMethod, req.Method)
	}

	if req.Header == nil {
		return WebHook{}, fmt.Errorf("%w: headers are nil", ErrWebHookRequest)
	}

	// Ensure other X-GitHub-* headers are populated.
	requiredHeaders := [...]string{
		api.EventHeader,
		api.HookIDHeader,
		api.DeliveryHeader,
		api.InstallationTargetTypeHeader,
		api.InstallationTargetIDHeader,
		api.ContentTypeHeader,
		api.SignatureSHA256Header,
	}
	missingHeaders := make([]string, 0, len(requiredHeaders))
	for _, item := range requiredHeaders {
		if req.Header.Get(item) == "" {
			missingHeaders = append(missingHeaders, item)
		}
	}

	if len(missingHeaders) > 0 {
		return WebHook{}, fmt.Errorf("%w: missing header(s): %v", ErrWebHookRequest, missingHeaders)
	}

	// Only support content type application/json.
	if req.Header.Get(api.ContentTypeHeader) != "application/json" {
		return WebHook{}, fmt.Errorf("%w: %q", ErrWebHookContentType,
			req.Header.Get(api.ContentTypeHeader))
	}

	// Ensure X-GitHub-Hook-Installation-Target-ID header is an integer.
	installID, err := strconv.ParseUint(req.Header.Get(api.InstallationTargetIDHeader), 10, 64)
	if err != nil {
		return WebHook{},
			fmt.Errorf("%w: invalid %s header", ErrWebHookRequest, api.InstallationTargetIDHeader)
	}

	// Ensure X-Hub-Signature-256 header exists and has a valid format.
	signature := req.Header.Get(api.SignatureSHA256Header)
	if !strings.HasPrefix(signature, "sha256=") {
		return WebHook{}, fmt.Errorf("%w: missing prefix sha256= from %s header",
			ErrWebHookRequest, api.SignatureSHA256Header)
	}

	// Decode hex encoded signature.
	untrusted, err := hex.DecodeString(strings.TrimPrefix(signature, "sha256="))
	if err != nil {
		return WebHook{}, fmt.Errorf("%w: signature not hex encoded", ErrWebHookRequest)
	}

	data, err := io.ReadAll(req.Body)
	if err != nil {
		return WebHook{}, fmt.Errorf("%w: failed to read request body", ErrWebHookRequest)
	}

	// Compute HMAC-SHA256.
	hasher := hmac.New(sha256.New, []byte(secret))
	hasher.Write(data)

	trusted := hasher.Sum(nil)

	// Check HMAC signature.
	if hmac.Equal(trusted, untrusted) {
		w := WebHook{
			ID:               req.Header.Get(api.HookIDHeader),
			DeliveryID:       req.Header.Get(api.DeliveryHeader),
			Event:            req.Header.Get(api.EventHeader),
			Signature:        signature,
			InstallationID:   installID,
			InstallationType: req.Header.Get(api.InstallationTargetTypeHeader),
			Payload:          data,
		}
		return w, nil
	}

	return WebHook{}, ErrWebhookSignature
}
