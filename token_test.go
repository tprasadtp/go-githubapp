// SPDX-FileCopyrightText: Copyright 2023 Prasad Tengse
// SPDX-License-Identifier: MIT

package githubapp

import (
	"context"
	"crypto"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/tprasadtp/go-githubapp/internal"
	"github.com/tprasadtp/go-githubapp/internal/testkeys"
)

func TestInstallationToken(t *testing.T) {
	t.Run("slog-log-valuer", func(t *testing.T) {
		now := time.Now().Truncate(time.Second)
		token := InstallationToken{
			Exp: now.Add(time.Minute + time.Second),
		}
		v := token.LogValue()
		for _, item := range v.Group() {
			if item.Key == "token" {
				if item.Value.Kind() != slog.KindString {
					t.Errorf("token should be of string kind: %s", item.Value.Kind())
				}
				if item.Value.String() == "token" {
					t.Errorf("token value should be redacted: %s", item.Value.String())
				}
			}
		}
	})
	t.Run("empty-value", func(t *testing.T) {
		token := InstallationToken{}
		if token.IsValid() {
			t.Errorf("empty token should be invalid")
		}
	})
	t.Run("exp", func(t *testing.T) {
		now := time.Now().Truncate(time.Second)
		token := InstallationToken{
			Exp:   now.Add(-time.Minute),
			Token: "token",
		}
		if token.IsValid() {
			t.Errorf("token should be invalid")
		}
	})
	t.Run("now+59s", func(t *testing.T) {
		now := time.Now().Truncate(time.Second)
		token := InstallationToken{
			Exp:   now.Add(time.Minute - time.Second),
			Token: "token",
		}
		if token.IsValid() {
			t.Errorf("token should be invalid")
		}
	})
	t.Run("now+60s", func(t *testing.T) {
		now := time.Now().Truncate(time.Second)
		token := InstallationToken{
			Exp:   now.Add(time.Minute + time.Second),
			Token: "token",
		}
		if !token.IsValid() {
			t.Errorf("token should be valid")
		}
	})
	t.Run("now+120s", func(t *testing.T) {
		now := time.Now().Truncate(time.Second)
		token := InstallationToken{
			Exp:   now.Add(2 * time.Minute),
			Token: "token",
		}
		if !token.IsValid() {
			t.Errorf("token should be valid")
		}
	})
}

