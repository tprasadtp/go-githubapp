// SPDX-FileCopyrightText: Copyright 2023 Prasad Tengse
// SPDX-License-Identifier: MIT

package githubapp

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/tprasadtp/go-githubapp/internal/testkeys"
)

var (
	_ contextSigner = (*ctxSigner)(nil)
	_ crypto.Signer = (*ctxSigner)(nil)
	_ crypto.Signer = (*errSigner)(nil)
)

// errSigner always returns [os.ErrNotExist] on Sign.
type errSigner struct {
	signer crypto.Signer
}

func (s *errSigner) Sign(_ io.Reader, _ []byte, _ crypto.SignerOpts) ([]byte, error) {
	return nil, fmt.Errorf("errSigner always returns error: %w", os.ErrNotExist)
}

func (s *errSigner) Public() crypto.PublicKey {
	return s.signer.Public()
}

// ctxSigner will panic when calling Sign, as it supports SignContext.
type ctxSigner struct {
	signer crypto.Signer
}

func (s *ctxSigner) SignContext(ctx context.Context, rand io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	if cc := context.Cause(ctx); cc != nil {
		return nil, cc
	}
	return s.signer.Sign(rand, digest, opts)
}

func (s *ctxSigner) Sign(_ io.Reader, _ []byte, _ crypto.SignerOpts) ([]byte, error) {
	panic(fmt.Sprintf("%T supports SignContext, Sign should not be called", s))
}

func (s *ctxSigner) Public() crypto.PublicKey {
	return s.signer.Public()
}

func TestJWTSignerRS256_Valid(t *testing.T) {
	tt := []struct {
		name   string
		appid  uint64
		ctx    context.Context
		signer crypto.Signer
	}{
		{
			name:   "rsa-key",
			signer: testkeys.RSA2048(),
			appid:  99,
		},
		{
			name:   "ctx-signer-rsa-key",
			signer: &ctxSigner{signer: testkeys.RSA2048()},
			appid:  99,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			token, err := NewJWT(tc.ctx, tc.appid, tc.signer)

			if err != nil {
				t.Fatalf("Failed to sign JWT: %s", err)
			}

			// Do not use this in server side code which verifies JWT.
			// This is not robust nor cryptographically correct.
			// This is just enough for unit tests.
			parts := strings.Split(token.Token, ".")

			if len(parts) != 3 {
				t.Fatalf("Malformed JWT has %d parts", len(parts))
			}

			// Verify signature.
			data := parts[0] + "." + parts[1]
			hasher := sha256.New()
			hasher.Write([]byte(data))
			hash := hasher.Sum(nil)

			signatureDecoded, err := base64.RawURLEncoding.DecodeString(parts[2])
			if err != nil {
				t.Fatalf("signature is not base64 url encoded: %s", err)
			}

			err = rsa.VerifyPKCS1v15(
				tc.signer.Public().(*rsa.PublicKey),
				crypto.SHA256, hash, signatureDecoded)
			if err != nil {
				t.Fatalf("jwt signature is invalid: %s", err)
			}

			// Verify header
			headerDecoded, err := base64.RawURLEncoding.DecodeString(parts[0])
			if err != nil {
				t.Errorf("JWT header is not base64 url encoded: %s", err)
			}
			header := jwtHeader{}
			err = json.Unmarshal(headerDecoded, &header)
			if err != nil {
				t.Errorf("JWT header not JSON encoded: %s", err)
			}
			expectedHeader := jwtHeader{Alg: "RS256", Type: "JWT"}
			if !reflect.DeepEqual(expectedHeader, header) {
				t.Errorf("expected JWT header=%v, got=%v", expectedHeader, header)
			}

			// Verify Payload
			payloadDecoded, err := base64.RawURLEncoding.DecodeString(parts[1])
			if err != nil {
				t.Errorf("JWT payload is not base64 url encoded: %s", err)
			}
			payload := jwtPayload{}
			err = json.Unmarshal(payloadDecoded, &payload)
			if err != nil {
				t.Errorf("JWT payload not JSON encoded: %s", err)
			}

			appid, err := strconv.ParseUint(payload.Issuer, 10, 64)
			if err != nil {
				t.Errorf("payload.Issuer is not a integer")
			}

			if appid != tc.appid {
				t.Errorf("expected appid=%d, got=%d", tc.appid, appid)
			}
		})
	}
}

