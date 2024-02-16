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
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"time"
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

// IsValid checks if [JWT] is valid for at-least 60 seconds.
func (t JWT) IsValid() bool {
	now := time.Now()
	return t.Token != "" && t.IssuedAt.Before(now) && t.Exp.After(now.Add(time.Minute))
}

// contextSigner is similar to [crypto.Signer] but is context-aware.
type contextSigner interface {
	SignContext(ctx context.Context, rand io.Reader, digest []byte, opt crypto.SignerOpts) ([]byte, error)
}

// jwtMinter mints GitHub app JWT.
type jwtMinter interface {
	MintJWT(ctx context.Context, iss uint64, now time.Time) (JWT, error)
}

// jwtRS256 mints JWT tokens using RS256.
type jwtRS256 struct {
	internal crypto.Signer
}

// JWT header. This is always of type RS256.
type jwtHeader struct {
	Type string `json:"type"`
	Alg  string `json:"alg"`
}

// JWT Payload as required by GitHub app.
type jwtPayload struct {
	Issuer   string `json:"iss"`
	IssuedAt int64  `json:"iat"`
	Exp      int64  `json:"exp"`
}

// MintJWT mints new  JWT token.
func (s *jwtRS256) MintJWT(ctx context.Context, iss uint64, now time.Time) (JWT, error) {
	// GitHub rejects timestamps that are not an integer.
	now = now.Truncate(time.Second)
	iat := now.Add(-30 * time.Second)
	exp := now.Add(2 * time.Minute)

	buf := bytes.NewBuffer(make([]byte, 0, 1024))
	encoder := base64.NewEncoder(base64.RawURLEncoding, buf)

	// Encode JWT Header.
	header, err := json.Marshal(&jwtHeader{Alg: "RS256", Type: "JWT"})
	if err != nil {
		return JWT{}, fmt.Errorf("githubapp(jwt): failed to encode JWT header: %w", err)
	}
	_, _ = encoder.Write(header)
	_ = encoder.Close()

	// Write separator.
	_ = buf.WriteByte('.')

	// Encode JWT Payload.
	payload, err := json.Marshal(&jwtPayload{
		Issuer:   strconv.FormatUint(iss, 10),
		Exp:      exp.Unix(),
		IssuedAt: iat.Unix(),
	})
	if err != nil {
		return JWT{}, fmt.Errorf("githubapp(jwt): failed to encode JWT payload: %w", err)
	}
	_, _ = encoder.Write(payload)
	_ = encoder.Close()

	// Sign JWT header and payload.
	hasher := sha256.New()
	_, _ = hasher.Write(buf.Bytes())

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
		return JWT{}, fmt.Errorf("githubapp(jwt): failed to sign JWT: %w", err)
	}

	// Write separator.
	buf.WriteByte('.')

	// Encode signature.
	_, _ = encoder.Write(signature)
	_ = encoder.Close()

	// BearerToken may be missing some metadata, but it will be handled by Transport.JWT.
	return JWT{Token: buf.String(), Exp: exp, IssuedAt: iat, AppID: iss}, nil
}

// NewJWT returns new JWT bearer token signed by the signer.
//
// Returned JWT is valid for at least 5min. Ensure that your machine's clock is accurate.
//
//   - Unlike [NewTransport], this does not validate app id and signer. This simply
//     mints the JWT as required by GitHub app authentication.
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
		return minter.MintJWT(ctx, appid, time.Now())
	default:
		return JWT{}, fmt.Errorf("githubapp(jwt): unsupported key type: %T", v)
	}
}
