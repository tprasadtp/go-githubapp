// SPDX-FileCopyrightText: Copyright 2023 Prasad Tengse
// SPDX-License-Identifier: MIT

package githubapp

import (
	"context"
	"crypto"
	"maps"
	"reflect"
	"slices"
	"testing"

	"github.com/tprasadtp/go-githubapp/internal/testkeys"
)

// transportCmp compares two transports. But ignores some fields.
func transportCmp(t *testing.T, a, b *Transport) bool {
	t.Helper()

	if a == nil && b == nil {
		t.Logf("both transport are nil")
		return true
	}

	if a == nil && b != nil {
		t.Logf("one transport is nil, other is not")
		return false
	}

	if b == nil && a != nil {
		t.Logf("one transport is nil, other is not")
		return false
	}

	if a.appID != b.appID {
		t.Logf("a.appID=%d, b.appID=%d", a.appID, b.appID)
		return false
	}

	if a.appSlug != b.appSlug {
		t.Logf("a.appSlug=%s, b.appSlug=%s", a.appSlug, b.appSlug)
		return false
	}

	if a.installID != b.installID {
		t.Logf("a.installID=%d, b.installID=%d", a.installID, b.installID)
		return false
	}

	if !reflect.DeepEqual(a.baseURL, b.baseURL) {
		t.Logf("a.baseURL=%s, b.baseURL=%s", a.baseURL, b.baseURL)
		return false
	}

	if !reflect.DeepEqual(a.next, b.next) {
		t.Logf("a.next=%#v, b.next=%#v", a.next, b.next)
		return false
	}

	if a.owner != b.owner {
		t.Logf("a.owner=%s, b.owner=%s", a.owner, b.owner)
		return false
	}

	if !slices.Equal(a.repos, b.repos) {
		t.Logf("a.next=%v, b.repos=%v", a.repos, b.repos)
		return false
	}

	if !maps.Equal(a.scopes, b.scopes) {
		t.Logf("a.scopes=%v, b.scopes=%v", a.scopes, b.scopes)
		return false
	}

	return true
}

func TestCtxJWT(t *testing.T) {
	ctx := context.Background()

	if ctxHasKeyJWT(ctx) {
		t.Errorf("context.Background() should not have a value")
	}

	clone := ctxWithJWTKey(ctx)
	value := clone.Value(keyJWT{})
	if value == nil {
		t.Errorf("ctxWithJWTKey(ctx).Value(keyJWT{}) should return non nil value")
	}

	if !ctxHasKeyJWT(clone) {
		t.Errorf("ctxHasKeyJWT(ctxWithJWTKey(ctx)) should return true")
	}
}

func TestNewTransport(t *testing.T) {
	tt := []struct {
		name    string
		ok      bool
		appID   uint64
		signer  crypto.Signer
		options []Option
		expect  *Transport
	}{
		{
			name: "no-signer",
		},
		{
			name:   "no-app-id",
			signer: testkeys.RSA2048(),
		},
		{
			name:    "endpoint-unsupported-scheme",
			signer:  testkeys.RSA2048(),
			options: []Option{WithEndpoint("file://")},
			appID:   99,
		},
		{
			name:    "endpoint-with-query",
			signer:  testkeys.RSA2048(),
			options: []Option{WithEndpoint("https://localhost:9999/foo?test=1")},
			appID:   99,
		},
		{
			name:    "endpoint-with-fragment",
			signer:  testkeys.RSA2048(),
			options: []Option{WithEndpoint("https://localhost:9999/foo#Fragment")},
			appID:   99,
		},
		{
			name:    "owner-invalid-name-empty",
			signer:  testkeys.RSA2048(),
			options: []Option{WithOwner("")},
			appID:   99,
		},
		{
			name:    "owner-invalid-name-has-dots",
			signer:  testkeys.RSA2048(),
			options: []Option{WithOwner("foo.bar")},
			appID:   99,
		},
		{
			name:    "owner-invalid-name-has-special-chars",
			signer:  testkeys.RSA2048(),
			options: []Option{WithOwner("foo?")},
			appID:   99,
		},
		{
			name:    "owner-invalid-name-end-with-dot",
			signer:  testkeys.RSA2048(),
			options: []Option{WithOwner("foo.")},
			appID:   99,
		},
		{
			name:    "repo-invalid-with-special-char",
			signer:  testkeys.RSA2048(),
			options: []Option{WithOwner("username"), WithRepositories("foo?")},
			appID:   99,
		},
		{
			name:    "repo-invalid-only-dot-is-reserved",
			signer:  testkeys.RSA2048(),
			options: []Option{WithRepositories("foo/.")},
			appID:   99,
		},
		{
			name:    "repo-invalid-dot-with-special-char",
			signer:  testkeys.RSA2048(),
			options: []Option{WithRepositories("foo/.=")},
			appID:   99,
		},
		{
			name:    "repo-invalid-no-owner-no-install-id",
			signer:  testkeys.RSA2048(),
			options: []Option{WithRepositories("foo", "bar")},
			appID:   99,
		},
		{
			name:    "repo-unsupported-key-ecdsa",
			signer:  testkeys.ECP256(),
			options: []Option{WithRepositories("foo/bar", "foo/baz")},
			appID:   99,
		},
		{
			name:    "repo-unsupported-key-ed25519",
			signer:  testkeys.ED25519(),
			options: []Option{WithRepositories("foo/bar", "foo/baz")},
			appID:   99,
		},
		{
			name:    "repo-unsupported-key-rsa-1024",
			signer:  testkeys.RSA1024(),
			options: []Option{WithRepositories("foo/bar", "foo/baz")},
			appID:   99,
		},
		{
			name:    "endpoint-invalid-url",
			signer:  testkeys.RSA2048(),
			options: []Option{WithEndpoint("file://  foo/bar")},
			appID:   99,
		},
		{
			name:    "endpoint-unreachable",
			signer:  testkeys.RSA2048(),
			options: []Option{WithEndpoint("http://308489a4-2f67-4d6a-9d8a-11d21f44bfa0")},
			appID:   99,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			transport, err := NewTransport(
				context.Background(),
				tc.appID,
				tc.signer,
				tc.options...,
			)

			if tc.ok {
				if err != nil {
					t.Errorf("expected no error, got %s", err)
				}
				if !transportCmp(t, tc.expect, transport) {
					t.Errorf("expected:%#v, got=%#v", tc.expect, transport)
				}
			} else {
				if err == nil {
					t.Errorf("expected an error, got nil")
				}

				if transport != nil {
					t.Errorf("if error is expected to be not nil, transport must be nil")
				}
			}
		})
	}
}

