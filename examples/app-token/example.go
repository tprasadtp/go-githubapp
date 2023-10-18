// SPDX-FileCopyrightText: Copyright 2023 Prasad Tengse
// SPDX-License-Identifier: MIT

// An example CLI which can fetch installation tokens for a github app.
package main // import "github.com/tprasadtp/go-githubapp/examples/app-token"

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/tprasadtp/go-githubapp"
)

var privFile string
var appID uint64
var installationID uint64
var repos string
var owner string
var modeJwt bool

func Usage() {
	fmt.Fprintf(flag.CommandLine.Output(), "Tool to obtain installation access token or JWT for a Github App\n\n")
	fmt.Fprintf(flag.CommandLine.Output(), "This is a simple example CLI and is not covered by semver compatibility guarantees.\n")
	fmt.Fprintf(flag.CommandLine.Output(), "Use https://github.com/tprasadtp/gh-app-token if you need a CLI.\n\n")
	fmt.Fprintf(flag.CommandLine.Output(), "Usage: go run github.com/tprasadtp/go-githubapp/examples/app-token@latest\n\n")
	fmt.Fprintf(flag.CommandLine.Output(), "Flags:\n")
	flag.PrintDefaults()
}

func main() {
	flag.StringVar(&privFile, "private-key", "", "Path to PKCS1 private key file (required)")
	flag.Uint64Var(&appID, "app-id", 0, "GitHub app ID (required)")
	flag.Uint64Var(&installationID, "install-id", 0, "Installation ID")
	flag.StringVar(&repos, "repos", "", "Comma separated list of repositories")
	flag.StringVar(&owner, "owner", "", "Installation owner")
	flag.BoolVar(&modeJwt, "jwt", false, "Generate JWT")

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

	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	if modeJwt {
		token, err := githubapp.NewJWT(ctx, appID, signer)
		if err != nil {
			log.Fatalf("Failed to mint JWT: %s", err)
		}
		fmt.Printf("JWT           : %s\n", token.Token)
	} else {
		var opts []githubapp.Option
		if installationID != 0 {
			opts = append(opts, githubapp.WithInstallationID(installationID))
		}

		if repos != "" {
			list := strings.Split(repos, ",")
			opts = append(opts, githubapp.WithRepositories(list...))
		}

		if owner != "" {
			opts = append(opts, githubapp.WithOwner(owner))
		}

		token, err := githubapp.NewInstallationToken(ctx, appID, signer, opts...)
		if err != nil {
			log.Fatalf("error generating token: %s", err)
		}

		fmt.Printf("Token        : %s\n", token.Token)
		fmt.Printf("Owner        : %v\n", token.Owner)
		fmt.Printf("Installation : %d\n", token.InstallationID)
		fmt.Printf("Repositories : %v\n", token.Repositories)
		fmt.Printf("Permissions  : %v\n", token.Permissions)
		fmt.Printf("user.name    : %s\n", token.BotUsername)
		fmt.Printf("user.email   : %s\n", token.BotCommitterEmail)
	}
}
