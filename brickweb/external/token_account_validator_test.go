/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package external_test

import (
	"brick/brickweb/external"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	jose "gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
)

var key = []byte("123456789123456789")
var secondkey = []byte("incorrect-key")
var name = "hello"

func TestSymmetricJWTIsParsedCorrectly(t *testing.T) {
	Convey("When a new JWT with a key is created", t, func() {
		validator, err := external.NewExternalSymmetricValidator(key, jose.HS256)
		So(err, ShouldBeNil)
		signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.HS256, Key: key}, (&jose.SignerOptions{}).WithType("JWT"))
		So(err, ShouldBeNil)
		cl := external.Claims{
			Name: name,
		}
		token, err := jwt.Signed(signer).Claims(cl).FullSerialize()
		So(err, ShouldBeNil)
		Convey("It should parse correctly, and return the name", func() {
			id, err := validator.Validate(context.Background(), map[string]interface{}{"token": token})
			So(err, ShouldBeNil)
			So(id, ShouldEqual, name)
		})
	})
}

func TestBadSymmetricJWTIsFailsVerify(t *testing.T) {
	Convey("When a new JWT with an incorrect key is created", t, func() {
		validator, err := external.NewExternalSymmetricValidator(key, jose.HS256)
		So(err, ShouldBeNil)
		signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.HS256, Key: secondkey}, (&jose.SignerOptions{}).WithType("JWT"))
		So(err, ShouldBeNil)
		cl := external.Claims{
			Name: name,
		}
		token, err := jwt.Signed(signer).Claims(cl).FullSerialize()
		So(err, ShouldBeNil)
		Convey("It should fail, because HMAC keys do not match", func() {
			_, err = validator.Validate(context.Background(), map[string]interface{}{"token": token})
			So(err, ShouldNotBeNil)
		})
	})
}

func TestInvalidKeyFailsSymmetric(t *testing.T) {
	Convey("When an empty Key is passed", t, func() {
		_, err := external.NewExternalSymmetricValidator([]byte(""), jose.HS256)
		Convey("it should fail", func() {
			So(err, ShouldBeError)
			So(err, ShouldNotBeNil)
		})
	})
	Convey("When a nil Key is passed", t, func() {
		_, err := external.NewExternalSymmetricValidator(nil, jose.HS256)
		Convey("it should fail", func() {
			So(err, ShouldBeError)
			So(err, ShouldNotBeNil)
		})
	})
}

func TestAsymmetricJWTIsParsedCorrectly(t *testing.T) {
	Convey("With a new ES384 Key", t, func() {
		priv, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
		So(err, ShouldBeNil)
		signer, err := jose.NewSigner(jose.SigningKey{Key: priv, Algorithm: jose.ES384}, (&jose.SignerOptions{}).WithType("JWT"))
		So(err, ShouldBeNil)
		cl := external.Claims{
			Name: name,
		}
		Convey("When a new Token is created", func() {
			token, err := jwt.Signed(signer).Claims(cl).FullSerialize()
			So(err, ShouldBeNil)
			Convey("The PublicTokenValidator should validate correctly", func() {
				validator, err := external.NewExternalAsymmetricValidator(priv.Public(), jose.ES384)
				So(err, ShouldBeNil)
				id, err := validator.Validate(context.Background(), map[string]interface{}{"token": token})
				So(err, ShouldBeNil)
				So(id, ShouldEqual, name)
			})
		})
	})
}
