// SPDX-FileCopyrightText: Copyright 2023 Prasad Tengse
// SPDX-License-Identifier: MIT

package githubapp

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"io"
	"log/slog"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
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

func TestJWTSignerRS256(t *testing.T) {
	type testCase struct {
		name   string
		ok     bool
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
		{
			name:   "rsa-key",
			signer: testkeys.RSA2048(),
			appid:  99,
			ok:     true,
		},
		{
			name:   "ctx-signer-rsa-key",
			signer: &ctxSigner{signer: testkeys.RSA2048()},
			appid:  99,
			ok:     true,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			token, err := NewJWT(tc.ctx, tc.appid, tc.signer)

			if tc.ok {
				if err != nil {
					t.Errorf("Failed to sign JWT: %s", err)
				}

				pubKeyFunc := func(t *jwt.Token) (any, error) {
					return tc.signer.Public(), nil
				}

				_, err = jwt.Parse(token.Token, pubKeyFunc)
				if err != nil {
					t.Errorf("Failed to parse jwt: %s", err)
				}
			} else {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}

				if !reflect.DeepEqual(token, JWT{}) {
					t.Errorf("Must return zero value %T upon errors", token)
				}
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

	b.Run("jwt/v5", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			v, err = jwtSigner.Mint(ctx, 99, time.Now())
		}
	})

	_ = v
}
