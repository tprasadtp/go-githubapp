// SPDX-FileCopyrightText: Copyright 2023 Prasad Tengse
// SPDX-License-Identifier: MIT

package githubapp

import (
	"testing"
)

func TestOptions_Nils(t *testing.T) {
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
}

func TestWithRepositories(t *testing.T) {
	type testCase struct {
		name string
		repo []string
		ok   bool
	}
	tt := []testCase{
		{
			name: "with-single-dot",
			repo: []string{"."},
		},
		{
			name: "with-single-dot-and-username",
			repo: []string{"username/."},
		},
		{
			name: "repo-name-invalid-1",
			repo: []string{"username/repo?"},
		},
		{
			name: "repo-name-invalid-2",
			repo: []string{"username/.github foo"},
		},
		{
			name: "invalid-username-1",
			repo: []string{"*username/.github"},
		},
		{
			name: "invalid-username-2",
			repo: []string{"user name/.github"},
		},
		{
			name: "invalid-username-3",
			repo: []string{"user.name/.github"},
		},
		{
			name: "owner-mismatch",
			repo: []string{"user/repo-1", "user/repo-2", "another-user/repo-1"},
		},
		{
			name: "valid-no-owner",
			repo: []string{"foo", "bar"},
			ok:   true,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			opt := WithRepositories(tc.repo...)
			err := opt.apply(&Transport{})
			if tc.ok {
				if err != nil {
					t.Errorf("unexpected error %s", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			}
		})
	}
}
