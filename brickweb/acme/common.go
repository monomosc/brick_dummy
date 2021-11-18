/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package acme

import (
	"context"
	"crypto"
	"encoding/base64"
	"fmt"

	opentracing "github.com/opentracing/opentracing-go"
	jose "gopkg.in/square/go-jose.v2"
)

// acme.Resource values identify different types of ACME resources
type Resource string

const (
	StatusPending     = "pending"
	StatusInvalid     = "invalid"
	StatusValid       = "valid"
	StatusExpired     = "expired"
	StatusProcessing  = "processing"
	StatusReady       = "ready"
	StatusDeactivated = "deactivated"

	IdentifierDNS = "dns"

	ChallengeHTTP01    = "http-01"
	ChallengeTLSALPN01 = "tls-alpn-01"
	ChallengeDNS01     = "dns-01"

	HTTP01BaseURL = ".well-known/acme-challenge/"

	ACMETLS1Protocol = "acme-tls/1"
)

type ChallengeType string

type Identifier struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type Account struct {
	Status  string   `json:"status"`
	Contact []string `json:"contact"`
	Orders  string   `json:"orders,omitempty"`
}

// An Order is created to request issuance for a CSR
type Order struct {
	Status         string          `json:"status"`
	Error          *ProblemDetails `json:"error,omitempty"`
	Expires        string          `json:"expires"`
	Identifiers    []Identifier    `json:"identifiers,omitempty"`
	Finalize       string          `json:"finalize"`
	NotBefore      string          `json:"notBefore,omitempty"`
	NotAfter       string          `json:"notAfter,omitempty"`
	Authorizations []string        `json:"authorizations"`
	Certificate    string          `json:"certificate,omitempty"`
}

// An Authorization is created for each identifier in an order
type Authorization struct {
	Status     string       `json:"status"`
	Identifier Identifier   `json:"identifier"`
	Challenges []*Challenge `json:"challenges"`
	Expires    string       `json:"expires"`
	// Wildcard is a Let's Encrypt specific Authorization field that indicates the
	// authorization was created as a result of an order containing a name with
	// a `*.`wildcard prefix. This will help convey to users that an
	// Authorization with the identifier `example.com` and one DNS-01 challenge
	// corresponds to a name `*.example.com` from an associated order.
	Wildcard bool `json:"wildcard,omitempty"`
}

// A Challenge is used to validate an Authorization
type Challenge struct {
	Type      string          `json:"type"`
	URL       string          `json:"url"`
	Token     string          `json:"token"`
	Status    string          `json:"status"`
	Validated string          `json:"validated,omitempty"`
	Error     *ProblemDetails `json:"error,omitempty"`
}

//ExpectedKeyAuthorization returns the <ACME> RFC 8.1 expected token constructed from the account key and the challenge token
func ExpectedKeyAuthorization(ctx context.Context, token string, key *jose.JSONWebKey) (string, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "ExpectedKeyAuthorization")
	defer span.Finish()
	span.SetTag("token", token)
	thumprint, err := key.Thumbprint(crypto.SHA256) //see acme rfc section 8.1
	if err != nil {
		span.SetTag("error", true)
		span.LogKV("event", "error", "error.object", err, "message", "Could not thumprint passed JWK")
		return "", err
	}
	return fmt.Sprintf("%s.%s", token, base64.RawURLEncoding.EncodeToString(thumprint)), nil
}

//AccountCreation Object according to ACME-Draft-14 Section7.3
//AgreeToTOS is unnecessary in an Enterprise Environment
type AccountCreation struct {
	Contact                []string               `json:"contact"`
	ExternalAccountBinding map[string]interface{} `json:"externalAccountBinding"`
	OnlyReturnExisting     bool                   `json:"onlyReturnExisting"`
}
