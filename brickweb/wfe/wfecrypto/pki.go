/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package wfecrypto

import (
	"crypto"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"

	"github.com/pkg/errors"
	jose "gopkg.in/square/go-jose.v2"
)

/*
 * KeyToID produces a string with the hex representation of the SHA256 digest
 * over a provided public key. We use this for acme.Account ID values
 * because it makes looking up a account by key easy (required by the spec
 * for retreiving existing account), and becauase it makes the reg URLs
 * somewhat human digestable/comparable.
 * Lifted from github.com/letsencrypt/pebble/wfe/wfe.go
 */
func KeyToID(key crypto.PublicKey) (string, error) {
	switch t := key.(type) {
	case *jose.JSONWebKey:
		if t == nil {
			return "", errors.New("Cannot compute ID of nil key")
		}
		return KeyToID(t.Key)
	case jose.JSONWebKey:
		return KeyToID(t.Key)
	default:
		keyDER, err := x509.MarshalPKIXPublicKey(key)
		if err != nil {
			return "", err
		}
		spkiDigest := sha256.Sum256(keyDER)
		return hex.EncodeToString(spkiDigest[:]), nil
	}
}
