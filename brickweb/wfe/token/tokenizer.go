/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package token

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

// randomString and newToken come from Boulder core/util.go
// randomString returns a randomly generated string of the requested length.
func randomString(byteLength int) string {
	b := make([]byte, byteLength)
	_, err := io.ReadFull(rand.Reader, b)
	if err != nil {
		panic(fmt.Sprintf("Error reading random bytes: %s", err))
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// newToken produces a random string for Challenges, etc.
func newToken() string {
	return randomString(128)
}

type Tokenizer interface {
	NewToken() string
}

type t struct {
	tokens chan string
}

func New() Tokenizer {
	var tok = &t{
		tokens: make(chan string, 2),
	}
	go func() {
		for {
			tok.tokens <- newToken()
		}
	}()
	return tok
}

func (tok *t) NewToken() string {
	x := <-tok.tokens
	return x
}