func TestInstallationToken_Revoke(t *testing.T) {
	type testCase struct {
		name  string
		token InstallationToken
		rt    http.RoundTripper
		ctx   context.Context
		ok    bool
	}
	tt := []testCase{
		{
			name: "empty-value",
			ctx:  context.Background(),
		},
		{
			name: "invalid-token-empty",
			token: InstallationToken{
				Token:          "",
				Server:         "https://api.github.com/",
				AppID:          99,
				InstallationID: 99,
				AppName:        "gh-integration-tests-demo",
				Exp:            time.Now().Add(time.Hour),
				Owner:          "gh-integration-tests",
			},
			ctx: context.Background(),
		},
		{
			name: "invalid-token-not-valid",
			token: InstallationToken{
				Token:          "ghs_token",
				Server:         "https://api.github.com/",
				AppID:          99,
				InstallationID: 99,
				AppName:        "gh-integration-tests-demo",
				Exp:            time.Now().Add(time.Minute),
				Owner:          "gh-integration-tests",
			},
			ctx: context.Background(),
		},
		{
			name: "invalid-server-url",
			token: InstallationToken{
				Token:          "ghs_token",
				Server:         "https://api. github.com/",
				AppID:          99,
				InstallationID: 99,
				AppName:        "gh-integration-tests-demo",
				Exp:            time.Now().Add(time.Hour),
				Owner:          "gh-integration-tests",
			},
			ctx: context.Background(),
		},
		{
			name: "invalid-server-url-scheme",
			token: InstallationToken{
				Token:          "ghs_token",
				Server:         "go-githubapp://api.github.com/",
				AppID:          99,
				InstallationID: 99,
				AppName:        "gh-integration-tests-demo",
				Exp:            time.Now().Add(time.Hour),
				Owner:          "gh-integration-tests",
			},
			ctx: context.Background(),
		},
		{
			name: "server-url-has-queries",
			token: InstallationToken{
				Token:          "ghs_token",
				Server:         "https://api.github.com/token?foo=bar",
				AppID:          99,
				InstallationID: 99,
				AppName:        "gh-integration-tests-demo",
				Exp:            time.Now().Add(time.Hour),
				Owner:          "gh-integration-tests",
			},
			ctx: context.Background(),
		},
		{
			name: "server-url-has-fragments",
			token: InstallationToken{
				Token:          "ghs_token",
				Server:         "https://api.github.com/token#bar",
				AppID:          99,
				InstallationID: 99,
				AppName:        "gh-integration-tests-demo",
				Exp:            time.Now().Add(time.Hour),
				Owner:          "gh-integration-tests",
			},
			ctx: context.Background(),
		},
		{
			name: "network-error-no-custom-round-tripper",
			token: InstallationToken{
				Token:          "ghs_token",
				Server:         "http://this-endpoin-is-not-resolvable.go-githubapp.test",
				AppID:          99,
				InstallationID: 99,
				AppName:        "gh-integration-tests-demo",
				Exp:            time.Now().Add(time.Hour),
				Owner:          "gh-integration-tests",
			},
			ctx: context.Background(),
		},
		{
			name: "api-error-not-204",
			token: InstallationToken{
				Token:          "ghs_token",
				AppID:          99,
				InstallationID: 99,
				AppName:        "gh-integration-tests-demo",
				Exp:            time.Now().Add(time.Hour),
				Owner:          "gh-integration-tests",
			},
			ctx: context.Background(),
			rt: internal.RoundTripFunc(func(r *http.Request) (*http.Response, error) {
				resp := httptest.NewRecorder()
				resp.Body = nil
				resp.WriteHeader(http.StatusNotFound)
				return resp.Result(), nil
			}),
		},
		{
			name: "no-error",
			token: InstallationToken{
				Token:          "ghs_token",
				Server:         "http://mock-endpoint.go-githubapp.test",
				AppID:          99,
				InstallationID: 99,
				AppName:        "gh-integration-tests-demo",
				Exp:            time.Now().Add(time.Hour),
				Owner:          "gh-integration-tests",
			},
			ctx: context.Background(),
			rt: internal.RoundTripFunc(func(r *http.Request) (*http.Response, error) {
				t.Helper()
				t.Logf("request=%v", r)
				if r.Header.Get(authzHeader) == "" {
					t.Errorf("Authorization header is empty")
				}

				if r.Header.Get(apiVersionHeader) == "" {
					t.Errorf("%s header is empty", apiVersionHeader)
				}

				if !strings.EqualFold(r.Method, http.MethodDelete) {
					t.Errorf("Method should be DELETE")
				}

				resp := httptest.NewRecorder()
				resp.WriteHeader(http.StatusNoContent)
				return resp.Result(), nil
			}),
			ok: true,
		},
		{
			name: "no-error-nil-context",
			token: InstallationToken{
				Token:          "ghs_token",
				Server:         "http://mock-endpoint.go-githubapp.test",
				AppID:          99,
				InstallationID: 99,
				AppName:        "gh-integration-tests-demo",
				Exp:            time.Now().Add(time.Hour),
				Owner:          "gh-integration-tests",
			},
			rt: internal.RoundTripFunc(func(r *http.Request) (*http.Response, error) {
				t.Logf("request=%v", r)
				resp := httptest.NewRecorder()
				resp.WriteHeader(http.StatusNoContent)
				return resp.Result(), nil
			}),
			ctx: nil,
			ok:  true,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.token.revoke(tc.ctx, tc.rt)
			if tc.ok {
				if err != nil {
					t.Errorf("unexpected error: %s", err)
				}
				if !tc.token.Exp.Before(time.Now()) {
					t.Errorf("token timestamp is not modified")
				}
			} else if !tc.ok && err == nil {
				t.Errorf("expected error, bit got nil")
			}
		})
	}
}

func TestNewInstallationToken(t *testing.T) {
	type testCase struct {
		name    string
		options []Option
		token   InstallationToken
		ctx     context.Context
		signer  crypto.Signer
		appID   uint64
		err     error
		// rt      http.RoundTripper
	}

	tt := []testCase{
		{
			name:    "invalid-options-nil-signer",
			ctx:     context.Background(),
			options: []Option{WithInstallationID(99)},
			appID:   99,
			err:     ErrOptions,
		},
		{
			name:    "invalid-options-invalid-app-id",
			ctx:     context.Background(),
			options: []Option{WithInstallationID(99)},
			signer:  testkeys.RSA2048(),
			err:     ErrOptions,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			token, err := NewInstallationToken(tc.ctx, tc.appID, tc.signer, tc.options...)
			if !errors.Is(err, tc.err) {
				t.Errorf("expected error=%s, got=%s", tc.err, err)
			}
			if !reflect.DeepEqual(tc.token, token) {
				t.Errorf("expected=%#v, got=%#v", tc.token, token)
			}

			if err != nil {
				if !tc.token.Exp.Before(time.Now()) {
					t.Errorf("token timestamp is not modified")
				}
			}
		})
	}
}
