// SPDX-FileCopyrightText: Copyright 2023 Prasad Tengse
// SPDX-License-Identifier: MIT

package githubapp

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	_ jwtMinter      = (*jwtRS256)(nil)
	_ slog.LogValuer = (*JWT)(nil)
)

// JWT is JWT token used to authenticate as app.
type JWT struct {
	// JWT token.
	Token string `json:"token" yaml:"token"`

	// GitHub app ID.
	AppID uint64 `json:"app_id,omitempty" yaml:"appID,omitempty"`

	// GitHub app name.
	AppName string `json:"app_name,omitempty" yaml:"appName,omitempty"`

	// Token exp time.
	Exp time.Time `json:"exp,omitempty" yaml:"exp,omitempty"`

	// Token issue time.
	IssuedAt time.Time `json:"iat,omitempty" yaml:"iat,omitempty"`
}

// LogValue implements [log/slog.LogValuer].
func (t JWT) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Uint64("app_id", t.AppID),
		slog.String("app_name", t.AppName),
		slog.Time("exp", t.Exp),
		slog.Time("iat", t.IssuedAt),
		slog.String("token", "REDACTED"),
	)
}

// Checks if [JWT] is valid for at-least 60 seconds.
func (t JWT) IsValid() bool {
	now := time.Now()
	return t.Token != "" && t.IssuedAt.Before(now) && t.Exp.After(now.Add(time.Minute))
}

// contextSigner is similar to [crypto.Signer] but is context aware.
type contextSigner interface {
	SignContext(ctx context.Context, rand io.Reader, digest []byte, opt crypto.SignerOpts) ([]byte, error)
}

// jwtMinter mints github app JWT.
type jwtMinter interface {
	Mint(ctx context.Context, iss uint64, now time.Time) (JWT, error)
}

// jwtRS256 mints JWT tokens using RS256.
type jwtRS256 struct {
	internal crypto.Signer
}

// Mint mints new  JWT token.
func (s *jwtRS256) Mint(ctx context.Context, iss uint64, now time.Time) (JWT, error) {
	// GitHub rejects expiry and issue timestamps that are not an integer,
	// Truncate them before passing to jwt-go.
	now = now.Truncate(time.Second)
	iat := now.Add(-30 * time.Second)
	exp := now.Add(2 * time.Minute)
	t := jwt.NewWithClaims(jwt.SigningMethodRS256,
		&jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(iat),
			ExpiresAt: jwt.NewNumericDate(exp),
			Issuer:    strconv.FormatUint(iss, 10),
		},
	)

	signingString, err := t.SigningString()
	if err != nil {
		return JWT{}, fmt.Errorf("githubapp(jwt): failed to get jwt string: %w", err)
	}

	hasher := sha256.New()
	_, _ = hasher.Write([]byte(signingString))

	var signature []byte

	// github.com/tprasadtp/cryptokms supports SignContext. try to check if we can use
	// context aware signer, fallback to default.
	if cs, ok := s.internal.(contextSigner); ok {
		if ctx == nil {
			signature, err = cs.SignContext(context.Background(), rand.Reader, hasher.Sum(nil), crypto.SHA256)
		} else {
			signature, err = cs.SignContext(ctx, rand.Reader, hasher.Sum(nil), crypto.SHA256)
		}
	} else {
		signature, err = s.internal.Sign(rand.Reader, hasher.Sum(nil), crypto.SHA256)
	}

	if err != nil {
		return JWT{}, fmt.Errorf("githubapp(jwt): failed to mint JWT: %w", err)
	}

	buf := bytes.NewBufferString(signingString)
	buf.WriteByte('.')
	buf.WriteString(t.EncodeSegment(signature))

	// BearerToken has incomplete metadata, but it will be handled by Transport.JWT.
	return JWT{Token: buf.String(), Exp: exp, IssuedAt: iat}, nil
}

// NewJWT returns new JWT bearer token signed by the signer.
//
// Returned JWT is valid for at-least 5min. Ensure that your machine's clock is accurate.
//
//   - Unlike [NewTransport], this does not validate app id and signer. This simply
//     mints the JWT as required by github app authentication.
//   - RSA keys of length less than 2048 bits are not supported.
//   - Only RSA keys are supported. Using ECDSA, ED25519 or other keys will return error.
func NewJWT(ctx context.Context, appid uint64, signer crypto.Signer) (JWT, error) {
	var err error
	if signer == nil {
		err = errors.Join(err, errors.New("no signer provided"))
	}

	if appid == 0 {
		err = errors.Join(err, errors.New("app id cannot be zero"))
	}

	if err != nil {
		return JWT{}, fmt.Errorf("githubapp(jwt): failed to mint JWT: %w", err)
	}

	switch v := signer.Public().(type) {
	case *rsa.PublicKey:
		if v.N.BitLen() < 2048 {
			return JWT{},
				fmt.Errorf("githubapp(jwt): rsa keys size(%d) < 2048 bits", v.N.BitLen())
		}
		minter := &jwtRS256{internal: signer}
		return minter.Mint(ctx, appid, time.Now())
	default:
		return JWT{}, fmt.Errorf("githubapp(jwt): unsupported key type: %T", v)
	}
}
