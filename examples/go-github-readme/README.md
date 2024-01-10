# Example

An example program to get readme file for a repository using [google/go-github].

## Example

```go
// SPDX-FileCopyrightText: Copyright 2023 Prasad Tengse
// SPDX-License-Identifier: MIT

// This is an example CLI which fetches README for a repository using app credentials and
// [github.com/google/go-github/v55/github]. This is meant as an example and is not covered
// by semver compatibility guarantees.
package main

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/google/go-github/v55/github"
	"github.com/tprasadtp/go-githubapp"
)

var privFile string
var appID uint64
var slug string

func Usage() {
	fmt.Fprintf(flag.CommandLine.Output(), "Example using github.com/google/go-github/v55/github\n\n")
	fmt.Fprintf(flag.CommandLine.Output(), "Flags:\n")
	flag.PrintDefaults()
}

func main() {
	flag.StringVar(&privFile, "private-key", "", "Path to PKCS1 private key file (required)")
	flag.Uint64Var(&appID, "app-id", 0, "GitHub app ID (required)")
	flag.StringVar(&slug, "repo", "", "Repository in {owner}/{repository} format (required)")

	flag.Usage = Usage
	flag.Parse()

	if appID == 0 {
		log.Fatal("GitHub app ID not specified")
	}

	if privFile == "" {
		log.Fatal("Private key file not specified")
	}

	file, err := os.Open(privFile)
	if err != nil {
		log.Fatalf("Failed to open private key: %s", err)
	}

	slurp, err := io.ReadAll(file)
	if err != nil {
		log.Fatalf("Failed to read private key: %s", err)
	}

	block, _ := pem.Decode(slurp)
	if block == nil {
		log.Fatalf("Invalid private key: %s", err)
	}

	// Try to parse key as private key.
	signer, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		log.Fatalf("Invalid private key: %s", err)
	}

	username, repository, ok := strings.Cut(slug, "/")
	if !ok {
		log.Fatalf("Repository MUST specified be in {owner}/{repository} format")
	}

	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	transport, err := githubapp.NewTransport(
		ctx, appID, signer,
		githubapp.WithOwner(username),
		githubapp.WithRepositories(repository),
	)
	if err != nil {
		log.Fatalf("Failed to build round tripper: %s", err)
	}

	// Build a new client
	client := github.NewClient(&http.Client{Transport: transport})

	// Use client
	readme, _, err := client.Repositories.GetReadme(ctx, username, repository, nil)
	if err != nil {
		log.Fatalf("Failed to get README: %s", err)
	}

	content, err := readme.GetContent()
	if err != nil {
		log.Fatalf("Failed to get README: %s", err)
		return
	}

	fmt.Printf("%s/%s README:\n\n%s\n", username, repository, content)
}
```

## Example Usage

To obtain README from a private repository accessible to installation,

```
go mod tidy
go run example.go \
    -app-id <app-id> \
    -private-key private-key.pem \
    -repo {owner}/{repo}
```

Should return something like,

```
gh-integration-tests/go-githubapp-repo-one README:

# About This Repository

This repository is used for integration tests for [github.com/tprasadtp/go-githubapp].

[github.com/tprasadtp/go-githubapp]: https://github.com/tprasadtp/go-githubapp
```

[google/go-github]: github.com/google/go-github
