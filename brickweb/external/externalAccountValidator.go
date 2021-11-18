/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

//Package external provides interfaces for hooking into the ACME-Issuance Lifetime
//, e.g. providing an external account validator, or hooking into the Authorization Cycle
package external

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"

	jose "gopkg.in/square/go-jose.v2"
)

//AccountValidator can be hooked into the account creation process and return arbitrary data to
//be stored and possibly used later by other hooks
type AccountValidator interface {
	Validate(context.Context, map[string]interface{}) (Identifier, error)
}

//Identifier represents (for now) the end-user entity (e.g. a username) in an external system, as it is stored in DB
type Identifier string

var (
	//SymmetricToken is the Validator that checks a token against a symmetric key
	SymmetricToken = "symmetric_token"
	//AsymmetricToken is the Validator that checks a token against an asymmetric key
	AsymmetricToken = "asymmetric_token"
)

func GetValidator(validatorName string, validatorConfig map[string]interface{}) AccountValidator {
	switch validatorName {
	case SymmetricToken:
		validator, err := NewExternalSymmetricValidator([]byte(validatorConfig["key"].(string)), jose.HS256)
		if err != nil {
			panic(err)
		}
		return validator
	case AsymmetricToken:
		keyPemRaw, err := ioutil.ReadFile(validatorConfig["key_file"].(string))
		if err != nil {
			panic("could not Read key_file for asymmetric validator config")
		}
		p, _ := pem.Decode(keyPemRaw)
		privKey, err := x509.ParsePKCS8PrivateKey(p.Bytes)
		if err != nil {
			panic("could not parse pkcs8private key")
		}
		validator, err := NewExternalAsymmetricValidator(privKey, jose.RS256)
		if err != nil {
			panic(err)
		}
		return validator

	default:
		panic("NYI")
	}
}
