// SPDX-FileCopyrightText: Copyright 2023 Prasad Tengse
// SPDX-License-Identifier: MIT

package githubapp

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
)

// Options takes a variadic slice of [Options] and returns
// a single [Options] which includes all the given options.
// This is useful for sharing presets. If conflicting options
// are specified, last one specified wins. As a special case,
// if no options are specified or all specified options are nil,
// this will return nil.
func Options(options ...Option) Option {
	nils := 0
	for i := range options {
		if options[i] == nil {
			nils++
		}
	}
	if len(options) == nils {
		return nil
	}

	return &funcOption{
		f: func(t *Transport) error {
			var err error
			for i := range options {
				// Not all platforms support all options,
				// on unsupported platform option function may
				// return nil.
				if options[i] != nil {
					err = errors.Join(options[i].apply(t))
				}
			}
			return err
		},
	}
}

// Option is option to apply for [Transport].
type Option interface {
	apply(t *Transport) error
}

// funcOption wraps a function that is applied to the Transport
// during its initial configuration. It implements [Option]
// interface.
type funcOption struct {
	f func(*Transport) error
}

func (opt *funcOption) apply(t *Transport) error {
	return opt.f(t)
}

var (
	repoNameRegExp  = regexp.MustCompile("^(((.)[a-z-0-9-.]+)|([a-z0-9-]([a-z0-9-.]+)?))$")
	userNameRegExp  = regexp.MustCompile("^([a-z0-9]([a-z0-9-]+)?)$")
	permissionRegEx = regexp.MustCompile("^[a-z]([a-z_]+[a-z])?[:|=](read|write|admin)$")
)

// WithEndpoint configures [Transport] to use custom REST API(v3) endpoint.
// for authenticating as app, obtaining installation metadata and creating
// installation access tokens. This MUST be REST(v3) endpoint even though
// client might be using GitHub GraphQL API.
//
// When not specified or empty, "https://api.github.com/" is used.
func WithEndpoint(endpoint string) Option {
	if endpoint == "" {
		return nil
	}
	return &funcOption{
		f: func(t *Transport) error {
			u, err := url.Parse(endpoint)
			if err != nil {
				return fmt.Errorf("invalid endpoint url: %w", err)
			}
			switch u.Scheme {
			case "http", "https":
			default:
				return fmt.Errorf("invalid url scheme : %s (%s)", u.Scheme, endpoint)
			}

			if u.Fragment != "" || u.RawQuery != "" {
				return fmt.Errorf("endpoint cannot have fragments in endpoint URL: %s", endpoint)
			}

			t.baseURL = u
			return nil
		},
	}
}

// WithRoundTripper configures [Transport] to use next as next [http.RoundTripper].
//
// This can be used to further customize headers, add logging or retries. This only
// applies to authentication API calls and not the http client using the [Transport].
func WithRoundTripper(next http.RoundTripper) Option {
	if next == nil {
		return nil
	}
	return &funcOption{
		f: func(t *Transport) error {
			t.next = next
			return nil
		},
	}
}

// WithUserAgent configures user agent header to use for token related API requests.
//
// Typically [Transport] which implements [http.RoundTripper] will re-use the User-Agent
// header specified by the [http.Request]. However, when building the [Transport] several
// HTTP requests need to be made to verify and configure it. User agent specified here
// will be used during bootstrapping. This is also as fallback for token renewal requests.
func WithUserAgent(ua string) Option {
	if strings.TrimSpace(ua) == "" {
		return nil
	}
	return &funcOption{
		f: func(t *Transport) error {
			t.ua = ua
			return nil
		},
	}
}

// WithRepositories configures [Transport] to use installation for repos specified.
// Unlike other installation options, this can be used multiple times.
func WithRepositories(repos ...string) Option {
	if len(repos) == 0 {
		return nil
	}
	return &funcOption{
		f: func(t *Transport) error {
			refOwner := t.owner
			invalid := make([]string, 0, len(repos))
			names := make([]string, 0, len(repos))
			for _, item := range repos {
				item = strings.ToLower(item)
				username, repo, ok := strings.Cut(item, "/")
				// Repository is in form username/repo.
				if ok {
					if !userNameRegExp.MatchString(username) {
						invalid = append(invalid, item)
						continue
					}

					// If refOwner is not set, set it first.
					if refOwner == "" {
						refOwner = username
					}

					// Repositories must be under a single installation.
					if username != refOwner {
						return fmt.Errorf("repositories from multiple owners specified: %v", repos)
					}

					// Assign repo to item if repo is in format username/repo.
					item = repo
				}

				// Ensure repository name is valid.
				if !repoNameRegExp.MatchString(item) {
					invalid = append(invalid, item)
				} else {
					t.repos = append(t.repos, item)
				}
			}

			if len(invalid) > 0 {
				return fmt.Errorf("invalid repositories specified: %v", invalid)
			}
			t.repos = append(t.repos, names...)

			// Sort before removing duplicates.
			slices.Sort(t.repos)

			// Remove duplicates
			t.repos = slices.Clip(slices.Compact(t.repos))

			// Set owner if not set.
			if t.owner == "" && refOwner != "" {
				t.owner = refOwner
			}
			return nil
		},
	}
}

// WithOwner configures installation owner to use.
func WithOwner(username string) Option {
	return &funcOption{
		f: func(t *Transport) error {
			username = strings.ToLower(username)
			if !userNameRegExp.MatchString(username) {
				return fmt.Errorf("invalid username: %s", username)
			}

			// If owner was already set, it might have been extracted from repos.
			// ensure they do not conflict.
			if t.owner != "" && t.owner != username {
				return fmt.Errorf("owner is already configured(%s): %s", t.owner, username)
			}

			t.owner = username
			return nil
		},
	}
}

// WithInstallationID configures [Transport] to use installation id specified.
//
// This is useful if it is required to access all repositories available for an
// installation without specifying them individually or if building [Transport]
// from data provided by [WebHook].
func WithInstallationID(id uint64) Option {
	return &funcOption{
		f: func(t *Transport) error {
			if id == 0 {
				return fmt.Errorf("installation id cannot be zero")
			}

			// If installation id is already set, ensure they do not conflict.
			if t.installID != 0 && t.installID != id {
				return fmt.Errorf("installation id is already configured(%d): %d", t.installID, id)
			}

			t.installID = id
			return nil
		},
	}
}

// WithPermissions configures permission scopes. This is useful when app has
// broader set of permissions a scoped access token is required.
//
// Permissions MUST be specified in <scope>:<access> or  <scope>=<access> format.
// Where scope is permission scope like "issues" and access can be one of
// "read", "write" or "admin".
//
// For example to request permissions to write issues and pull request can be specified as,
//
//	githubapp.WithPermissions("issues:write", "pull_requests:write")
func WithPermissions(permissions ...string) Option {
	if len(permissions) == 0 {
		return nil
	}
	return &funcOption{
		f: func(t *Transport) error {
			m := make(map[string]string, len(permissions))
			invalid := make([]string, 0, len(permissions))
			for _, item := range permissions {
				item = strings.ToLower(item)
				if permissionRegEx.MatchString(item) {
					// Replace = with :
					item = strings.ReplaceAll(item, "=", ":")

					// Ignore error checks as regex already validates
					// that permissions are in format <scope>:<level> format.
					scope, level, _ := strings.Cut(item, ":")
					m[scope] = level
				} else {
					invalid = append(invalid, item)
				}
			}
			if len(invalid) != 0 {
				return fmt.Errorf("invalid permissions: %v", invalid)
			}
			t.scopes = m
			return nil
		},
	}
}