func TestJWTSignerRS256_Invalid(t *testing.T) {
	type testCase struct {
		name   string
		appid  uint64
		ctx    context.Context
		signer crypto.Signer
	}

	tt := []testCase{
		{
			name:  "no-key",
			appid: 99,
		},
		{
			name:   "ecdsa-key",
			signer: testkeys.ECP256(),
			appid:  99,
		},
		{
			name:   "rsa-key-1024",
			signer: testkeys.RSA1024(),
			appid:  99,
		},
		{
			name:   "invalid-app-id",
			signer: &ctxSigner{signer: testkeys.RSA2048()},
		},
		{
			name:   "signer-error",
			ctx:    context.Background(),
			signer: &errSigner{signer: testkeys.RSA2048()},
			appid:  99,
		},
		{
			name: "signer-ctx-cancelled-with-cause",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancelCause(context.Background())
				cancel(os.ErrPermission)
				return ctx
			}(),
			signer: &ctxSigner{signer: testkeys.RSA2048()},
			appid:  99,
		},
		{
			name: "ctx-signer-ctx-cancelled",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			signer: &ctxSigner{signer: testkeys.RSA2048()},
			appid:  99,
		},
		{
			name: "context-signer-ctx-cancelled-with-cause",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancelCause(context.Background())
				cancel(os.ErrPermission)
				return ctx
			}(),
			signer: &ctxSigner{signer: testkeys.RSA2048()},
			appid:  99,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			token, err := NewJWT(tc.ctx, tc.appid, tc.signer)

			if err == nil {
				t.Errorf("Expected error, got nil")
			}

			if !reflect.DeepEqual(token, JWT{}) {
				t.Errorf("Must return zero value %T upon errors", token)
			}
		})
	}
}

func TestJWT(t *testing.T) {
	t.Run("slog-log-valuer", func(t *testing.T) {
		now := time.Now().Truncate(time.Second)
		token := JWT{
			Exp:      now.Add(time.Minute + time.Second),
			IssuedAt: now.Add(-30 * time.Second),
			Token:    "token",
		}
		v := token.LogValue()
		for _, item := range v.Group() {
			if item.Key == "token" {
				if item.Value.Kind() != slog.KindString {
					t.Errorf("token should be of string kind: %s", item.Value.Kind())
				}
				if item.Value.String() == "token" {
					t.Errorf("token value should be redacted: %s", item.Value.String())
				}
			}
		}
	})
	t.Run("empty-value", func(t *testing.T) {
		token := JWT{}
		if token.IsValid() {
			t.Errorf("empty token should be invalid")
		}
	})
	t.Run("exp", func(t *testing.T) {
		now := time.Now().Truncate(time.Second)
		token := JWT{
			Exp:      now.Add(-time.Minute),
			IssuedAt: now,
			Token:    "token",
		}
		if token.IsValid() {
			t.Errorf("token should be invalid")
		}
	})
	t.Run("now+59s", func(t *testing.T) {
		now := time.Now().Truncate(time.Second)
		token := JWT{
			Exp:      now.Add(time.Minute - time.Second),
			IssuedAt: now,
			Token:    "token",
		}
		if token.IsValid() {
			t.Errorf("token should be invalid")
		}
	})
	t.Run("now+60s", func(t *testing.T) {
		now := time.Now().Truncate(time.Second)
		token := JWT{
			Exp:      now.Add(time.Minute + time.Second),
			IssuedAt: now,
			Token:    "token",
		}
		if !token.IsValid() {
			t.Errorf("token should be valid")
		}
	})
	t.Run("now+120s", func(t *testing.T) {
		now := time.Now().Truncate(time.Second)
		token := JWT{
			Exp:      now.Add(2 * time.Minute),
			IssuedAt: now,
			Token:    "token",
		}
		if !token.IsValid() {
			t.Errorf("token should be valid")
		}
	})
}

func BenchmarkMintJWT(b *testing.B) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		b.Fatalf("failed to generated rsa 2048 key: %s", err)
	}
	jwtSigner := jwtRS256{internal: key}
	ctx := context.Background()
	var v JWT

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v, _ = jwtSigner.Mint(ctx, 99, time.Now())
	}
	_ = v
}
