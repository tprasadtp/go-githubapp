// SPDX-FileCopyrightText: Copyright 2023 Prasad Tengse
// SPDX-License-Identifier: MIT

package githubapp

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"log/slog"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

var _ io.Reader = (*errReader)(nil)

// errReader always returns os.ErrClosed on read.
type errReader struct{}

func (*errReader) Read([]byte) (int, error) {
	return 0, os.ErrClosed
}

func TestVerifyWebHook_LogValuer(t *testing.T) {
	w := WebHook{}
	if w.LogValue().Kind() != slog.KindGroup {
		t.Errorf("WebHook must implement LogValuer with KindGroup")
	}
}

func TestVerifyWebHookSignature(t *testing.T) {
	type testCase struct {
		name    string
		request *http.Request
		expect  WebHook
		secret  string
		err     error
	}
	const secret = "It's a Secret to Everybody"
	const payload = "Hello, World!"
	var headers = make(http.Header) // must be cloned between tests!
	headers.Set(deliveryHeader, "72d3162e-cc78-11e3-81ab-4c9367dc0958")
	headers.Set(signatureSHA256Header, "sha256=757107ea0eb2509fc211221cce984b8a37570b6d7586c22c46f4379c8b043e17")
	headers.Set("X-Hub-Signature", "sha1=01dc10d0c83e72ed246219cdd91669667fe2ca59")
	headers.Set("User-Agent", "GitHub-Hookshot/044aadd")
	headers.Set("Content-Type", "application/json")
	headers.Set(eventHeader, "issues")
	headers.Set(hookIDHeader, "292430182")
	headers.Set(installationTargetIDHeader, "79929171")
	headers.Set(installationTargetTypeHeader, "repository")

	tt := []testCase{
		{
			name:   "nil-request",
			err:    ErrWebHookRequest,
			secret: secret,
		},
		{
			name:   "invalid-method",
			err:    ErrWebHookMethod,
			secret: secret,
			request: func() *http.Request {
				r := httptest.NewRequest(
					http.MethodGet,
					"/",
					bytes.NewBufferString(payload),
				)
				r.Header = maps.Clone(headers)
				return r
			}(),
		},
		{
			name:   "nil-headers",
			err:    ErrWebHookRequest,
			secret: secret,
			request: func() *http.Request {
				r := httptest.NewRequest(
					http.MethodPost,
					"/",
					bytes.NewBufferString(payload),
				)
				r.Header = nil
				return r
			}(),
		},
		{
			name:   "empty-headers",
			secret: secret,
			err:    ErrWebHookRequest,
			request: func() *http.Request {
				r := httptest.NewRequest(
					http.MethodPost,
					"/",
					bytes.NewBufferString(payload),
				)
				r.Header = make(http.Header)
				return r
			}(),
		},
		{
			name:   "missing-content-type-header",
			err:    ErrWebHookRequest,
			secret: secret,
			request: func() *http.Request {
				r := httptest.NewRequest(
					http.MethodPost,
					"/",
					bytes.NewBufferString(payload),
				)
				r.Header = maps.Clone(headers)
				r.Header.Del(contentTypeHeader)
				return r
			}(),
		},
		{
			name:   "unsupported-content-type-header",
			err:    ErrWebHookContentType,
			secret: secret,
			request: func() *http.Request {
				r := httptest.NewRequest(
					http.MethodPost,
					"/",
					bytes.NewBufferString(payload),
				)
				r.Header = maps.Clone(headers)
				r.Header.Set(contentTypeHeader, "application/x-www-form-urlencoded")
				return r
			}(),
		},
		{
			name:   "missing-signature-header",
			err:    ErrWebHookRequest,
			secret: secret,
			request: func() *http.Request {
				r := httptest.NewRequest(
					http.MethodPost,
					"/",
					bytes.NewBufferString(payload),
				)
				r.Header = maps.Clone(headers)
				r.Header.Del(signatureSHA256Header)
				return r
			}(),
		},
		{
			name:   "missing-signature-prefix",
			err:    ErrWebHookRequest,
			secret: secret,
			request: func() *http.Request {
				r := httptest.NewRequest(
					http.MethodPost,
					"/",
					bytes.NewBufferString(payload),
				)
				r.Header = maps.Clone(headers)
				r.Header.Set(
					signatureSHA256Header,
					"757107ea0eb2509fc211221cce984b8a37570b6d7586c22c46f4379c8b043e17")
				return r
			}(),
		},
		{
			name:   "signature-prefix-invalid",
			err:    ErrWebHookRequest,
			secret: secret,
			request: func() *http.Request {
				r := httptest.NewRequest(
					http.MethodPost,
					"/",
					bytes.NewBufferString(payload),
				)
				r.Header = maps.Clone(headers)
				r.Header.Set(
					signatureSHA256Header,
					"sha1=757107ea0eb2509fc211221cce984b8a37570b6d7586c22c46f4379c8b043e17")
				return r
			}(),
		},
		{
			name:   "signature-not-hex-encoded",
			err:    ErrWebHookRequest,
			secret: secret,
			request: func() *http.Request {
				r := httptest.NewRequest(
					http.MethodPost,
					"/",
					bytes.NewBufferString(payload),
				)
				r.Header = maps.Clone(headers)
				r.Header.Set(
					signatureSHA256Header,
					"sha256=?57107ea0eb2509fc211221cce984b8a37570b6d7586c22c46f4379c8b043e17")
				return r
			}(),
		},
		{
			name:   "error-reading-payload",
			err:    ErrWebHookRequest,
			secret: secret,
			request: func() *http.Request {
				r := httptest.NewRequest(
					http.MethodPost,
					"/",
					&errReader{},
				)
				r.Header = maps.Clone(headers)
				return r
			}(),
		},
		{
			name:   "installation-id-is-not-integer",
			err:    ErrWebHookRequest,
			secret: secret,
			request: func() *http.Request {
				r := httptest.NewRequest(
					http.MethodPost,
					"/",
					bytes.NewBufferString(payload),
				)
				r.Header = maps.Clone(headers)
				r.Header.Set(installationTargetIDHeader, "abcd")
				return r
			}(),
		},
		{
			name:   "payload-does-not-match-signature",
			err:    ErrWebhookSignature,
			secret: secret,
			request: func() *http.Request {
				r := httptest.NewRequest(
					http.MethodPost,
					"/",
					bytes.NewBufferString("something"),
				)
				r.Header = maps.Clone(headers)
				return r
			}(),
		},
		{
			name:   "signature-valid",
			secret: secret,
			request: func() *http.Request {
				r := httptest.NewRequest(
					http.MethodPost,
					"/",
					bytes.NewBufferString(payload),
				)
				r.Header = maps.Clone(headers)
				return r
			}(),
			expect: WebHook{
				ID:               "292430182",
				Event:            "issues",
				Payload:          []byte(payload),
				DeliveryID:       "72d3162e-cc78-11e3-81ab-4c9367dc0958",
				Signature:        "sha256=757107ea0eb2509fc211221cce984b8a37570b6d7586c22c46f4379c8b043e17",
				InstallationID:   79929171,
				InstallationType: "repository",
			},
		},
	}

	// More test data from responses.
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			hook, err := VerifyWebHookRequest(tc.secret, tc.request)
			if !reflect.DeepEqual(tc.expect, hook) {
				t.Errorf("expected=%#v, got=%#v", tc.expect, hook)
			}
			if !errors.Is(err, tc.err) {
				t.Errorf("expected error=%s, got=%s", tc.err, err)
			}
		})
	}
}