func TestTransport_checkInstallationPermissions(t *testing.T) {
	type testCase struct {
		name        string
		permissions map[string]string
		scopes      map[string]string
		ok          bool
	}
	tt := []testCase{
		{
			name: "invalid-missing-from-install",
			permissions: map[string]string{
				"contents": "read",
			},
			scopes: map[string]string{
				"actions": "write",
			},
		},
		{
			name: "invalid-all-scopes-missing",
			permissions: map[string]string{
				"metadata": "read",
			},
			scopes: map[string]string{
				"actions":  "write",
				"contents": "write",
				"issues":   "read",
			},
		},
		{
			name: "invalid-has-project-write-but-scope-admin",
			permissions: map[string]string{
				"metadata": "read",
				"projects": "write",
			},
			scopes: map[string]string{
				"projects": "admin",
			},
		},
		{
			name: "invalid-has-contents-read-but-scope-write",
			permissions: map[string]string{
				"metadata": "read",
				"contents": "read",
			},
			scopes: map[string]string{
				"contents": "write",
			},
		},
		{
			name: "invalid-unknown-scope-level",
			permissions: map[string]string{
				"metadata": "read",
				"contents": "read",
			},
			scopes: map[string]string{
				"contents": "unknown_scope",
			},
		},
		{
			name: "invalid-unknown-install-level",
			permissions: map[string]string{
				"metadata": "read",
				"contents": "unknown_scope",
			},
			scopes: map[string]string{
				"contents": "read",
			},
		},
		{
			name: "valid-empty-scope",
			permissions: map[string]string{
				"contents": "read",
			},
			ok: true,
		},
		{
			name: "valid-same-scope",
			permissions: map[string]string{
				"contents": "read",
			},
			scopes: map[string]string{
				"contents": "read",
			},
			ok: true,
		},
		{
			name: "valid-less-scopes",
			permissions: map[string]string{
				"contents": "write",
				"projects": "admin",
			},
			scopes: map[string]string{
				"contents": "read",
				"projects": "write",
			},
			ok: true,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			transport := Transport{
				scopes: tc.scopes,
			}
			err := transport.checkInstallationPermissions(tc.permissions)
			if tc.ok {
				if err != nil {
					t.Errorf("unexpected error: %s", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			}
		})
	}
}

func TestTransport_JWT(t *testing.T) {
	ctx := context.Background()

	t.Run("no-existing-jwt", func(t *testing.T) {
		transport := &Transport{
			appID:  99,
			minter: &jwtRS256{internal: testkeys.RSA2048()},
		}
		t.Logf("Minting first JWT")
		jwt1, err := transport.JWT(ctx)
		if err != nil {
			t.Errorf("unexpected error minting fresh jwt: %s", err)
		}

		if transport.bearer.Load() == nil {
			t.Errorf("saved bearer token is nil")
		}

		jwt2, err := transport.JWT(ctx)
		if err != nil {
			t.Errorf("unexpected error getting existing jwt: %s", err)
		}

		if !reflect.DeepEqual(jwt1, jwt2) {
			t.Errorf("calling JWT() twice in short interval must return same JWT")
		}
	})

	t.Run("refresh-invalid-jwt", func(t *testing.T) {
		transport := &Transport{
			appID:  99,
			minter: &jwtRS256{internal: testkeys.RSA2048()},
		}
		t.Logf("Minting first JWT")
		jwt1, err := transport.JWT(ctx)
		if err != nil {
			t.Errorf("unexpected error minting fresh jwt: %s", err)
		}

		if transport.bearer.Load() == nil {
			t.Errorf("saved bearer token is nil")
		}

		jwt2, err := transport.JWT(ctx)
		if err != nil {
			t.Errorf("unexpected error getting existing jwt: %s", err)
		}

		if !reflect.DeepEqual(jwt1, jwt2) {
			t.Errorf("calling JWT() twice in short interval must return same JWT")
		}
	})

	t.Run("signer-errors", func(t *testing.T) {
		transport := &Transport{
			appID:  99,
			minter: &jwtRS256{internal: &errSigner{signer: testkeys.RSA2048()}},
		}
		token, err := transport.JWT(ctx)
		if err == nil {
			t.Errorf("expected error on a signer which always errors")
		}
		if !reflect.DeepEqual(token, JWT{}) {
			t.Errorf("on error JWT should returns empty jwt")
		}
	})
}
