/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package wfecrypto

import (
	"brick/brickweb/policy"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"strings"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"gopkg.in/square/go-jose.v2"
)

func AlgorithmForKey(key *jose.JSONWebKey) (string, error) {
	switch k := key.Key.(type) {
	case *rsa.PublicKey:
		return string(jose.RS256), nil
	case *ecdsa.PublicKey:
		switch k.Params().Name {
		case "P-256":
			return string(jose.ES256), nil
		case "P-384":
			return string(jose.ES384), nil
		case "P-521":
			return string(jose.ES512), nil
		}
	}
	return "", errors.New("no signature algorithms suitable for given key type")
}

const (
	NoAlgorithmForKey     = "WFE.Errors.NoAlgorithmForKey"
	InvalidJWSAlgorithm   = "WFE.Errors.InvalidJWSAlgorithm"
	InvalidAlgorithmOnKey = "WFE.Errors.InvalidAlgorithmOnKey"
)

// CheckAlgorithm checks that (1) there is a suitable algorithm for the provided key based on its
// Golang type, (2) the Algorithm field on the JWK is either absent, or matches
// that algorithm, (3) the Algorithms compliance,
// and (4) the Algorithm field on the JWK is present and matches
// that algorithm. Precondition: parsedJws must have exactly one signature on
// it. Returns stat name to increment if err is non-nil.
func CheckAlgorithm(ctx context.Context, key *jose.JSONWebKey, parsedJws *jose.JSONWebSignature) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "CheckAlgorithm")
	defer span.Finish()
	algorithm, err := AlgorithmForKey(key)
	if err != nil {
		setSpanError(span, err, NoAlgorithmForKey)
		return err
	}
	jwsAlgorithm := parsedJws.Signatures[0].Header.Algorithm
	if jwsAlgorithm != algorithm {
		err := fmt.Errorf(
			"signature type '%s' in JWS header is not supported, expected one of RS256, ES256, ES384 or ES512",
			jwsAlgorithm)
		return err
	}
	allowedAlgorithms := policy.GetAllowedJWSAlgorithms()
	ok := func() bool {
		for _, algo := range allowedAlgorithms {
			if algo == algorithm {
				return true
			}
		}
		return false
	}()
	if !ok {
		err := fmt.Errorf("algorithm '%s' is disallowed by Policy. Use one of %s", algorithm, strings.Join(allowedAlgorithms, ","))
		setSpanError(span, err, InvalidJWSAlgorithm)
		return err

	}
	if key.Algorithm != "" && key.Algorithm != algorithm {
		err := fmt.Errorf("algorithm '%s' on JWK is unacceptable", key.Algorithm)
		setSpanError(span, err, InvalidAlgorithmOnKey)
		return err
	}
	return nil
}

//KeyDigest produces a padded, standard Base64-encoded SHA256 digest of a
// provided public key. See the original Boulder implementation for more details:
// https://github.com/letsencrypt/boulder/blob/9c2859c87b70059a2082fc1f28e3f8a033c66d43/core/util.go#L92
func KeyDigest(key crypto.PublicKey) (string, error) {
	switch t := key.(type) {
	case *jose.JSONWebKey:
		if t == nil {
			return "", errors.New("Cannot compute digest of nil key")
		}
		return KeyDigest(t.Key)
	case jose.JSONWebKey:
		return KeyDigest(t.Key)
	default:
		keyDER, err := x509.MarshalPKIXPublicKey(key)
		if err != nil {
			return "", err
		}
		spkiDigest := sha256.Sum256(keyDER)
		return base64.StdEncoding.EncodeToString(spkiDigest[0:32]), nil
	}
}

//KeyDigestEquals determines whether two public keys have the same digest.
func KeyDigestEquals(j, k crypto.PublicKey) bool {
	digestJ, errJ := KeyDigest(j)
	digestK, errK := KeyDigest(k)
	// Keys that don't have a valid digest (due to marshalling problems)
	// are never equal. So, e.g. nil keys are not equal.
	if errJ != nil || errK != nil {
		return false
	}
	return digestJ == digestK
}

func setSpanError(span opentracing.Span, err error, msg string) {
	span.SetTag("error", true)
	span.LogKV(
		"event", "error",
		"error.object", err,
		"message", msg,
	)
}
