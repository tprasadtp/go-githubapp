// SPDX-FileCopyrightText: Copyright 2023 Prasad Tengse
// SPDX-License-Identifier: MIT

package githubapp

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/tprasadtp/go-githubapp/internal/api"
)

var (
	_ http.RoundTripper = (*Transport)(nil)
)

// keyJWT is context key to indicate round tripper needs to to use jwt
// instead of installation token.
type keyJWT struct{}

// ctxWithJWTKey adds jwtKey to context to indicate round tripper should use JWT.
// This is required because refreshing [InstallationToken] or fetching app metadata
// requires using JWT.
func ctxWithJWTKey(ctx context.Context) context.Context {
	return context.WithValue(ctx, keyJWT{}, struct{}{})
}

// ctxHasKeyJWT checks if context has key keyJWT. This is used to re-use the
// same transport for token renewals.
func ctxHasKeyJWT(ctx context.Context) bool {
	return ctx.Value(keyJWT{}) != nil
}

// Transport provides a [http.RoundTripper] by wrapping an existing
// http.RoundTripper and provides GitHub Apps authenticating as a
// GitHub App or as an GitHub app installation.
//
// 'Authorization' header is automatically populated with suitable installation
// token or JWT token for all requests. If it already exists it is ignored.
// Token renewal requests will always override 'Accept' and "X-GitHub-Api-Version" headers.
type Transport struct {
	appID       uint64            // app ID
	appSlug     string            // app slug/name
	installID   uint64            // installation id
	owner       string            // owner of repositories
	repos       []string          // repository names
	next        http.RoundTripper // next round tripper
	baseURL     *url.URL          // REST API v3 base URL
	minter      jwtMinter         // jwt minter
	bearer      atomic.Value      // bearer token
	token       atomic.Value      // installation token
	tokenURL    string            // token url to create installation token
	botUsername string            // bot user.name
	botEmail    string            // bot user.email
	scopes      map[string]string // scoped permissions
}

// NewTransport creates a new [Transport] for authenticating as an app/installation.
//
// How [Transport] authenticates depends on installation options specified.
//
//   - If no installation options are specified, then [Transport] can only authenticate
//     as app (using JWT). This is not something you want typically, as very limited number
//     of actions like accessing available installations.
//   - Use [WithInstallationID] to have access to all permissions available to the
//     installation including organization scopes and repositories. This can be used
//     together with [WithPermissions] to limit scope of access tokens. Typical
//     example would be to close all stale issues for all repositories in an organization.
//     This task does not require access to code, thus "issues:write" permission should
//     be sufficient.
//   - Use [WithOwner] if your app has only access to organization/user permissions
//     and none of the repositories of the owner. Typical example would be an
//     app which manages self hosted runners in an organization or manages organization
//     level projects.
//   - Use [WithRepositories] if your app intends to access only a set of repositories.
//     Do note that, if app has access to organization permissions they will also be
//     available to the access token, unless limited with [WithPermissions].
//   - [WithPermissions] can be used to limit the scope of permissions available
//     to the access token.
//
// Access token and JWT are automatically refreshed whenever required.
//
// If only installation access token or JWT is required but not the round tripper,
// use [NewInstallationToken] or [NewJWT] respectively.
func NewTransport(ctx context.Context, appid uint64, signer crypto.Signer, opts ...Option) (*Transport, error) {
	var err error
	if signer == nil {
		err = errors.Join(err, errors.New("no signer provided"))
	}

	if appid == 0 {
		err = errors.Join(err, errors.New("app id cannot be zero"))
	}

	if err != nil {
		return nil, fmt.Errorf("githubapp: invalid options: %w", err)
	}

	// Apply all options.
	t := &Transport{
		appID: appid,
	}

	for i := range opts {
		if opts[i] != nil {
			err = errors.Join(err, opts[i].apply(t))
		}
	}

	// If only repository names are given, but not the owner.
	if len(t.repos) > 0 && t.owner == "" {
		err = errors.Join(err, errors.New("owner not specified"))
	}

	if err != nil {
		return nil, fmt.Errorf("githubapp: invalid options: %w", err)
	}

	// If there is no existing round tripper, use DefaultTransport.
	if t.next == nil {
		t.next = http.DefaultTransport
	}

	// If endpoint is not configured, use default endpoint.
	if t.baseURL == nil {
		t.baseURL, _ = url.Parse(defaultEndpoint)
	}

	// If context is nil, assign a default context.
	if ctx == nil {
		ctx = context.Background()
	}

	// Select JWT signer based on public key of the signer.
	switch v := signer.Public().(type) {
	case *rsa.PublicKey:
		if v.N.BitLen() < 2048 {
			return nil,
				fmt.Errorf("githubapp: rsa keys size(%d) < 2048 bits", v.N.BitLen())
		}
		t.minter = &jwtRS256{internal: signer}
	case *ecdsa.PublicKey:
		return nil, fmt.Errorf("githubapp: ECDSA keys are not supported")
	case *ed25519.PublicKey, ed25519.PublicKey:
		return nil, fmt.Errorf("githubapp: ED-25519 keys are not supported")
	default:
		return nil, fmt.Errorf("githubapp: unknown key type: %T", v)
	}

	// Shared client for init operations.
	client := &http.Client{
		Transport: t,
	}

	// Verify app id and signer are both valid.
	err = t.checkApp(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("githubapp: failed to verify app: %w", err)
	}

	// t.owner is only populated if WithOrganization or WithRepositories
	// is provided as an option. t.install is only populated if installation
	// id is specified.
	if t.owner != "" || t.installID != 0 {
		// Check installation.
		err = t.checkInstallation(ctx, client)
		if err != nil {
			return nil, fmt.Errorf("githubapp: failed to verify installation: %w", err)
		}

		// Fetch bot user metadata.
		err = t.fetchBotUserID(ctx, client)
		if err != nil {
			return nil, fmt.Errorf("githubapp: failed to fetch bot user metadata: %w", err)
		}
	}

	return t, nil
}

