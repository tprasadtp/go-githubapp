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

	"github.com/tprasadtp/go-githubapp/internal/api"
)

var (
	_ slog.LogValuer = (*InstallationToken)(nil)
)

// InstallationToken is an installation access token from GitHub.
type InstallationToken struct {
	// Installation access token. Typically starts with "ghs_".
	Token string `json:"token,omitempty" yaml:"token,omitempty"`

	// GitHub app ID.
	AppID uint64 `json:"app_id,omitempty" yaml:"appID,omitempty"`

	// GitHub app name.
	AppName string `json:"app_name,omitempty" yaml:"appName,omitempty"`

	// Installation ID for the app.
	InstallationID uint64 `json:"installation_id,omitempty" yaml:"installationID,omitempty"`

	// GitHub API endpoint. This is also used for token revocation.
	// If omitted, assume the default value of "https://api.githhub.com/".
	Server string `json:"server,omitempty" yaml:"server,omitempty"`

	// UserAgent used to fetch this installation access token.
	UserAgent string `json:"user_agent,omitempty" yaml:"user_agent,omitempty"`

	// Token exp time.
	Exp time.Time `json:"exp,omitempty" yaml:"exp,omitempty"`

	// Installation owner. This is owner of the installation.
	Owner string `json:"owner,omitempty" yaml:"owner,omitempty"`

	// Repositories which can be accessed with the token. This may be empty
	//  if a scoped token is not requested. In such cases, token will have access to all
	// repositories accessible by the installation.
	Repositories []string `json:"repositories,omitempty" yaml:"repositories,omitempty"`

	// Permissions available for the token. This may be omitted if scoped permissions are not
	// requested. In such cases token has all permissions available to the installation.
	Permissions map[string]string `json:"permissions,omitempty" yaml:"permissions,omitempty"`

	// BotUsername is app's github username.
	BotUsername string `json:"bot_username,omitempty" yaml:"bot_username,omitempty"`

	// BotCommitterEmail is committer email to use to attribute commits to the bot.
	// This is in the form "<user-id>+<app-name>[bot]@users.noreply.github.com".
	BotCommitterEmail string `json:"bot_committer_email,omitempty" yaml:"bot_committer_email,omitempty"`
}

// LogValue implements [log/slog.LogValuer].
func (t *InstallationToken) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("server", t.Server),
		slog.Uint64("app_id", t.AppID),
		slog.String("app_name", t.AppName),
		slog.String("user_agent", t.UserAgent),
		slog.Uint64("installation_id", t.InstallationID),
		slog.Any("repositories", t.Repositories),
		slog.String("token", "REDACTED"),
		slog.Time("exp", t.Exp),
		slog.Any("permissions", t.Permissions),
		slog.String("bot_username", t.BotUsername),
		slog.String("bot_committer_email", t.BotCommitterEmail),
	)
}

// IsValid checks if [InstallationToken] is valid for at-least 60 seconds.
func (t *InstallationToken) IsValid() bool {
	return t.Token != "" && (t.Exp.After(time.Now().Add(time.Minute)) || t.Exp.IsZero())
}

// Revoke revokes the installation access token.
func (t *InstallationToken) Revoke(ctx context.Context) error {
	return t.revoke(ctx, nil)
}

// revoke is an internal version of Revoke, which supports custom round tripper
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
		server = api.DefaultEndpoint
	}
	u, err := url.Parse(server)
	if err != nil {
		return fmt.Errorf("githubapp: failed to revoke token - invalid server url: %w", err)
	}
	u = u.JoinPath(u.Path, "installation", "token")

	switch u.Scheme {
	case "http", "https":
	default:
		return fmt.Errorf("invalid url scheme : %s (%s)", u.Scheme, server)
	}

	if u.Fragment != "" || u.RawQuery != "" {
		return fmt.Errorf("githubapp: failed to revoke token - server url cannot have fragments or queries: %s", server)
	}

	// NewRequestWithContext returns an error on invalid methods and nil context,
	// and invalid URL. All of which are non-reachable code-paths. But we check for
	// error anyway as it is an implementation detail.
	r, err := http.NewRequestWithContext(ctx, http.MethodDelete, u.String(), nil)
	if err != nil {
		return fmt.Errorf("githubapp: failed to revoke token - failed to build request: %w", err)
	}

	// Add Headers.
	r.Header.Set(api.VersionHeader, api.VersionHeaderValue)
	r.Header.Set(api.AuthzHeader, api.AuthzHeaderValue(t.Token))
	r.Header.Add(api.AcceptHeader, api.AcceptHeaderValue)
	if t.UserAgent == "" {
		r.Header.Add(api.UAHeader, api.UAHeaderValue)
	} else {
		r.Header.Add(api.UAHeader, t.UserAgent)
	}

	client := &http.Client{}

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
