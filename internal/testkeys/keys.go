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
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"sync"
)

var (
	rsa1024Once   sync.Once
	rsa2048Once   sync.Once
	ecdsaP256Once sync.Once
)

var (
	rsa1024Private   *rsa.PrivateKey
	rsa2048Private   *rsa.PrivateKey
	ecdsaP256Private *ecdsa.PrivateKey
)

// Ephemeral RSA-1024 key which is unique per execution of the binary.
func RSA1024() *rsa.PrivateKey {
	rsa1024Once.Do(func() {
		var err error
		//nolint:gosec // check to ensure key size < 2048 is rejected.
		rsa1024Private, err = rsa.GenerateKey(rand.Reader, 1024)
		if err != nil {
			panic(err)
		}
	})
	return rsa1024Private
}

// Ephemeral RSA-1024 key which is unique per execution of the binary.
func RSA2048() *rsa.PrivateKey {
	rsa2048Once.Do(func() {
		var err error
		rsa2048Private, err = rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			panic(err)
		}
	})
	return rsa2048Private
}

// Ephemeral ECDSA-P256 key which is unique per execution of the binary.
func ECP256() *ecdsa.PrivateKey {
	ecdsaP256Once.Do(func() {
		var err error
		ecdsaP256Private, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			panic(err)
		}
	})
	return ecdsaP256Private
}
