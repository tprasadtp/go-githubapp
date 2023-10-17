// SPDX-FileCopyrightText: Copyright 2023 Prasad Tengse
// SPDX-License-Identifier: MIT

package githubapp

import (
	"context"
	"crypto"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/tprasadtp/go-githubapp/internal"
	"github.com/tprasadtp/go-githubapp/internal/testdata/apitestdata"
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
				if r.Header.Get(authzHeader) == "" {
					t.Errorf("%s header is empty", authzHeader)
				}

				if r.Header.Get(apiVersionHeader) == "" {
					t.Errorf("%s header is empty", apiVersionHeader)
				}

				if !strings.EqualFold(r.Method, http.MethodDelete) {
					t.Errorf("request method should be DELETE")
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

				// ensure is valid returns false.
				if tc.token.IsValid() {
					t.Errorf("Token is still valid after revoke: %#v", tc.token)
				}
			} else if !tc.ok && err == nil {
				t.Errorf("expected error, bit got nil")
			}
		})
	}
}

func TestNewInstallationToken_TransportErr(t *testing.T) {
	type testCase struct {
		name    string
		options []Option
		ctx     context.Context
		signer  crypto.Signer
		appID   uint64
	}

	tt := []testCase{
		{
			name:    "invalid-options-nil-signer",
			ctx:     context.Background(),
			options: []Option{WithInstallationID(99)},
			appID:   99,
		},
		{
			name:    "invalid-options-invalid-app-id",
			ctx:     context.Background(),
			options: []Option{WithInstallationID(99)},
			signer:  testkeys.RSA2048(),
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			token, err := NewInstallationToken(tc.ctx, tc.appID, tc.signer, tc.options...)
			if err == nil {
				t.Errorf("expected an error, got nil")
			}

			if !reflect.DeepEqual(token, InstallationToken{}) {
				t.Errorf("expected token to be empty")
			}
		})
	}
}

