// SPDX-FileCopyrightText: Copyright 2023 Prasad Tengse
// SPDX-License-Identifier: MIT

package githubapp_test

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/tprasadtp/go-githubapp"
	"github.com/tprasadtp/go-githubapp/internal/api"
	"github.com/tprasadtp/go-githubapp/internal/testkeys"
	"github.com/tprasadtp/go-githubapp/internal/testutils"
)

// This tests makes live API calls to default GitHub api endpoint.
func TestIntegration(t *testing.T) {
	if testing.Short() {
		t.Skipf("Skip => Integration tests")
	}

	// Try to read private key from env variable or file defined in env variable.
	var appKeyEnv string
	if keyFile := os.Getenv("GO_GITHUBAPP_TEST_APP_PRIVATE_KEY_FILE"); keyFile != "" {
		t.Logf("Reading private key from - %s", keyFile)
		buf, err := os.ReadFile(keyFile)
		if err != nil {
			t.Fatalf("Failed to open private key (%s): %s", keyFile, err)
		}
		appKeyEnv = string(buf)
	} else {
		t.Logf("Reading private key from GO_GITHUBAPP_TEST_APP_PRIVATE_KEY")
		appKeyEnv = os.Getenv("GO_GITHUBAPP_TEST_APP_PRIVATE_KEY")
	}

	if appKeyEnv == "" {
		t.Skipf("Skip => GO_GITHUBAPP_TEST_APP_PRIVATE_KEY is not defined")
	}

	// Verify GO_GITHUBAPP_TEST_APP_PRIVATE_KEY is PEM encoded and is valid.
	block, _ := pem.Decode([]byte(appKeyEnv))
	if block == nil {
		t.Fatalf("GO_GITHUBAPP_TEST_APP_PRIVATE_KEY is not PEM encoded")
	}

	signer, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		t.Fatalf("GO_GITHUBAPP_TEST_APP_PRIVATE_KEY is not PKCS1 encoded: %s", err)
	}

	appIDEnv := os.Getenv("GO_GITHUBAPP_TEST_APP_ID")
	ghOwnerEnv := os.Getenv("GO_GITHUBAPP_TEST_OWNER")

	// Verify GO_GITHUBAPP_TEST_APP_ID is set and is an integer.
	if appIDEnv == "" {
		t.Skipf("Skip => GO_GITHUBAPP_TEST_APP_ID is not defined")
	}
	appID, err := strconv.ParseUint(appIDEnv, 10, 64)
	if err != nil {
		t.Fatalf("GO_GITHUBAPP_TEST_APP_ID is invalid: %s", err)
	}

	// Verify GO_GITHUBAPP_TEST_APP_ID is set.
	if ghOwnerEnv == "" {
		t.Skipf("Skip => GO_GITHUBAPP_TEST_OWNER is not defined")
	}

	// Check if GO_GITHUBAPP_TEST_API_URL is set.
	baseURLEnv := os.Getenv("GO_GITHUBAPP_TEST_API_URL")

	// Fallback to GH_HOST if GO_GITHUBAPP_TEST_API_URL is not set.
	if baseURLEnv == "" {
		baseURLEnv = os.Getenv("GH_HOST")
	}

	// If both GH_HOST and GO_GITHUBAPP_TEST_API_URL are unset,
	// use default endpoint.
	if baseURLEnv == "" {
		baseURLEnv = api.DefaultEndpoint
	}

	// Verify endpoint URL is valid.
	baseURL, err := url.Parse(baseURLEnv)
	if err != nil {
		t.Fatalf("Invalid REST API endpoint URL: %s", baseURLEnv)
	}

	// Checks if we can connect to GitHub api endpoint.
	t.Logf("Checking connectivity to REST API endpoint URL: %s", baseURLEnv)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, baseURL.String(), nil)
	if err != nil {
		t.Fatalf("Error building request: %s", err)
	}

	// Add User-Agent Header.
	req.Header.Add(api.UAHeader, api.UAHeaderValue)

	// Get token from env variable.
	//
	// API endpoint is usually rate limited. On CI/CD systems using a NAT gateway
	// or a proxy, this can lead to errors due to GitHub servers seeing the same IP.
	// To avoid it, authenticate the requests. This token need not have any permissions,
	// it just needs to be a valid token.
	var githubTokenEnv string

	// Try to lookup GO_GITHUBAPP_TEST_TOKEN
	githubTokenEnv = os.Getenv("GO_GITHUBAPP_TEST_TOKEN")

	// Try to lookup GITHUB_ENTERPRISE_TOKEN, if baseURL is not default.
	if githubTokenEnv == "" {
		if baseURL.Host != "api.github.com" {
			githubTokenEnv = os.Getenv("GITHUB_ENTERPRISE_TOKEN")
		}
	}

	// Fallback to GITHUB_TOKEN env variable.
	if githubTokenEnv == "" {
		githubTokenEnv = os.Getenv("GITHUB_TOKEN")
	}

	if githubTokenEnv != "" {
		t.Logf("Using provided token")
		req.Header.Add(api.AuthzHeader, fmt.Sprintf("Bearer: %s", githubTokenEnv))
	}

	client := http.Client{}

	baseURLResponse, err := client.Do(req)
	if err != nil {
		t.Skipf("Skip => REST API endpoint(%s) is not reachable: %s", baseURLEnv, err)
	}
	defer baseURLResponse.Body.Close()

	t.Logf("Successfully connected to REST API endpoint: %s", baseURLEnv)
	switch baseURLResponse.StatusCode {
	case http.StatusOK, http.StatusNoContent:
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable:
		t.Skipf("Skip => REST API endpoint(%s) returned server error: %s",
			baseURLEnv, baseURLResponse.Status)
	default:
		t.Fatalf("Invalid response from REST API endpoint(%s): %s",
			baseURLEnv, baseURLResponse.Status)
	}

	t.Run("InvalidAppPrivateKey", func(t *testing.T) {
		ctx, cancel := testutils.TestingContext(t, time.Minute)
		defer cancel()
		transport, err := githubapp.NewTransport(ctx, appID,
			testkeys.RSA2048(), githubapp.WithEndpoint(baseURLEnv))
		if err == nil {
			t.Errorf("expected an error, got nil")
		}

		if transport != nil {
			t.Errorf("expected NewTransport to return nil on invalid keys")
		}
	})

	t.Run("InvalidInstallation", func(t *testing.T) {
		ctx, cancel := testutils.TestingContext(t, time.Minute)
		defer cancel()
		transport, err := githubapp.NewTransport(ctx, appID,
			testkeys.RSA2048(), githubapp.WithEndpoint(baseURLEnv),
			githubapp.WithRepositories("tprasadtp/go-githubapp"),
		)
		if err == nil {
			t.Errorf("expected an error, got nil")
		}

		if transport != nil {
			t.Errorf("expected NewTransport to return nil on not installed repository")
		}
	})

	// Verify JWT returns valid app.
	t.Run("VerifyJWT", func(t *testing.T) {
		ctx, cancel := testutils.TestingContext(t, time.Minute)
		defer cancel()

		transport, err := githubapp.NewTransport(ctx, appID, signer, githubapp.WithEndpoint(baseURLEnv))
		if err != nil {
			t.Fatalf("Failed to build transport: %s", err)
		}

		client := &http.Client{
			Transport: transport,
		}

		// Check if GET /app works
		requestURL := baseURL.JoinPath("app")
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
		if err != nil {
			t.Fatalf("GET %s: Failed to build request: %s", request.URL, err)
		}

		t.Logf("%s %s: make request", request.Method, request.URL)
		response, err := client.Do(request)
		if err != nil {
			t.Fatalf("%s %s: request error: %s", request.Method, request.URL, err)
		}
		defer response.Body.Close()
		t.Logf("%s %s: response code: %s", request.Method, request.URL, response.Status)

		if response.StatusCode != http.StatusOK {
			t.Errorf("GET %s: invalid status code: %s", request.URL, response.Status)
		}

		t.Logf("%s %s: read response body", request.Method, request.URL)
		body, err := io.ReadAll(response.Body)
		if err != nil {
			t.Errorf("GET %s: failed to read response body", request.URL)
		}

		// Check if returned JSON is parsable.
		app := &api.App{}
		err = json.Unmarshal(body, app)
		if err != nil {
			t.Errorf("%s %s: invalid JSON: %s", request.Method, request.URL, err)
		}

		if app.ID == nil {
			t.Fatalf("%s %s: ID not populated, %#v", request.Method, request.URL, app)
		}

		if *app.ID != int64(appID) {
			t.Errorf("%s %s: expected App-ID %d, got %d", request.Method, request.URL, appID, *app.ID)
		}

		// Transport Attributes
		if transport.AppID() != appID {
			t.Errorf("expected app id=%d, got=%d", appID, transport.AppID())
		}

		if transport.AppName() == "" {
			t.Errorf("AppName not populated")
		}
	})

	// App has contents:read and issues:read permission
	// limit to contents:read only.
	t.Run("ScopedPermissions", func(t *testing.T) {
		ctx, cancel := testutils.TestingContext(t, time.Minute)
		defer cancel()

		transport, err := githubapp.NewTransport(
			ctx, appID, signer,
			githubapp.WithEndpoint(baseURLEnv),
			githubapp.WithOwner(ghOwnerEnv),
			githubapp.WithPermissions("contents:read"),
		)
		if err != nil {
			t.Fatalf("Failed to build transport: %s", err)
		}

		client := &http.Client{
			Transport: transport,
		}

		// Try to get issues.
		requestURL := baseURL.JoinPath("repos", ghOwnerEnv, "go-githubapp-repo-one", "issues")
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
		if err != nil {
			t.Fatalf("GET %s: Failed to build request: %s", request.URL, err)
		}

		t.Logf("%s %s: make request", request.Method, request.URL)
		response, err := client.Do(request)
		if err != nil {
			t.Fatalf("%s %s: request error: %s", request.Method, request.URL, err)
		}
		response.Body.Close()

		if response.StatusCode == http.StatusOK {
			t.Errorf("GET %s: expected non 200 response: %s", request.URL, response.Status)
		} else {
			t.Logf("%s %s: %s", request.Method, request.URL, response.Status)
		}

		// Try to get README
		requestURL = baseURL.JoinPath("repos", ghOwnerEnv, "go-githubapp-repo-one", "readme")
		request, err = http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
		if err != nil {
			t.Fatalf("GET %s: Failed to build request: %s", request.URL, err)
		}

		t.Logf("%s %s: make request", request.Method, request.URL)
		response, err = client.Do(request)
		if err != nil {
			t.Fatalf("%s %s: request error: %s", request.Method, request.URL, err)
		}
		response.Body.Close()

		if response.StatusCode != http.StatusOK {
			t.Errorf("GET %s: expected 200 response: %s", request.URL, response.Status)
		} else {
			t.Logf("%s %s: %s", request.Method, request.URL, response.Status)
		}
	})

	t.Run("RepositoryNotAccessible", func(t *testing.T) {
		ctx, cancel := testutils.TestingContext(t, time.Minute)
		defer cancel()

		transport, err := githubapp.NewTransport(
			ctx, appID, signer,
			githubapp.WithEndpoint(baseURLEnv),
			githubapp.WithOwner(ghOwnerEnv),
			// This installation should not have access to this repo.
			// But This repository MUST exist.
			githubapp.WithRepositories("go-githubapp-repo-no-access"),
		)

		if transport != nil {
			t.Errorf("NewTransport must return nil Transport with in accessible repositories")
		}

		if err == nil {
			t.Fatalf("NewTransport must return non-nil error with in accessible repositories")
		}

		if !strings.Contains(err.Error(), "422") {
			t.Errorf("error string should contain \"422\" error code")
		}
	})

	t.Run("TransportAttributes", func(t *testing.T) {
		ctx, cancel := testutils.TestingContext(t, time.Minute)
		defer cancel()

		transport, err := githubapp.NewTransport(
			ctx, appID, signer,
			githubapp.WithEndpoint(baseURLEnv),
			githubapp.WithOwner(ghOwnerEnv),
		)

		if err != nil {
			t.Fatalf("Failed to build transport: %s", err)
		}

		if transport.BotUsername() == "" {
			t.Errorf("BotUsername() returns empty")
		}

		if transport.BotCommitterEmail() == "" {
			t.Errorf("BotCommitterEmail() returns empty")
		}

		if transport.AppID() != appID {
			t.Errorf("Expected app id=%d, got=%d", appID, transport.AppID())
		}

		if transport.AppName() == "" {
			t.Errorf("Expected app id to be populated, but got empty")
		}
	})

	t.Run("ScopedRepositories", func(t *testing.T) {
		ctx, cancel := testutils.TestingContext(t, time.Minute)
		defer cancel()

		transport, err := githubapp.NewTransport(
			ctx, appID, signer,
			githubapp.WithEndpoint(baseURLEnv),
			githubapp.WithOwner(ghOwnerEnv),
			githubapp.WithRepositories("go-githubapp-repo-one"),
		)
		if err != nil {
			t.Fatalf("Failed to build transport: %s", err)
		}

		client := &http.Client{
			Transport: transport,
		}

		// Try to get readme for go-githubapp-repo-one.
		requestURL := baseURL.JoinPath("repos", ghOwnerEnv, "go-githubapp-repo-one", "readme")
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
		if err != nil {
			t.Fatalf("GET %s: Failed to build request: %s", request.URL, err)
		}

		t.Logf("%s %s: make request", request.Method, request.URL)
		response, err := client.Do(request)
		if err != nil {
			t.Fatalf("%s %s: %s", request.Method, request.URL, err)
		}
		defer response.Body.Close()

		if response.StatusCode != http.StatusOK {
			t.Errorf("GET %s: expected 200 response: %s", request.URL, response.Status)
		} else {
			t.Logf("%s %s: %s", request.Method, request.URL, response.Status)
		}

		// Try to get readme for go-githubapp-repo-two.
		requestURL = baseURL.JoinPath("repos", ghOwnerEnv, "go-githubapp-repo-two", "readme")
		request, err = http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
		if err != nil {
			t.Fatalf("GET %s: Failed to build request: %s", request.URL, err)
		}

		t.Logf("%s %s: make request", request.Method, request.URL)
		response, err = client.Do(request)
		if err != nil {
			t.Fatalf("%s %s: %s", request.Method, request.URL, err)
		}
		defer response.Body.Close()

		if response.StatusCode == http.StatusOK {
			t.Errorf("GET %s: expected non 200 response: %s", request.URL, response.Status)
		} else {
			t.Logf("%s %s: %s", request.Method, request.URL, response.Status)
		}
	})

	t.Run("VerifyWithOwner", func(t *testing.T) {
		ctx, cancel := testutils.TestingContext(t, time.Minute)
		defer cancel()

		transport, err := githubapp.NewTransport(
			ctx, appID, signer,
			githubapp.WithEndpoint(baseURLEnv),
			githubapp.WithOwner(ghOwnerEnv),
		)
		if err != nil {
			t.Errorf("Failed to build transport: %s", err)
		}

		client := &http.Client{
			Transport: transport,
		}

		for _, repo := range [...]string{"go-githubapp-repo-one", "go-githubapp-repo-two"} {
			requestURL := baseURL.JoinPath("repos", ghOwnerEnv, repo, "readme")
			request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
			if err != nil {
				t.Fatalf("GET %s: Failed to build request: %s", request.URL, err)
			}

			t.Logf("%s %s: make request", request.Method, request.URL)
			response, err := client.Do(request)
			if err != nil {
				t.Fatalf("%s %s: request error: %s", request.Method, request.URL, err)
			}
			t.Logf("%s %s: response code: %s", request.Method, request.URL, response.Status)

			if response.StatusCode != http.StatusOK {
				t.Errorf("GET %s: expected 200 response: %s", request.URL, response.Status)
			}

			if response.Body != nil {
				response.Body.Close()
			}
		}
	})
}
