// SPDX-FileCopyrightText: Copyright 2023 Prasad Tengse
// SPDX-License-Identifier: MIT

package testkeys_test

import (
	"testing"

	"github.com/tprasadtp/go-githubapp/internal/testkeys"
)

func TestKeys(t *testing.T) {
	t.Run("RSA-1024", func(t *testing.T) {
		key := testkeys.RSA1024()
		if key.PublicKey.N.BitLen() != 1024 {
			t.Errorf("expected rsa key size 1024, got %d", key.PublicKey.N.BitLen())
		}
	})

	t.Run("RSA-2048", func(t *testing.T) {
		key := testkeys.RSA2048()
		if key.PublicKey.N.BitLen() != 2048 {
			t.Errorf("expected rsa key size 2048, got %d", key.PublicKey.N.BitLen())
		}
	})

	t.Run("EC-P256", func(t *testing.T) {
		key := testkeys.ECP256()
		if key.Curve.Params().BitSize != 256 {
			t.Errorf("expected ecdsa key size 256, got %d", key.Curve.Params().BitSize)
		}
	})
}