func TestVerifyWebHookSignature_Replay(t *testing.T) {
	dir := filepath.Join("internal", "testdata", "webhooks")
	items, le := os.ReadDir(dir)
	if le != nil {
		t.Fatalf("failed to read dir %s: %s", dir, le)
	}

	replays := make([]string, 0, len(items))
	for _, item := range items {
		if filepath.Ext(item.Name()) == ".replay" && item.Type().IsRegular() {
			replays = append(replays, item.Name())
		}
	}

	//nolint:gosec // used only for testing, ephemeral webhook server.
	const secret = "fa1286b4-ff70-4cf0-9471-443c796ff13b"
	for _, tc := range replays {
		file, err := os.Open(filepath.Join(dir, tc))
		if err != nil {
			t.Fatalf("failed to read webhook test data file: %s", err)
		}
		defer file.Close()
		request, err := http.ReadRequest(bufio.NewReader(file))
		if err != nil {
			t.Fatalf("failed to parse request from file: %s", err)
		}

		t.Run("Valid-"+strings.TrimSuffix(tc, ".replay"), func(t *testing.T) {
			webhook, werr := VerifyWebHookRequest(secret, request)
			if werr != nil {
				t.Errorf("expected no error, got: %s", werr)
			}
			if webhook.DeliveryID != strings.TrimSuffix(tc, ".replay") {
				t.Errorf("webhook.Delivery id is not valid")
			}
		})

		t.Run("Invalid-"+strings.TrimSuffix(tc, ".replay"), func(t *testing.T) {
			webhook, werr := VerifyWebHookRequest("secret", request)
			if !errors.Is(werr, ErrWebhookSignature) {
				t.Errorf("expected error %s, got: %s", ErrWebhookSignature, werr)
			}
			if !reflect.DeepEqual(webhook, WebHook{}) {
				t.Errorf("invalid signature should not populate webhook fields")
			}
		})
	}
}

func BenchmarkVerifyWebHookSignature(b *testing.B) {
	const secret = "It's a Secret to Everybody"
	const payload = "Hello, World!"
	var headers = make(http.Header)
	headers.Set(deliveryHeader, "72d3162e-cc78-11e3-81ab-4c9367dc0958")
	headers.Set(signatureSHA256Header, "sha256=757107ea0eb2509fc211221cce984b8a37570b6d7586c22c46f4379c8b043e17")
	headers.Set("X-Hub-Signature", "sha1=01dc10d0c83e72ed246219cdd91669667fe2ca59")
	headers.Set("User-Agent", "GitHub-Hookshot/044aadd")
	headers.Set("Content-Type", "application/json")
	headers.Set(eventHeader, "issues")
	headers.Set(hookIDHeader, "292430182")
	headers.Set(installationTargetIDHeader, "79929171")
	headers.Set(installationTargetTypeHeader, "repository")

	valid := httptest.NewRequest(http.MethodPost,
		"https://webhooks.go-githubapp.golang.test",
		bytes.NewBufferString(payload),
	)
	valid.Header = maps.Clone(headers)

	invalid := httptest.NewRequest(http.MethodPost,
		"https://webhooks.go-githubapp.golang.test",
		bytes.NewBufferString(`{"foo":"bar"}`),
	)
	invalid.Header = maps.Clone(headers)

	var webhook WebHook
	var err error

	b.Run("Valid-Signature", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			webhook, err = VerifyWebHookRequest(secret, valid)
		}
	})

	b.Run("Invalid-Signature", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			webhook, err = VerifyWebHookRequest(secret, invalid)
		}
	})

	_ = err
	_ = webhook
}