func TestNewInstallationToken_MockServer(t *testing.T) {
	type testCase struct {
		name    string
		options []Option
		ok      bool
		handler http.Handler
		scopes  map[string]string
		repos   []string
	}
	m := apitestdata.Get(t)

	tt := []testCase{
		{
			name:    "WithInstallationID",
			options: []Option{WithInstallationID(apitestdata.InstallationID)},
			ok:      true,
			scopes:  map[string]string{"contents": "read", "issues": "read", "metadata": "read"},
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var key string
				switch r.URL.Path {
				case "/app":
					key = "get-app"
				case fmt.Sprintf("/app/installations/%d", apitestdata.InstallationID):
					key = "get-installation-by-id"
				case fmt.Sprintf("/app/installations/%d/access_tokens", apitestdata.InstallationID):
					key = "post-installation-token"
					w.WriteHeader(http.StatusCreated)
				case fmt.Sprintf("/users/%s[bot]", apitestdata.AppSlug):
					key = "get-user-bot"
				default:
					panic(fmt.Sprintf("Unknown/Invalid Request => %s", r.URL))
				}
				resp, ok := m[key]
				if ok {
					_, _ = w.Write(resp)
				} else {
					t.Fatalf("Key not found in response data: %q", key)
				}
			}),
		},
		{
			name:    "WithOwner",
			options: []Option{WithOwner(apitestdata.InstallationOwner)},
			ok:      true,
			scopes:  map[string]string{"contents": "read", "issues": "read", "metadata": "read"},
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var key string
				switch r.URL.Path {
				case "/app":
					key = "get-app"
				case fmt.Sprintf("/users/%s/installation", apitestdata.InstallationOwner):
					key = "get-installation-by-user"
				case fmt.Sprintf("/app/installations/%d/access_tokens", apitestdata.InstallationID):
					key = "post-installation-token"
					w.WriteHeader(http.StatusCreated)
				case fmt.Sprintf("/users/%s[bot]", apitestdata.AppSlug):
					key = "get-user-bot"
				default:
					panic(fmt.Sprintf("Unknown/Invalid Request => %s", r.URL))
				}
				resp, ok := m[key]
				if ok {
					_, _ = w.Write(resp)
				} else {
					t.Fatalf("Key not found in response data: %q", key)
				}
			}),
		},
		{
			name: "WithRepositories",
			options: []Option{
				WithRepositories(
					apitestdata.InstallationOwner + "/" + apitestdata.InstallationOwner,
				),
			},
			ok:     true,
			scopes: map[string]string{"contents": "read", "issues": "read", "metadata": "read"},
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var key string
				switch r.URL.Path {
				case "/app":
					key = "get-app"
				case fmt.Sprintf("/users/%s/installation", apitestdata.InstallationOwner):
					key = "get-installation-by-repo"
				case fmt.Sprintf("/app/installations/%d/access_tokens", apitestdata.InstallationID):
					key = "post-installation-token"
					w.WriteHeader(http.StatusCreated)
				case fmt.Sprintf("/users/%s[bot]", apitestdata.AppSlug):
					key = "get-user-bot"
				default:
					panic(fmt.Sprintf("Unknown/Invalid Request => %s", r.URL))
				}
				resp, ok := m[key]
				if ok {
					_, _ = w.Write(resp)
				} else {
					t.Fatalf("Key not found in response data: %q", key)
				}
			}),
		},
		{
			name: "WithPermissions",
			options: []Option{
				WithInstallationID(apitestdata.InstallationID),
				WithPermissions("metadata:read"),
			},
			ok:     true,
			scopes: map[string]string{"metadata": "read"},
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var key string
				switch r.URL.Path {
				case "/app":
					key = "get-app"
				case fmt.Sprintf("/app/installations/%d", apitestdata.InstallationID):
					key = "get-installation-by-id"
				case fmt.Sprintf("/app/installations/%d/access_tokens", apitestdata.InstallationID):
					key = "post-installation-token-with-scopes"
					w.WriteHeader(http.StatusCreated)
				case fmt.Sprintf("/users/%s[bot]", apitestdata.AppSlug):
					key = "get-user-bot"
				default:
					panic(fmt.Sprintf("Unknown/Invalid Request => %s", r.URL))
				}
				resp, ok := m[key]
				if ok {
					_, _ = w.Write(resp)
				} else {
					panic(fmt.Sprintf("Response key not found %s", key))
				}
			}),
		},
		{
			name: "WithPermissionsNotAvailable",
			options: []Option{
				WithInstallationID(apitestdata.InstallationID),
				WithPermissions("actions:read"),
			},
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var key string
				switch r.URL.Path {
				case "/app":
					key = "get-app"
				case fmt.Sprintf("/app/installations/%d", apitestdata.InstallationID):
					key = "get-installation-by-id"
				case fmt.Sprintf("/app/installations/%d/access_tokens", apitestdata.InstallationID):
					key = "post-installation-token-with-scopes"
					w.WriteHeader(http.StatusCreated)
				case fmt.Sprintf("/users/%s[bot]", apitestdata.AppSlug):
					key = "get-user-bot"
				default:
					panic(fmt.Sprintf("Unknown/Invalid Request => %s", r.URL))
				}
				resp, ok := m[key]
				if ok {
					_, _ = w.Write(resp)
				} else {
					panic(fmt.Sprintf("Response key not found %s", key))
				}
			}),
		},
		{
			name: "ErrorInvalidAppKey",
			options: []Option{
				WithInstallationID(apitestdata.InstallationID),
			},
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var key string
				switch r.URL.Path {
				case "/app":
					w.WriteHeader(http.StatusUnauthorized)
					key = "error-invalid-jwt"
				default:
					panic(fmt.Sprintf("Unknown/Invalid Request => %s", r.URL))
				}
				resp, ok := m[key]
				if ok {
					_, _ = w.Write(resp)
				} else {
					panic(fmt.Sprintf("Response key not found %s", key))
				}
			}),
		},
		{
			name: "WithServerError",
			options: []Option{
				WithInstallationID(apitestdata.InstallationID),
			},
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			}),
		},
		{
			name: "InstallationHasNoAccess",
			options: []Option{
				WithRepositories(
					fmt.Sprintf("%s/%s",
						apitestdata.InstallationOwner,
						apitestdata.InstallationRepository),
				),
			},
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var key string
				switch r.URL.Path {
				case "/app":
					key = "get-app"
				case fmt.Sprintf("/repos/%s/%s/installation",
					apitestdata.InstallationOwner, apitestdata.InstallationRepository):
					key = "get-installation-by-repo"
				case fmt.Sprintf("/app/installations/%d/access_tokens", apitestdata.InstallationID):
					key = "error-installation-token-no-access"
					w.WriteHeader(http.StatusUnprocessableEntity)
				case fmt.Sprintf("/users/%s[bot]", apitestdata.AppSlug):
					key = "get-user-bot"
				default:
					panic(fmt.Sprintf("Unknown/Invalid Request => %s", r.URL))
				}
				resp, ok := m[key]
				if ok {
					_, _ = w.Write(resp)
				} else {
					panic(fmt.Sprintf("Response key not found %s", key))
				}
			}),
		},
		{
			name: "GetInstallation-InstallationDisabled",
			options: []Option{
				WithInstallationID(apitestdata.InstallationID),
			},
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var key string
				switch r.URL.Path {
				case "/app":
					key = "get-app"
				case fmt.Sprintf("/app/installations/%d", apitestdata.InstallationID):
					key = "get-installation-disabled"
				default:
					panic(fmt.Sprintf("Unknown/Invalid Request => %s", r.URL))
				}
				resp, ok := m[key]
				if ok {
					_, _ = w.Write(resp)
				} else {
					panic(fmt.Sprintf("Response key not found %s", key))
				}
			}),
		},
		{
			name: "GetInstallation-NotFound",
			options: []Option{
				WithInstallationID(apitestdata.InstallationID),
			},
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var key string
				switch r.URL.Path {
				case "/app":
					key = "get-app"
				case fmt.Sprintf("/app/installations/%d", apitestdata.InstallationID):
					key = "error-not-found"
					w.WriteHeader(http.StatusNotFound)
				default:
					panic(fmt.Sprintf("Unknown/Invalid Request => %s", r.URL))
				}
				resp, ok := m[key]
				if ok {
					_, _ = w.Write(resp)
				} else {
					panic(fmt.Sprintf("Response key not found %s", key))
				}
			}),
		},
		{
			name: "GetInstallation-ServerError",
			options: []Option{
				WithInstallationID(apitestdata.InstallationID),
			},
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var key string
				switch r.URL.Path {
				case "/app":
					key = "get-app"
				case fmt.Sprintf("/app/installations/%d", apitestdata.InstallationID):
					w.WriteHeader(http.StatusServiceUnavailable)
					return
				default:
					panic(fmt.Sprintf("Unknown/Invalid Request => %s", r.URL))
				}
				resp, ok := m[key]
				if ok {
					_, _ = w.Write(resp)
				} else {
					panic(fmt.Sprintf("Response key not found %s", key))
				}
			}),
		},
		{
			name: "GetBotUser-NotFound",
			options: []Option{
				WithInstallationID(apitestdata.InstallationID),
			},
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var key string
				switch r.URL.Path {
				case "/app":
					key = "get-app"
				case fmt.Sprintf("/app/installations/%d", apitestdata.InstallationID):
					key = "get-installation-by-id"
				case fmt.Sprintf("/users/%s/installation", apitestdata.InstallationOwner):
					key = "get-installation-by-user"
				case fmt.Sprintf("/repos/%s/%s/installation",
					apitestdata.InstallationOwner, apitestdata.InstallationRepository):
					key = "get-installation-by-repo"
				case fmt.Sprintf("/app/installations/%d/access_tokens", apitestdata.InstallationID):
					key = "post-installation-token"
					w.WriteHeader(http.StatusCreated)
				case fmt.Sprintf("/users/%s[bot]", apitestdata.AppSlug):
					key = "error-not-found"
					w.WriteHeader(http.StatusNotFound)
				default:
					panic(fmt.Sprintf("Unknown/Invalid Request => %s", r.URL))
				}
				resp, ok := m[key]
				if ok {
					_, _ = w.Write(resp)
				} else {
					panic(fmt.Sprintf("Response key not found %s", key))
				}
			}),
		},
		{
			name: "GetBotUser-ServerError",
			options: []Option{
				WithInstallationID(apitestdata.InstallationID),
			},
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var key string
				switch r.URL.Path {
				case "/app":
					key = "get-app"
				case fmt.Sprintf("/app/installations/%d", apitestdata.InstallationID):
					key = "get-installation-by-id"
				case fmt.Sprintf("/users/%s/installation", apitestdata.InstallationOwner):
					key = "get-installation-by-user"
				case fmt.Sprintf("/repos/%s/%s/installation",
					apitestdata.InstallationOwner, apitestdata.InstallationRepository):
					key = "get-installation-by-repo"
				case fmt.Sprintf("/app/installations/%d/access_tokens", apitestdata.InstallationID):
					key = "post-installation-token"
					w.WriteHeader(http.StatusCreated)
				case fmt.Sprintf("/users/%s[bot]", apitestdata.AppSlug):
					w.WriteHeader(http.StatusServiceUnavailable)
					return
				default:
					panic(fmt.Sprintf("Unknown/Invalid Request => %s", r.URL))
				}
				resp, ok := m[key]
				if ok {
					_, _ = w.Write(resp)
				} else {
					panic(fmt.Sprintf("Response key not found %s", key))
				}
			}),
		},
	}
	ctx := context.Background()
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewUnstartedServer(tc.handler)
			t.Logf("Running test server - %s", server.URL)

			server.Start()
			defer server.Close()

			options := slices.Clone(tc.options)
			options = append(
				options,
				WithEndpoint(server.URL),
			)
			token, err := NewInstallationToken(ctx,
				apitestdata.AppID,
				testkeys.RSA2048(),
				options...,
			)

			if tc.ok {
				if err != nil {
					t.Errorf("expected no error, got %s", err)
				}

				// Ideally we could use token.IsValid, but json
				// responses are not dynamic. so fallback to check
				// important fields.
				if token.Token == "" {
					t.Errorf("expected token to be non empty")
				}

				if token.BotUsername == "" {
					t.Errorf("expected BotUsername to be non empty")
				}

				if token.BotCommitterEmail == "" {
					t.Errorf("expected BotCommitterEmail to be non empty")
				}

				if token.InstallationID == 0 {
					t.Errorf("expected InstallationID to be non zero")
				}

				if token.AppID == 0 {
					t.Errorf("expected AppID to be non zero")
				}

				if token.Exp.IsZero() {
					t.Errorf("expected Exp to be non zero")
				}

				if len(token.Repositories) != len(tc.repos) {
					t.Errorf("expected repos(len)=%d, got(len)=%d",
						len(token.Repositories), len(tc.repos))
				}

				if !maps.Equal(tc.scopes, token.Permissions) {
					t.Errorf("expected scopes=%v, got=%v",
						tc.scopes, token.Permissions)
				}
			} else {
				if err == nil {
					t.Errorf("expected an error, got nil")
				}

				if !reflect.DeepEqual(token, InstallationToken{}) {
					t.Errorf("expected token to be empty")
				}
			}
		})
	}
}
