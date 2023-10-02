// SPDX-FileCopyrightText: Copyright 2023 Prasad Tengse
// SPDX-License-Identifier: MIT

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

	"github.com/tprasadtp/githubapp"
)

var port uint
var dir string
var secret string
var wg sync.WaitGroup

func main() {
	flag.UintVar(&port, "port", 8899, "webhook server port")
	flag.StringVar(&dir, "dir", "", "webhook request log dir")
	flag.StringVar(&secret, "secret", "", "webhook secret")
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
		var serr error
		defer wg.Done()
		//nolint:gosimple // https://github.com/dominikh/go-tools/issues/503
		for {
			select {
			// on stop, return and stops ticks.
			case <-ctx.Done():
				log.Printf("Stopping server - %s", srv.Addr)
				serr = srv.Shutdown(ctx)
				if serr != nil && !errors.Is(serr, http.ErrServerClosed) {
					slog.Error("failed to shutdown server: %s", serr)
				}
				return
			}
		}
	}()

	// Start server
	slog.Info("Starting server", "addr", srv.Addr)
	err = srv.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("Failed to start the server", "err", err)
		wg.Wait()
		os.Exit(1)
	}

	wg.Wait()
	log.Printf("Server stopped %s", srv.Addr)
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
			_, _ = w.Write([]byte("missing header X-GitHub-Delivery"))
			return
		}

		data, err := httputil.DumpRequest(r, true)
		if err != nil {
			slog.Error("failed to dump request", "err", err)
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error()))
			return
		}

		webhook, err := githubapp.VerifyWebHookRequest(secret, r)
		if err != nil {
			slog.Error("failed to verify webhook", "err", err)
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(err.Error()))
			return
		}

		slog.Info("webhook request", "webhook", webhook.LogValue())

		// create a file to dump raw request.
		file, err := os.Create(filepath.Join(dir, fmt.Sprintf("%s.replay", id)))
		if err != nil {
			slog.Error("failed to create file", "err", err)
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error()))
			return
		}
		defer file.Close()

		_, err = file.Write(data)
		if err != nil {
			slog.Error("failed to write to file", "err", err)
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error()))
			return
		}

		w.WriteHeader(http.StatusAccepted)
	})
	return mux
}
