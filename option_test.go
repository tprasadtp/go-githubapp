// SPDX-FileCopyrightText: Copyright 2023 Prasad Tengse
// SPDX-License-Identifier: MIT

package githubapp

import (
	"slices"
	"testing"
)

func TestOptions(t *testing.T) {
	t.Run("all-nils", func(t *testing.T) {
		if Options(nil, nil, WithEndpoint(""), WithRepositories()) != nil {
			t.Errorf("expected nil")
		}
	})

	t.Run("nil-round-tripper", func(t *testing.T) {
		if WithRoundTripper(nil) != nil {
			t.Errorf("WithRoundTripper with nil round-tripper must return nil")
		}
	})

	t.Run("no-repos", func(t *testing.T) {
		if WithRepositories() != nil {
			t.Errorf("WithRepositories with no-args  must return nil")
		}
	})

	t.Run("no-permissions", func(t *testing.T) {
		if WithPermissions() != nil {
			t.Errorf("WithPermissions with no-args  must return nil")
		}
	})

	t.Run("no-endpoint", func(t *testing.T) {
		if WithEndpoint("") != nil {
			t.Errorf("WithEndpoint with empty string must return nil")
		}
	})

	t.Run("all-non-nils", func(t *testing.T) {
		transport := Transport{}
		expect := Transport{
			owner:     "username",
			repos:     []string{"bar", "foo"},
			endpoint:  "https://api.endpoint.test",
			installID: 99,
			scopes: map[string]string{
				"issues":   "write",
				"contents": "read",
				"metadata": "read",
			},
		}
		opts := Options(
			WithEndpoint("https://api.endpoint.test"),
			WithOwner("username"),
			WithRepositories("username/foo", "username/bar"),
			WithInstallationID(99),
			WithPermissions("issues:write", "contents:read", "metadata:read"),
		)
		err := opts.apply(&transport)
		if err != nil {
			t.Errorf("expected no error, got %s", err)
		}
		if !transportCmp(t, &expect, &transport) {
			t.Errorf("transport not equal")
		}
	})
}

func TestWithRepositories(t *testing.T) {
	type testCase struct {
		name  string
		input []string
		repos []string // must be sorted
		owner string
		ok    bool
	}
	tt := []testCase{
		{
			name:  "with-single-dot",
			input: []string{"."},
		},
		{
			name:  "with-single-dot-and-username",
			input: []string{"username/."},
		},
		{
			name:  "repo-name-invalid-1",
			input: []string{"username/repo?"},
		},
		{
			name:  "repo-name-invalid-2",
			input: []string{"username/.github foo"},
		},
		{
			name:  "invalid-username-1",
			input: []string{"*username/.github"},
		},
		{
			name:  "invalid-username-2",
			input: []string{"user name/.github"},
		},
		{
			name:  "invalid-username-3",
			input: []string{"user.name/.github"},
		},
		{
			name:  "owner-mismatch",
			input: []string{"user/repo-1", "user/repo-2", "another-user/repo-1"},
		},
		{
			name:  "valid-no-owner",
			input: []string{"foo", "bar"},
			owner: "",
			repos: []string{"foo", "bar"},
			ok:    true,
		},
		{
			name:  "valid-no-owner-deduplicate",
			input: []string{"foo", "bar", "foo"},
			owner: "",
			repos: []string{"bar", "foo"},
			ok:    true,
		},
		{
			name:  "valid-deduplicate",
			input: []string{"username/foo", "username/bar", "username/foo"},
			owner: "username",
			repos: []string{"username/bar", "username/foo"},
			ok:    true,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			transport := Transport{}
			opt := WithRepositories(tc.input...)
			err := opt.apply(&transport)
			if tc.ok {
				if err != nil {
					t.Fatalf("unexpected error %s", err)
				}

				if tc.owner != transport.owner {
					t.Errorf("expected Transport.owner=%s, got=%s", tc.owner, transport.owner)
				}

				if slices.Equal(tc.repos, transport.repos) {
					t.Errorf("expected Transport.repos=%v, got=%v", tc.repos, transport.repos)
				}
			} else {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			}
		})
	}
}

func TestWithEndpoint(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		opts := Options(WithEndpoint(""))
		if opts != nil {
			t.Errorf("on empty endpoint options should return nil")
		}
	})
	t.Run("invalid-protocol", func(t *testing.T) {
		transport := Transport{}
		opts := Options(WithEndpoint("ftp://endpoint.api-endpoint-golang.test"))
		err := opts.apply(&transport)
		if err == nil {
			t.Errorf("expected error, got nil")
		}
		if transport.endpoint != "" {
			t.Errorf("transport endpoint should not be modified")
		}
	})
	t.Run("url-has-fragments", func(t *testing.T) {
		transport := Transport{}
		opts := Options(WithEndpoint("https://api-endpoint-golang.test/endpoint#foo"))
		err := opts.apply(&transport)
		if err == nil {
			t.Errorf("expected error, got nil")
		}
		if transport.endpoint != "" {
			t.Errorf("transport endpoint should not be modified")
		}
	})

	t.Run("url-has-queries", func(t *testing.T) {
		transport := Transport{}
		opts := Options(WithEndpoint("https://api-endpoint-golang.test/endpoint?foo=bar"))
		err := opts.apply(&transport)
		if err == nil {
			t.Errorf("expected error, got nil")
		}
		if transport.endpoint != "" {
			t.Errorf("transport endpoint should not be modified")
		}
	})
	t.Run("url-invalid-1", func(t *testing.T) {
		transport := Transport{}
		opts := Options(WithEndpoint("https://url is invalid/"))
		err := opts.apply(&transport)
		if err == nil {
			t.Errorf("expected error, got nil")
		}
		if transport.endpoint != "" {
			t.Errorf("transport endpoint should not be modified")
		}
	})
	t.Run("url-invalid-2", func(t *testing.T) {
		transport := Transport{}
		opts := Options(WithEndpoint("https://url-is-#invalid/"))
		err := opts.apply(&transport)
		if err == nil {
			t.Errorf("expected error, got nil")
		}
		if transport.endpoint != "" {
			t.Errorf("transport endpoint should not be modified")
		}
	})

	t.Run("url-valid", func(t *testing.T) {
		transport := Transport{}
		opts := Options(WithEndpoint(defaultEndpoint))
		err := opts.apply(&transport)
		if err != nil {
			t.Errorf("expected no error, got %s", err)
		}
		if transport.endpoint != defaultEndpoint {
			t.Errorf("transport endpoint should be: %s", defaultEndpoint)
		}
	})
}
