// SPDX-FileCopyrightText: Copyright 2023 Prasad Tengse
// SPDX-License-Identifier: MIT

// An example CLI which can fetch installation tokens for a github app
// and act like git credentials plugin.
package main

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/tprasadtp/go-githubapp"
)

var privFile string
var appID uint64
var installationID uint64
var repos string
var gitCredMode bool

//nolint:forbidigo // example script.
func main() {
	flag.StringVar(&privFile, "key", "", "private key")
	flag.Uint64Var(&appID, "app-id", 0, "app id")
	flag.Uint64Var(&installationID, "install-id", 0, "installation id")
	flag.StringVar(&repos, "repos", "", "repos")
	flag.BoolVar(&gitCredMode, "git-credentials", false, "git credentials mode")
	flag.Parse()

	if appID == 0 {
		log.Fatal("app id not specified")
	}

	if appID == 0 {
		log.Fatal("private key not specified")
	}

	file, err := os.Open(privFile)
	if err != nil {
		log.Fatal(err)
	}

	slurp, err := io.ReadAll(file)
	if err != nil {
		log.Fatal(err)
	}

	block, _ := pem.Decode(slurp)
	if block == nil {
		log.Fatal(err)
	}

	// Try to parse key as private key.
	signer, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		log.Fatal(err)
	}

	var opts []githubapp.Option
	if installationID != 0 {
		opts = append(opts, githubapp.WithInstallationID(installationID))
	}

	if repos != "" {
		repoList := strings.Split(repos, ",")
		opts = append(opts, githubapp.WithRepositories(repoList...))
	}

	ctx := context.Background()
	transport, err := githubapp.NewTransport(ctx, appID, signer,
		githubapp.Options(opts...),
	)
	if err != nil {
		log.Fatal(err)
	}

	token, err := transport.InstallationToken(ctx)
	if err != nil {
		log.Fatal(err)
	}

	if gitCredMode {
		fmt.Printf("protocol=https\n")
		fmt.Printf("username=x-access-token\n")
		fmt.Printf("password=%s\n", token.Token)
		fmt.Printf("password_expiry_utc=%d\n", token.Exp.Truncate(time.Second).Unix())
		fmt.Println()
	} else {
		fmt.Printf("Token: %s\n", token.Token)
		fmt.Printf("user.name: %s\n", token.BotUsername)
		fmt.Printf("user.email: %s\n", token.BotCommitterEmail)
	}
}
