// SPDX-FileCopyrightText: Copyright 2024 Prasad Tengse
// SPDX-License-Identifier: MIT

package api

// JWT header. This is always of type RS256.
type JWTHeader struct {
	Type string `json:"typ"`
	Alg  string `json:"alg"`
}

// JWTPayload as required by GitHub app.
type JWTPayload struct {
	Issuer   string `json:"iss"`
	IssuedAt int64  `json:"iat"`
	Exp      int64  `json:"exp"`
}

// EncodedJWTHeader is pre-encoded JWT header. GitHub apps only use
// RS256 JWT. Use pre-encoded header.
const EncodedJWTHeader = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9"
