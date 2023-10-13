// SPDX-FileCopyrightText: Copyright 2023 Prasad Tengse
// SPDX-License-Identifier: MIT

package githubapp

import (
	"net/http"
	"net/url"
	"slices"
	"testing"

	"github.com/tprasadtp/go-githubapp/internal"
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
		urlString := "https://api.endpoint.test"
		urlURL, _ := url.Parse("https://api.endpoint.test")
		transport := Transport{}
		expect := Transport{
			owner:     "username",
			repos:     []string{"bar", "foo"},
			baseURL:   urlURL,
			installID: 99,
			scopes: map[string]string{
				"issues":   "write",
				"contents": "read",
				"metadata": "read",
			},
		}
		opts := Options(
			WithEndpoint(urlString),
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
		name   string
		input  []string
		expect []string // must be sorted
		owner  string
		ok     bool
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
			name:   "valid-no-owner",
			input:  []string{"foo", "bar"},
			owner:  "",
			expect: []string{"bar", "foo"},
			ok:     true,
		},
		{
			name:   "valid-no-owner-deduplicate",
			input:  []string{"foo", "bar", "foo"},
			owner:  "",
			expect: []string{"bar", "foo"},
			ok:     true,
		},
		{
			name:   "valid-deduplicate",
			input:  []string{"username/foo", "username/bar", "username/foo"},
			owner:  "username",
			expect: []string{"bar", "foo"},
			ok:     true,
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

				if !slices.Equal(tc.expect, transport.repos) {
					t.Errorf("expected Transport.repos=%v, got=%v", tc.expect, transport.repos)
				}
			} else {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			}
		})
	}
}

func TestWithOwner(t *testing.T) {
	type testCase struct {
		name   string
		input  string
		expect string
		ok     bool
	}
	tt := []testCase{
		{
			name:  "with-single-dot",
			input: ".",
		},
		{
			name:  "with-empty-string",
			input: "",
		},
		{
			name:  "with-spaces",
			input: "   ",
		},
		{
			name:  "username-starts-with-dash",
			input: "-username",
		},
		{
			name:  "hash-dots",
			input: "user.name",
		},
		{
			name:   "username-ends-with-dash",
			input:  "user-",
			expect: "user-",
			ok:     true,
		},
		{
			name:   "username-has-dashes",
			input:  "user-name-org",
			expect: "user-name-org",
			ok:     true,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			transport := Transport{}
			opt := WithOwner(tc.input)
			err := opt.apply(&transport)
			if tc.ok {
				if err != nil {
					t.Fatalf("unexpected error %s", err)
				}

				if tc.expect != transport.owner {
					t.Errorf("expected Transport.owner=%s, got=%s", tc.expect, transport.owner)
				}
			} else {
				if err == nil {
					t.Errorf("expected error, got nil")
				}

				if transport.owner != "" {
					t.Errorf("on error transport.owner must be empty")
				}
			}
		})
	}

	t.Run("multiple-owners-conflicting", func(t *testing.T) {
		transport := Transport{}
		opts := Options(WithOwner("git"), WithOwner("github"))
		err := opts.apply(&transport)
		if err == nil {
			t.Errorf("expected error, got nil")
		}
	})
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
		if transport.baseURL != nil {
			t.Errorf("transport baseURL should not be modified")
		}
	})
	t.Run("url-has-fragments", func(t *testing.T) {
		transport := Transport{}
		opts := Options(WithEndpoint("https://api-endpoint-golang.test/endpoint#foo"))
		err := opts.apply(&transport)
		if err == nil {
			t.Errorf("expected error, got nil")
		}
		if transport.baseURL != nil {
			t.Errorf("transport baseURL should not be modified")
		}
	})

	t.Run("url-has-queries", func(t *testing.T) {
		transport := Transport{}
		opts := Options(WithEndpoint("https://api-endpoint-golang.test/endpoint?foo=bar"))
		err := opts.apply(&transport)
		if err == nil {
			t.Errorf("expected error, got nil")
		}
		if transport.baseURL != nil {
			t.Errorf("transport baseURL should not be modified")
		}
	})
	t.Run("url-invalid-1", func(t *testing.T) {
		transport := Transport{}
		opts := Options(WithEndpoint("https://url is invalid/"))
		err := opts.apply(&transport)
		if err == nil {
			t.Errorf("expected error, got nil")
		}
		if transport.baseURL != nil {
			t.Errorf("transport baseURL should not be modified")
		}
	})
	t.Run("url-invalid-2", func(t *testing.T) {
		transport := Transport{}
		opts := Options(WithEndpoint("https://url-is-#invalid/"))
		err := opts.apply(&transport)
		if err == nil {
			t.Errorf("expected error, got nil")
		}
		if transport.baseURL != nil {
			t.Errorf("transport baseURL should not be modified")
		}
	})

	t.Run("url-valid-default", func(t *testing.T) {
		transport := Transport{}
		opts := Options(WithEndpoint(defaultEndpoint))
		err := opts.apply(&transport)
		if err != nil {
			t.Errorf("expected no error, got %s", err)
		}
		if transport.baseURL.String() != defaultEndpoint {
			t.Errorf("transport baseURL should be %s, got %s",
				defaultEndpoint, transport.baseURL)
		}
	})
}

