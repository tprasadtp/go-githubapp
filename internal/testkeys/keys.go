// SPDX-FileCopyrightText: Copyright 2023 Prasad Tengse
// SPDX-License-Identifier: MIT

// Package testkeys generates ephemeral test keys.
//
// Generated keys are unique per execution of the binary and are generated
// on demand.
//
// DO NOT USE THESE KEYS OUTSIDE OF UNIT TESTING.
package testkeys

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"sync"
)

var (
	rsa1024Once   sync.Once
	rsa2048Once   sync.Once
	ecdsaP256Once sync.Once
	ed25519Once   sync.Once
)

var (
	rsa1024Private   *rsa.PrivateKey
	rsa2048Private   *rsa.PrivateKey
	ecdsaP256Private *ecdsa.PrivateKey
	ed25519Private   ed25519.PrivateKey
)

// Ephemeral RSA-1024 key which is unique per execution of the binary.
func RSA1024() *rsa.PrivateKey {
	rsa1024Once.Do(func() {
		//nolint:gosec // check to ensure key size < 2048 is rejected.
		rsa1024Private, _ = rsa.GenerateKey(rand.Reader, 1024)
	})
	return rsa1024Private
}

// Ephemeral RSA-2048 key which is unique per execution of the binary.
func RSA2048() *rsa.PrivateKey {
	rsa2048Once.Do(func() {
		rsa2048Private, _ = rsa.GenerateKey(rand.Reader, 2048)
	})
	return rsa2048Private
}

// Ephemeral ECDSA-P256 key which is unique per execution of the binary.
func ECP256() *ecdsa.PrivateKey {
	ecdsaP256Once.Do(func() {
		ecdsaP256Private, _ = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	})
	return ecdsaP256Private
}

// Ephemeral ED25519 key which is unique per execution of the binary.
func ED25519() ed25519.PrivateKey {
	ed25519Once.Do(func() {
		_, ed25519Private, _ = ed25519.GenerateKey(rand.Reader)
	})
	return ed25519Private
}
