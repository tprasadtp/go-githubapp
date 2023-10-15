// SPDX-FileCopyrightText: Copyright 2023 Prasad Tengse
// SPDX-License-Identifier: MIT

package githubapp_test

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/tprasadtp/go-githubapp"
	"github.com/tprasadtp/go-githubapp/internal"
	"github.com/tprasadtp/go-githubapp/internal/api"
)

// This tests makes live API calls to default github api endpoint.
//
//nolint:cyclop
func TestIntegration(t *testing.T) {
	if testing.Short() {
		t.Skipf("Skip => Integration tests in short mode")
	}

	appKeyEnv := os.Getenv("GO_GITHUBAPP_TEST_APP_PRIVATE_KEY")
	appIDEnv := os.Getenv("GO_GITHUBAPP_TEST_APP_ID")
	ghOwnerEnv := os.Getenv("GO_GITHUBAPP_TEST_OWNER")

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

	// Verify GO_GITHUBAPP_TEST_APP_ID is valid integer.
	if appIDEnv == "" {
		t.Skipf("Skip => GO_GITHUBAPP_TEST_APP_ID is not defined")
	}

	appID, err := strconv.ParseUint(appIDEnv, 10, 64)
	if err != nil {
		t.Fatalf("GO_GITHUBAPP_TEST_APP_ID is invalid: %s", err)
	}

	if ghOwnerEnv == "" {
		t.Skipf("Skip => GO_GITHUBAPP_TEST_OWNER is not defined")
	}

	baseURLEnv := os.Getenv("GO_GITHUBAPP_TEST_BASE_URL")
	if baseURLEnv == "" {
		baseURLEnv = internal.DefaultEndpoint
	}

	// Verify endpoint URL is valid.
	baseURL, err := url.Parse(baseURLEnv)
	if err != nil {
		t.Fatalf("Invalid GO_GITHUBAPP_TEST_BASE_URL: %s", baseURLEnv)
	}

	// Check if can connect to github api endpoint.
	t.Logf("Checking connectivity to GO_GITHUBAPP_TEST_BASE_URL: %s", baseURLEnv)
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, baseURL.String(), nil)
	if err != nil {
		t.Fatalf("Error building request: %s", err)
	}

	baseURLResponse, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Skipf("Skip => GO_GITHUBAPP_TEST_BASE_URL(%s) is not reachable: %s", baseURLEnv, err)
	}
	defer baseURLResponse.Body.Close()

	t.Logf("Successfully connected to GO_GITHUBAPP_TEST_BASE_URL: %s", baseURLEnv)
	switch baseURLResponse.StatusCode {
	case http.StatusOK, http.StatusNoContent:
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable:
		t.Skipf("Skip => GO_GITHUBAPP_TEST_BASE_URL(%s) returned server error: %s",
			baseURLEnv, baseURLResponse.Status)
	default:
		t.Fatalf("Invalid response from GO_GITHUBAPP_TEST_BASE_URL(%s): %s",
			baseURLEnv, baseURLResponse.Status)
	}

	ctx := context.Background()

	// Verify JWT returns valid app.
	t.Run("VerifyJWT", func(t *testing.T) {
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
	})

	// App has contents:read and issues:read permission
	// limit to contents:read only.
	t.Run("ScopedPermissions", func(t *testing.T) {
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

	t.Run("ScopedRepositories", func(t *testing.T) {
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
			defer response.Body.Close()
			t.Logf("%s %s: response code: %s", request.Method, request.URL, response.Status)

			if response.StatusCode != http.StatusOK {
				t.Errorf("GET %s: expected 200 response: %s", request.URL, response.Status)
			}
		}
	})
}
