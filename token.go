// SPDX-FileCopyrightText: Copyright 2023 Prasad Tengse
// SPDX-License-Identifier: MIT

package githubapp

import (
	"context"
	"crypto"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

var (
	_ slog.LogValuer = (*InstallationToken)(nil)
)

// InstallationToken is an installation access token from GitHub.
type InstallationToken struct {
	// Installation access token. Typically starts with "ghs_".
	Token string `json:"token" yaml:"token"`

	// GitHub API endpoint. This is also used for token revocation.
	Server string `json:"server,omitempty" yaml:"server,omitempty"`

	// GitHub app ID.
	AppID uint64 `json:"app_id,omitempty" yaml:"appID,omitempty"`

	// GitHub app name.
	AppName string `json:"app_name,omitempty" yaml:"appName,omitempty"`

	// Installation ID for the app.
	InstallationID uint64 `json:"installation_id,omitempty" yaml:"installationID,omitempty"`

	// Token exp time.
	Exp time.Time `json:"exp,omitempty" yaml:"exp,omitempty"`

	// Installation owner. This is owner of the installation.
	Owner string `json:"owner,omitempty" yaml:"owner,omitempty"`

	// Repositories which can be accessed with the token. This may be empty
	// if scoped token is not requested. In such cases, token will have access to all
	// repositories accessible by the installation.
	Repositories []string `json:"repositories,omitempty" yaml:"repositories,omitempty"`

	// Permissions available for the token. This may be omitted if scoped permissions are not
	// requested. In such cases token has all permissions available to the installation.
	Permissions map[string]string `json:"permissions,omitempty" yaml:"permissions,omitempty"`

	// BotUsername is app's github username.
	BotUsername string

	// BotCommitterEmail is committer email to use to attribute commits to the bot.
	BotCommitterEmail string
}

// LogValue implements [log/slog.LogValuer].
func (t *InstallationToken) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("server", t.Server),
		slog.Uint64("app_id", t.AppID),
		slog.String("app_name", t.AppName),
		slog.Uint64("installation_id", t.InstallationID),
		slog.Any("repositories", t.Repositories),
		slog.String("token", "REDACTED"),
		slog.Time("exp", t.Exp),
		slog.Any("permissions", t.Permissions),
	)
}

// Checks if [InstallationToken] is valid for at-least 60 seconds.
func (t *InstallationToken) IsValid() bool {
	return t.Token != "" && t.Exp.After(time.Now().Add(time.Minute))
}

// Revoke revokes the installation access token.
func (t *InstallationToken) Revoke(ctx context.Context) error {
	return t.revoke(ctx, nil)
}

// revoke is internal version of Revoke which supports custom round tripper
// for testing and customization.
func (t *InstallationToken) revoke(ctx context.Context, rt http.RoundTripper) error {
	if ctx == nil {
		ctx = context.Background()
	}

	if !t.IsValid() {
		return fmt.Errorf("githubapp: cannot revoke already invalid token")
	}

	server := t.Server
	if t.Server == "" {
		server = DefaultEndpoint
	}
	u, err := url.Parse(server)
	if err != nil {
		return fmt.Errorf("githubapp: failed to revoke token - invalid server url: %w", err)
	}

	// url.JoinPath only returns an error when parsing base path fails.
	// but always base path is u.Path which itself is returned by url.Parse.
	// Thus this error check is redundant, but as it is an implementation detail,
	// we check for errors anyways.
	u.Path, err = url.JoinPath(u.Path, "installation", "token")
	if err != nil {
		return fmt.Errorf("githubapp: failed to revoke token - invalid server url: %w", err)
	}

	switch u.Scheme {
	case "http", "https":
	default:
		return fmt.Errorf("invalid url scheme : %s (%s)", u.Scheme, server)
	}

	if u.Fragment != "" || u.RawQuery != "" {
		return fmt.Errorf("githubapp: failed to revoke token - server url cannot have fragments or queries: %s", server)
	}

	// NewRequestWithContext returns an error on invalid methods and nil context,
	// and invalid URL. All of which are non reachable code-paths. But we check for
	// error anyways as it is an implementation detail.
	r, err := http.NewRequestWithContext(ctx, http.MethodDelete, u.String(), nil)
	if err != nil {
		return fmt.Errorf("githubapp: failed to revoke token - failed to build request: %w", err)
	}

	// Add Headers.
	r.Header.Set(apiVersionHeader, apiVersionHeaderValue)
	r.Header.Set(authzHeader, t.Token)
	r.Header.Add(acceptHeader, acceptHeaderValue)
	r.Header.Add(uaHeader, uaHeaderValue)

	// Revoke token.
	client := http.Client{
		Timeout: time.Minute,
	}

	// Uses custom round tripper specified.
	if rt != nil {
		client.Transport = rt
	}

	resp, err := client.Do(r)
	if err != nil {
		return fmt.Errorf("githubapp: failed to revoke token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("githubapp: failed to revoke token, expected(204) but got %s", resp.Status)
	}

	// If successful indicate token is no longer valid.
	t.Exp = time.Now()

	return nil
}

// NewInstallationToken returns new installation access token.
// This takes same options as [Transport].
func NewInstallationToken(ctx context.Context, appid uint64, signer crypto.Signer, opts ...Option) (InstallationToken, error) {
	t, err := NewTransport(ctx, appid, signer, opts...)
	if err != nil {
		return InstallationToken{}, err
	}
	return t.InstallationToken(ctx)
}
