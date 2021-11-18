/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package external

import (
	"brick/brickweb/acme"
	"context"
	"crypto"
	"errors"
	"fmt"

	"github.com/opentracing/opentracing-go"
	jose "gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"

	"brick/core/berrors"
)

type symmetricTokenAccountValidator struct {
	key *jose.SigningKey
}
type asymmetricTokenAccountValidator struct {
	key *jose.JSONWebKey
}

func NewExternalSymmetricValidator(key []byte, algo jose.SignatureAlgorithm) (*symmetricTokenAccountValidator, error) {
	if len(key) < 5 { //Reject ridiculously short keys
		return nil, errors.New("Key-Length too short; c'mon")
	}
	return &symmetricTokenAccountValidator{key: &jose.SigningKey{Key: key, Algorithm: algo}}, nil
}

func NewExternalAsymmetricValidator(key crypto.PublicKey, algo jose.SignatureAlgorithm) (*asymmetricTokenAccountValidator, error) {
	return &asymmetricTokenAccountValidator{key: &jose.JSONWebKey{Key: key, Algorithm: string(algo)}}, nil
}

func (validator *symmetricTokenAccountValidator) Validate(ctx context.Context, externalAccountBinding map[string]interface{}) (Identifier, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "ValidateSymmetric")
	defer span.Finish()
	token, err := extractToken(externalAccountBinding)
	if err != nil {
		span.LogKV("Token not found or not a string")
		return "", err
	}
	parsedToken, err := jwt.ParseSigned(token)
	if err != nil {
		span.LogKV("event", "error", "error.object", err, "error", true, "message", "Could not parse JWT")
		return "", berrors.UnknownError(err)
	}
	var claims Claims
	err = parsedToken.Claims(validator.key.Key, &claims)
	if err != nil {
		span.LogKV("event", "error", "error.object", err, "error", true, "message", "JWT Claims validation failed or bad key")
		return "", acme.MalformedProblem("The Signature on your External Account Binding Token could not be verified")
	}
	span.LogKV("claims", claims)
	return Identifier(claims.Name), nil
}

func (validator *asymmetricTokenAccountValidator) Validate(ctx context.Context, externalAccountBinding map[string]interface{}) (Identifier, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "ValidateASymmetric")
	defer span.Finish()
	token, err := extractToken(externalAccountBinding)
	if err != nil {
		span.LogKV("Token not found or not a string")
		return "", err
	}
	span.SetTag("token", token)
	parsedToken, err := jwt.ParseSigned(token)
	if err != nil {
		span.LogKV("event", "error", "error.object", err, "error", true, "message", "Could not parse JWT")
		return "", berrors.UnknownError(err)
	}
	var claims Claims
	err = parsedToken.Claims(validator.key.Key, &claims)
	if err != nil {
		span.LogKV("event", "error", "error.object", err, "error", true, "message", "JWT Claims validation failed or bad key")
		return "", berrors.UnknownError(err)
	}
	span.LogKV("claims", claims)
	return Identifier(claims.Name), nil
}

func extractToken(externalAccountBinding map[string]interface{}) (string, error) {
	token, ok := externalAccountBinding["token"].(string)
	if !ok {
		return "", berrors.UnknownError(errors.New("Token not a string"))
	}
	return token, nil
}

//Claims represents the claims necessary from an external account
type Claims struct {
	Name string `json:"name"`
}

//String implements the Stringer interface
func (c Claims) String() string {
	return fmt.Sprintf("Claims: Name:%s", c.Name)
}