func TestWithPermissions(t *testing.T) {
	t.Run("invalid-scope-levels", func(t *testing.T) {
		transport := Transport{}
		opts := Options(WithPermissions("issues:read", "contents:foo"))
		err := opts.apply(&transport)
		if err == nil {
			t.Errorf("expected an error, got nil")
		}
		if transport.scopes != nil {
			t.Errorf("transport.scopes should be nil: %v", transport.scopes)
		}
	})
	t.Run("invalid-scope-format", func(t *testing.T) {
		transport := Transport{}
		opts := Options(WithPermissions("issues=read", "contents=foo"))
		err := opts.apply(&transport)
		if err == nil {
			t.Errorf("expected an error, got nil")
		}
		if transport.scopes != nil {
			t.Errorf("transport.scopes should be nil: %v", transport.scopes)
		}
	})
	t.Run("nil-scopes", func(t *testing.T) {
		opts := Options(WithPermissions())
		if opts != nil {
			t.Errorf("expected nil options when no scopes are specified")
		}
	})
}

func TestWithRoundTripper(t *testing.T) {
	t.Run("non-nil", func(t *testing.T) {
		transport := Transport{}
		opts := Options(WithRoundTripper(
			internal.RoundTripFunc(func(r *http.Request) (*http.Response, error) {
				t.Logf("request=%v", r)
				return http.DefaultTransport.RoundTrip(r)
			})))
		err := opts.apply(&transport)
		if err != nil {
			t.Errorf("expected no error, got %s", err)
		}

		if transport.next == nil {
			t.Errorf("transport.next should be non nil")
		}
	})
	t.Run("nil-round-tripper", func(t *testing.T) {
		opts := Options(WithRoundTripper(nil))
		if opts != nil {
			t.Errorf("expected nil options when no round tripper is specified")
		}
	})
}

func TestWithInstallationID(t *testing.T) {
	t.Run("zero", func(t *testing.T) {
		transport := Transport{}
		opts := Options(WithInstallationID(0))
		err := opts.apply(&transport)
		if err == nil {
			t.Errorf("expected an error, got nil")
		}
	})

	t.Run("multiple-conflicting-ids", func(t *testing.T) {
		transport := Transport{}
		opts := Options(WithInstallationID(99), WithInstallationID(9))
		err := opts.apply(&transport)
		if err == nil {
			t.Errorf("expected an error, got nil")
		}
	})

	t.Run("multiple-same", func(t *testing.T) {
		transport := Transport{}
		opts := Options(WithInstallationID(99), WithInstallationID(99))
		err := opts.apply(&transport)
		if err != nil {
			t.Errorf("expected no error, got %s", err)
		}

		if transport.installID != 99 {
			t.Errorf("expected instalaltion id to be 99, got %d", transport.installID)
		}
	})
}
