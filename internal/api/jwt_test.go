// SPDX-FileCopyrightText: Copyright 2024 Prasad Tengse
// SPDX-License-Identifier: MIT

package api

import (
	"encoding/base64"
	"encoding/json"
	"reflect"
	"testing"
)

func TestEncodedJWTHeader(t *testing.T) {
	expect := JWTHeader{
		Type: "JWT",
		Alg:  "RS256",
	}

	decoded, err := base64.RawURLEncoding.DecodeString(EncodedJWTHeader)
	if err != nil {
		t.Fatalf("EncodedJWTHeader is invalid: %s", err)
	}
	v := JWTHeader{}

	err = json.Unmarshal(decoded, &v)
	if err != nil {
		t.Errorf("decoded EncodedJWTHeader is not a valid JSON: %s", err)
	}

	if !reflect.DeepEqual(v, expect) {
		t.Errorf("expected=%+v, got=%+v", expect, v)
	}
}
