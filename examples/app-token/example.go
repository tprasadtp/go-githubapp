// SPDX-FileCopyrightText: Copyright 2023 Prasad Tengse
// SPDX-License-Identifier: MIT

// An example CLI which can fetch installation tokens for a GitHub app.
package main // import "github.com/tprasadtp/go-githubapp/examples/app-token"

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/template"

	"github.com/tprasadtp/go-githubapp"
)

var privFile string
var app uint64
var installation uint64
var repos string
var owner string
var format string
var revoke bool

func Usage() {
	fmt.Fprintf(flag.CommandLine.Output(), "Tool to obtain installation access token or JWT for a Github App\n\n")
	fmt.Fprintf(flag.CommandLine.Output(), "This is a simple example CLI and is not covered by semver compatibility guarantees.\n")
	fmt.Fprintf(flag.CommandLine.Output(), "Use https://github.com/tprasadtp/gh-app-token if you need a CLI.\n\n")
	fmt.Fprintf(flag.CommandLine.Output(), "Usage: go run github.com/tprasadtp/go-githubapp/examples/app-token@latest\n\n")
	fmt.Fprintf(flag.CommandLine.Output(), "Flags:\n")
	flag.PrintDefaults()
}

func run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// check if running as revoke mode
	if revoke {
		if flag.NArg() < 1 {
			return fmt.Errorf("no tokens provided")
		}
		ec := 0
		for _, item := range flag.Args() {
			token := githubapp.InstallationToken{Token: item}
			err := token.Revoke(ctx)
			if err != nil {
				slog.Error("Failed to revoke token",
					"token", fmt.Sprintf("%s****", token.Token[0:8]),
					"err", err,
				)
				ec++
			} else {
				slog.Info("Token successfully revoked",
					"token", fmt.Sprintf("%s****", token.Token[0:8]),
				)
			}
		}
		if ec != 0 {
			return fmt.Errorf("failed to revoke %d tokens", ec)
		}
		return nil
	}

	if app == 0 {
		return fmt.Errorf("GitHub app ID not specified")
	}

	if privFile == "" {
		return fmt.Errorf("private key file not specified")
	}

	file, err := os.Open(privFile)
	if err != nil {
		return fmt.Errorf("failed to open private key: %w", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat private key file: %w", err)
	}
	if stat.Size() > 32e3 {
		return fmt.Errorf("private key file is too large: %d", stat.Size())
	}

	slurp, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read private key: %w", err)
	}

	block, _ := pem.Decode(slurp)
	if block == nil {
		return fmt.Errorf("invalid private key: %w", err)
	}

	// Try to parse key as a private key.
	signer, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("invalid private key: %w", err)
	}

	// Check if output template is valid.
	var tpl *template.Template
	if format != "" {
		tpl, err = template.New("format").Parse(format)
		if err != nil {
			return fmt.Errorf("invalid template: %w", err)
		}
	}

	// If no repos, owner and installation id are specified use JWT.
	if owner == "" && repos == "" && installation == 0 {
		token, err := githubapp.NewJWT(ctx, app, signer)
		if err != nil {
			return fmt.Errorf("failed to mint JWT: %w", err)
		}
		if tpl != nil {
			err = tpl.Execute(os.Stdout, token)
			if err != nil {
				return fmt.Errorf("failed to render template: %w", err)
			}
		} else {
			fmt.Printf("App ID            : %d\n", token.AppID)
			fmt.Printf("JWT               : %s\n", token.Token)
		}
		return nil
	}

	// One of repos/owner or installation id is specified.
	// Get installation access token,
	var opts []githubapp.Option
	if installation != 0 {
		opts = append(opts, githubapp.WithInstallationID(installation))
	}

	if repos != "" {
		list := strings.Split(repos, ",")
		opts = append(opts, githubapp.WithRepositories(list...))
	}

	if owner != "" {
		opts = append(opts, githubapp.WithOwner(owner))
	}

	token, err := githubapp.NewInstallationToken(ctx, app, signer, opts...)
	if err != nil {
		return fmt.Errorf("error generating token: %w", err)
	}

	if tpl != nil {
		err = tpl.Execute(os.Stdout, token)
		if err != nil {
			return fmt.Errorf("failed to render template: %w", err)
		}
	} else {
		fmt.Printf("App Name          : %s\n", token.AppName)
		fmt.Printf("App ID            : %d\n", token.AppID)
		fmt.Printf("Token             : %s\n", token.Token)
		fmt.Printf("Owner             : %s\n", token.Owner)
		fmt.Printf("Installation      : %d\n", token.InstallationID)
		fmt.Printf("Repositories      : %v\n", token.Repositories)
		fmt.Printf("Permissions       : %v\n", token.Permissions)
		fmt.Printf("BotUsername       : %s\n", token.BotUsername)
		fmt.Printf("BotCommitterEmail : %s\n", token.BotCommitterEmail)
	}
	return nil
}

func main() {
	flag.StringVar(&privFile, "private-key", "", "Path to PKCS1 private key file (required)")
	flag.Uint64Var(&app, "app-id", 0, "GitHub app ID (required)")
	flag.Uint64Var(&installation, "installation-id", 0, "Installation ID")
	flag.StringVar(&repos, "repos", "", "Comma separated list of repositories")
	flag.StringVar(&owner, "owner", "", "Installation owner")
	flag.StringVar(&format, "format", "", "Output format template")
	flag.BoolVar(&revoke, "revoke", false, "Revoke all tokens provided")

	flag.Usage = Usage
	flag.Parse()

	err := run()
	if err != nil {
		slog.Error("Error", "err", err)
		os.Exit(1)
	}
}