// AppID returns the github app id.
func (t *Transport) AppID() uint64 {
	return t.appID
}

// AppName returns the github app slug.
func (t *Transport) AppName() string {
	return t.appSlug
}

// BotUsername returns the github app's username.
// This is same as AppName, but with [bot] suffix.
func (t *Transport) BotUsername() string {
	return t.botUsername
}

// BotCommitterEmail returns the github app's no-reply email to use for git metadata.
func (t *Transport) BotCommitterEmail() string {
	return t.botEmail
}

// InstallationID returns the github installation id. If not repositories
// or organizations are configured, This will returns 0.
func (t *Transport) InstallationID() uint64 {
	return t.installID
}

// ScopedPermissions returns permissions configured for the transport.
// This is not the same as app permissions. This will return nil, if
// no scoped permissions are set.
func (t *Transport) ScopedPermissions() map[string]string {
	return maps.Clone(t.scopes)
}

// checkApp verifies app id and signer both are valid. This also populates app's name.
func (t *Transport) checkApp(ctx context.Context, client *http.Client) error {
	u := t.baseURL.JoinPath("app")

	// Set context to use JWT.
	r, _ := http.NewRequestWithContext(ctxWithJWTKey(ctx), http.MethodGet, u.String(), nil)

	// Verify the key is valid by making a request to /app.
	// See - https://docs.github.com/en/rest/apps/apps?apiVersion=2022-11-28
	resp, err := client.Do(r)
	if err != nil {
		return fmt.Errorf("failed to verify key for app id %d: %w", t.appID, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusForbidden, http.StatusUnauthorized:
		return fmt.Errorf("invalid app id or credentials: %s", resp.Status)
	default:
		return fmt.Errorf("failed to verify key for app id %d - %s", t.appID, resp.Status)
	}

	// Populate app's slug.
	appResp := api.App{}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	err = json.Unmarshal(data, &appResp)
	if err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	t.appSlug = *appResp.Slug
	return nil
}

// checkInstallation gets installation for a repo/org and verify permissions on the
// installation matches installation (app permissions can be updated independent of)
// installation. Also checks installation has access to all repositories configured.
//
// https://docs.github.com/en/rest/apps/apps?apiVersion=2022-11-28#get-a-repository-installation-for-the-authenticated-app--parameters
func (t *Transport) checkInstallation(ctx context.Context, client *http.Client) error {
	var u *url.URL
	if t.installID != 0 {
		u = t.baseURL.JoinPath("app", "installations", strconv.FormatUint(t.installID, 10))
	} else {
		u = t.baseURL.JoinPath("users", t.owner, "installation")
	}

	// Set context to use JWT.
	r, _ := http.NewRequestWithContext(ctxWithJWTKey(ctx), http.MethodGet, u.String(), nil)
	resp, err := client.Do(r)
	if err != nil {
		return fmt.Errorf("error fetching installation for %s: %w", t.owner, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error fetching installation id: %s", resp.Status)
	}

	getInstallationResp := api.Installation{}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	err = json.Unmarshal(data, &getInstallationResp)
	if err != nil {
		return fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	// Check if installation is suspended.
	if getInstallationResp.SuspendedAt != nil {
		if getInstallationResp.SuspendedAt.Time.Before(time.Now()) {
			return fmt.Errorf("installation id %d is not active", *getInstallationResp.ID)
		}
	}

	// Check is scoped permissions are supported by the app's installation.
	// permissions on app itself are not checked as effective permissions depend
	// on those granted by installation and scopes defined.
	err = t.checkInstallationPermissions(getInstallationResp.Permissions)
	if err != nil {
		return err
	}

	// Save installation ID.
	if t.installID == 0 {
		t.installID = uint64(*getInstallationResp.ID)
	} else if t.installID != 0 && t.installID != uint64(*getInstallationResp.ID) {
		return fmt.Errorf("configured installation id %d, does not match actual value %d",
			t.installID, *getInstallationResp.ID)
	}

	// Save access token url for the installation.
	if getInstallationResp.AccessTokensURL != nil {
		t.tokenURL = *getInstallationResp.AccessTokensURL
	}

	// Save owner if not specified. This is the case where only installation id is given.
	if t.owner == "" {
		t.owner = *getInstallationResp.Account.Login
	}

	return nil
}

// fetchBotUserID fetches bot's github user id.
func (t *Transport) fetchBotUserID(ctx context.Context, client *http.Client) error {
	u := t.baseURL.JoinPath("users", fmt.Sprintf("%s[bot]", t.appSlug))
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}

	resp, err := client.Do(r)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("invalid HTTP status: %s", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	user := api.User{}
	err = json.Unmarshal(data, &user)
	if err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if user.ID == nil || user.Login == nil {
		return fmt.Errorf("missing user id or login in API response")
	}

	t.botUsername = *user.Login
	t.botEmail = fmt.Sprintf("%d+%s@users.noreply.github.com", *user.ID, *user.Login)
	return nil
}

// checkInstallationPermissions checks if installation permissions support scoped permissions.
//
// This is a separate method to make unit testing easier. Do not fold it into checkInstallation.
func (t *Transport) checkInstallationPermissions(permissions map[string]string) error {
	// No scoped permissions are specified, app's default permissions apply.
	// no additional validation is required.
	if len(t.scopes) == 0 {
		return nil
	}

	missing := make([]string, 0, len(t.scopes))
	for scopeName, scopeLevel := range t.scopes {
		// Lookup if installation permission has that scope.
		installLevel, ok := permissions[scopeName]
		if !ok {
			missing = append(missing, scopeName)
			continue
		}

		// Installation permissions can be read/write/admin. So for scoped permissions,
		// if admin level is requested, installation permission must also be admin.
		// if write level is requested, installation permission on app can be write or admin.
		// if read level is requested installation permission can be either read, write or admin.
		switch scopeLevel {
		case "admin":
			if installLevel != "admin" {
				missing = append(missing, fmt.Sprintf("%s:%s",
					scopeName, scopeLevel))
			}
		case "write":
			switch installLevel {
			case "write", "admin":
			default:
				missing = append(missing, fmt.Sprintf("%s:%s", scopeName, scopeLevel))
			}
		case "read":
			switch installLevel {
			case "read", "write", "admin":
			default:
				missing = append(missing, fmt.Sprintf("%s:%s", scopeName, scopeLevel))
			}
		default:
			return fmt.Errorf("unknown %s level - %s", scopeName, scopeLevel)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing requested permissions: %v", missing)
	}
	return nil
}

// JWT returns already existing JWT bearer token or mints a new one.
func (t *Transport) JWT(ctx context.Context) (JWT, error) {
	v := t.bearer.Load()
	if v != nil {
		if bearer, _ := v.(JWT); bearer.IsValid() {
			return bearer, nil
		}
	}

	bearer, err := t.minter.Mint(ctx, t.appID, time.Now())
	if err != nil {
		return JWT{}, fmt.Errorf("githubapp: failed to mint JWT: %w", err)
	}

	// Sign returns BearerToken without app slug, add it.
	bearer.AppName = t.appSlug
	t.bearer.Store(bearer)
	return bearer, nil
}

// InstallationToken returns a new installation access token. This, always returns
// a new token, thus callers can safely revoke the token whenever required.
func (t *Transport) InstallationToken(ctx context.Context) (InstallationToken, error) {
	if t.installID == 0 {
		return InstallationToken{}, fmt.Errorf("githubapp: installation id is not configured")
	}

	buf, err := json.Marshal(api.InstallationTokenRequest{
		Repositories: t.repos,
		Permissions:  t.scopes,
	})
	if err != nil {
		return InstallationToken{},
			fmt.Errorf("githubapp: failed to marshal token request data: %w", err)
	}

	r, err := http.NewRequestWithContext(
		ctxWithJWTKey(ctx), http.MethodPost, t.tokenURL, bytes.NewBuffer(buf))
	if err != nil {
		return InstallationToken{},
			fmt.Errorf("githubapp: failed to build token request: %w", err)
	}

	client := http.Client{
		Transport: t,
	}

	resp, err := client.Do(r)
	if err != nil {
		return InstallationToken{},
			fmt.Errorf("githubapp: failed to get installation token: %w", err)
	}
	defer resp.Body.Close()

	// Check response code.
	// https://docs.github.com/en/rest/apps/apps?apiVersion=2022-11-28#create-an-installation-access-token-for-an-app--status-codes
	switch resp.StatusCode {
	case http.StatusCreated:
	case http.StatusForbidden:
		return InstallationToken{},
			fmt.Errorf("githubapp: private key is invalid: %s", resp.Status)
	case http.StatusNotFound:
		return InstallationToken{},
			fmt.Errorf("githubapp: installation not found: %s", resp.Status)
	case http.StatusUnprocessableEntity:
		return InstallationToken{},
			fmt.Errorf("githubapp: validation error/too many token requests: %s", resp.Status)
	default:
		return InstallationToken{},
			fmt.Errorf("githubapp: API returned error: %s", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return InstallationToken{},
			fmt.Errorf("githubapp: failed to read installation token response: %w", err)
	}

	tokenResp := api.InstallationTokenResponse{}
	err = json.Unmarshal(data, &tokenResp)
	if err != nil {
		return InstallationToken{},
			fmt.Errorf("githubapp: failed to unmarshal installation token response: %w", err)
	}

	// InstallationToken
	token := InstallationToken{
		Server:         t.baseURL.String(),
		AppID:          t.appID,
		AppName:        t.appSlug,
		InstallationID: t.installID,
		Token:          tokenResp.Token,
		Exp:            tokenResp.Exp.Time,
		Owner:          t.owner,
	}

	if tokenResp.Repositories != nil {
		token.Repositories = make([]string, 0, len(tokenResp.Repositories))
		for _, item := range tokenResp.Repositories {
			if item != nil {
				token.Repositories = append(token.Repositories, *item.Name)
			}
		}
	}

	token.BotCommitterEmail = t.botEmail
	token.BotUsername = t.botUsername
	if tokenResp.Permissions != nil {
		token.Permissions = tokenResp.Permissions
	}

	return token, nil
}

// installationAuthzHeaderValue returns Authorization header value to be used
// for accessing API as installation. The token is automatically refreshed
// whenever required. This already includes prefix Bearer and can be directly
// used with [net/http.Header.Set]. If error occurs during creating a new token
// header string value is empty.
func (t *Transport) installationAuthzHeaderValue(ctx context.Context) (string, error) {
	v := t.token.Load()
	if v != nil {
		if token, _ := v.(InstallationToken); token.IsValid() {
			return "Bearer " + token.Token, nil
		}
	}
	token, err := t.InstallationToken(ctx)
	if err != nil {
		return "", err
	}
	return "Bearer " + token.Token, nil
}

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	clone := cloneRequest(req) // RoundTripper should not modify request

	// ctxHasKeyJWT is only set for token renewals, ignore 'Accept' and
	// 'X-GitHub-Api-Version' headers if any and always use library defaults.
	if ctxHasKeyJWT(ctx) {
		clone.Header.Set(acceptHeader, acceptHeaderValue)
		clone.Header.Set(apiVersionHeader, apiVersionHeaderValue)
	}

	// installation id is populated when WithRepositories or WithOrganization
	// or WithInstallationID etc are used. ctxHasKeyJWT returns true when context
	// value is set. If any of these are true, transport will use JWT for authentication.
	// Otherwise uses installation access token for authentication.
	if t.installID == 0 || ctxHasKeyJWT(ctx) {
		jwt, err := t.JWT(ctx)
		if err != nil {
			return nil, err
		}
		clone.Header.Set(authzHeader, "Bearer "+jwt.Token)
	} else {
		authzHeaderValue, err := t.installationAuthzHeaderValue(ctx)
		if err != nil {
			return nil, err
		}
		clone.Header.Set(authzHeader, authzHeaderValue)
	}

	//nolint:wrapcheck // don't wrap errors returned by underlying round-tripper.
	return t.next.RoundTrip(clone)
}

// cloneRequest returns a clone of the provided *http.Request.
// The clone is a shallow copy of the struct and its shallow copy of
// Header map.
func cloneRequest(r *http.Request) *http.Request {
	// shallow copy of the struct
	clone := new(http.Request)
	*clone = *r

	// shallow copy of the Headers.
	clone.Header = maps.Clone(r.Header)
	return clone
}
