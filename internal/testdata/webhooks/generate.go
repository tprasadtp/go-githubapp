// SPDX-FileCopyrightText: Copyright 2023 Prasad Tengse
// SPDX-License-Identifier: MIT

// Simple CLI tool to dump received GitHub webhook requests to directory.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/tprasadtp/go-githubapp"
)

var port uint
var dir string
var secret string
var wg sync.WaitGroup

func Usage() {
	fmt.Fprintf(flag.CommandLine.Output(), "CLI to dump github webhook requests to directory.\n\n")
	fmt.Fprintf(flag.CommandLine.Output(), "This is not covered by semver compatibility guarantees.\n")
	fmt.Fprintf(flag.CommandLine.Output(), "Usage: go run internal/testdata/webhooks/generate.go\n\n")
	fmt.Fprintf(flag.CommandLine.Output(), "Flags:\n")
	flag.PrintDefaults()
}

func main() {
	flag.UintVar(&port, "port", 8899, "webhook server port")
	flag.StringVar(&dir, "dir", "", "webhook request log dir")
	flag.StringVar(&secret, "secret", "", "webhook secret")
	flag.Usage = Usage
	flag.Parse()

	if secret == "" {
		secret = os.Getenv("GH_WEBHOOK_SECRET")
	}

	if secret == "" {
		secret = os.Getenv("WEBHOOK_SECRET")
	}

	stat, err := os.Stat(dir)
	if err != nil {
		slog.Error("Error stating dir", "dir", dir, "err", err)
		os.Exit(1)
	}
	if !stat.IsDir() {
		slog.Error("Invalid data dir", "dir", dir)
		os.Exit(1)
	}

	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	srv := http.Server{
		Addr:              fmt.Sprintf("localhost:%d", port),
		ReadHeaderTimeout: time.Second,
		Handler:           Mux(),
	}

	// Starts a go routine which handles server shutdown.
	wg.Add(1)
	go func() {
		var err error
		defer wg.Done()
		//nolint:gosimple // https://github.com/dominikh/go-tools/issues/503
		for {
			select {
			// on cancel, stops server and return.
			case <-ctx.Done():
				log.Printf("Stopping server - %s", srv.Addr)
				err = srv.Shutdown(ctx)
				if err != nil && !errors.Is(err, http.ErrServerClosed) {
					slog.Error("Failed to shutdown server", slog.Any("err", err))
				}
				return
			}
		}
	}()

	// Start server.
	slog.Info("Starting server", slog.String("addr", srv.Addr))
	err = srv.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("Failed to start the server", slog.Any("err", err))
		wg.Wait()
		os.Exit(1)
	}

	wg.Wait()
	slog.Info("Server stopped", slog.String("addr", srv.Addr))
}

func Mux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})

	mux.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-GitHub-Delivery")
		if id == "" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("Missing header: X-GitHub-Delivery"))
			return
		}

		data, err := httputil.DumpRequest(r, true)
		if err != nil {
			slog.Error("Failed to dump request", slog.Any("err", err))
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error()))
			return
		}

		webhook, err := githubapp.VerifyWebHookRequest(secret, r)
		if err != nil {
			slog.Error("Failed to verify webhook", slog.Any("err", err))
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(err.Error()))
			return
		}

		slog.Info("webhook request", "webhook", webhook.LogValue())

		// create a file to dump raw request.
		file, err := os.Create(filepath.Join(dir, fmt.Sprintf("%s.replay", id)))
		if err != nil {
			slog.Error("Failed to create file", slog.Any("err", err))
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error()))
			return
		}
		defer file.Close()

		_, err = file.Write(data)
		if err != nil {
			slog.Error("Failed to write to file", slog.Any("err", err))
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error()))
			return
		}

		w.WriteHeader(http.StatusAccepted)
	})
	return mux
}
