/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package nonce

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"sync"
)

//MaxUsed defines the maximum number of Nonces we're willing to hold in memory
const MaxUsed = 512
const nonceLen = 16

//Nonce is a Nonce in the Sense of ACME-RFC-Draft-14 Section 6.4.1
type Nonce string

//NonceService is the abstract interface a nonce-creator needs to implement
//In the future this might be a lockless B-Tree or a Skiplist
//For now we use a standard hashmap with channels for locking
type NonceService interface {
	Next() Nonce
	Valid(Nonce) bool
}

//NewNoncer creates the designed default implementation of a NonceService
func NewNoncer() NonceService {
	n := &nonceImpl{
		nextNonce: make(chan Nonce, 20),
		nonces:    make(map[Nonce]struct{}),
	}
	go n.generate()
	return n
}

type nonceImpl struct {
	lock      sync.Mutex
	nextNonce chan Nonce
	nonceLen  int
	maxUsed   int
	nonces    map[Nonce]struct{}
}

func (n *nonceImpl) generate() {
	for {
		n.nextNonce <- generateRandomNonce()
	}
}
func (n *nonceImpl) Valid(nonce Nonce) bool {
	n.lock.Lock()
	defer n.lock.Unlock()
	if _, ok := n.nonces[nonce]; ok {
		delete(n.nonces, nonce)
		return true
	}
	return false
}

func (n *nonceImpl) Next() Nonce {
	if len(n.nonces) > MaxUsed {
		//TODO: DO SOMETHING
		n.nonces = make(map[Nonce]struct{})
	}
	nonce := <-n.nextNonce
	n.lock.Lock()
	n.nonces[nonce] = struct{}{}
	n.lock.Unlock()
	return nonce
}

func generateRandomNonce() Nonce {
	b := make([]byte, nonceLen)
	_, err := io.ReadFull(rand.Reader, b)
	if err != nil {
		panic(fmt.Sprintf("Error reading random bytes: %s", err))
	}
	return Nonce(base64.RawURLEncoding.EncodeToString(b))
}
