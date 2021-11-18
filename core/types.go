/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package core

import (
	"brick/brickweb/external"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"os"
	"time"

	"brick/brickweb/acme"

	"github.com/sirupsen/logrus"
	jose "gopkg.in/square/go-jose.v2"
)

type Account struct {
	acme.Account
	Key                *jose.JSONWebKey `json:"key"`
	ID                 string
	CreatedAt          time.Time
	ExternalIdentifier external.Identifier
}

type Authorization struct {
	Status      string
	Identifier  acme.Identifier
	Wildcard    bool // false plz
	Challenges  []*Challenge
	ID          string
	ExpiresDate time.Time
	AccountID   string
}

//Challenge represents a acme.Challenge as it is stored internally
type Challenge struct {
	Token       string
	Status      string
	ValidatedAt time.Time //Check for time.IsZero
	Error       *acme.ProblemDetails
	Type        string
	ID          string
	AuthzID     string
}

type Order struct {
	acme.Order
	ID            string
	CertificateID string
	AuthzIDs      []string
	AccountID     string
}

//Certificate is the internal representation of an ACME Certificate Object
type Certificate struct {
	ID             string
	Cert           *x509.Certificate
	DER            []byte
	IssuerID       string
	IssuerNameHash []byte
	RevocationTime time.Time
	Serial         *big.Int
	OrderID        string
}

func (c Certificate) PEM() []byte {
	var buf bytes.Buffer

	err := pem.Encode(&buf, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: c.DER,
	})
	if err != nil {
		panic(fmt.Sprintf("Unable to encode certificate %q to PEM: %s",
			c.ID, err.Error()))
	}

	return buf.Bytes()
}

//CertificateChain represents a chain from leaf Cert (index 0) up to a trusted Root
type CertificateChain []*x509.Certificate

//PEM returns a PEM-Chain
func (ch CertificateChain) PEM(OmitRoot bool, logger logrus.FieldLogger) []byte {
	if ch == nil {
		panic("Nil Reference in PEM Generation")
	}
	c := []*x509.Certificate(ch)
	if len(c) == 0 {
		panic("Certificate Chain of Length 0 ?")
	}

	chain := make([][]byte, 0)
	d := make([]byte, 8)
	_, _ = rand.Read(d)
	id := hex.EncodeToString(d)
	fd, err := os.Create(fmt.Sprintf("/tmp/cert-%s.pem", id))
	if err == nil {
		defer fd.Close()
		logger.Info("Writing Returned Cert to %s", fmt.Sprintf("/tmp/cert-%s.pem", id))
	} else {
		fd = nil
	}

	for i, cert := range c {
		if OmitRoot == true && i == len(c)-1 {
			continue
		}
		var buf bytes.Buffer
		var writer io.Writer = &buf
		if fd != nil {
			writer = io.MultiWriter(fd, &buf)
		}
		_ = pem.Encode(writer, &pem.Block{
			Type:  "CERTIFICATE",
			Bytes: cert.Raw,
		})
		chain = append(chain, buf.Bytes())
	}
	return bytes.Join(chain, nil)
}

type ValidationRecord struct {
	URL         string
	Error       *acme.ProblemDetails
	ValidatedAt time.Time
}

//AddOrderRequest is a special Case of a Data Exchange Structure being defined inside this Package.
//This is because Adding an Order is a complicated request to express with the core.Order Datatype
type AddOrderRequest struct {
	Authz                  []string
	ExpiresDate            string
	RequestedNotBeforeDate string
	RequestedNotAfterDate  string
	AccountID              string
}

//AddAuthz builds a new Authz, representing a stub Authz
type AddAuthz struct {
	Challenges  []AddChallenge
	ExpiresDate string
	Identifier  acme.Identifier
	AccountID   string
}

type AddChallenge struct {
	Type  string
	Token string
}

type CaCertificate struct {
	NameHash   []byte
	DER        []byte
	WillIssue  bool
	CommonName string
	ID         string
}

type VerificationRequest struct {
	Context       context.Context
	Challenge     *Challenge
	Authorization *Authorization
	AccountJWK    *jose.JSONWebKey
	Retries       int //Added, does not break compatibility, becuase Zero Value is: 0
}
