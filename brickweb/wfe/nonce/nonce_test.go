/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package nonce_test

import (
	"brick/brickweb/wfe/nonce"
	"sync"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestCreateABunchOfNonces(t *testing.T) {
	nonceSvc := nonce.NewNoncer()
	size := 300
	nonces := make([]nonce.Nonce, size)
	for i := 0; i < size; i++ {
		nonces[i] = nonceSvc.Next()
	}
	//All nonces must be valid
	for i := 0; i < size; i++ {
		if !nonceSvc.Valid(nonces[i]) {
			t.Fail()
		}
	}
	//Nonce of those nonces can still be valid
	for i := 0; i < size; i++ {
		if nonceSvc.Valid(nonces[i]) {
			t.Fail()
		}
	}
}

func TestCreateParallelNonces(t *testing.T) {
	noncer := nonce.NewNoncer()
	nonces := make([]nonce.Nonce, 50000)
	wg := sync.WaitGroup{}
	wg.Add(5)
	for i := 0; i < 5; i++ {
		go func(count int) {
			for j := 0; j < 10000; j++ {
				nonces[count*10000+j] = noncer.Next()
				ok := noncer.Valid(nonces[count*10000+j])
				if !ok {
					t.Errorf("Nonce #%d not valid: %s", count*10000+j, nonces[count*10000+j])
					t.Fail()
				}
			}
			wg.Done()
		}(i)
	}
	wg.Wait()
	nonceMap := make(map[nonce.Nonce]time.Time)
	i := 0
	for _, n := range nonces {
		i++
		_, ok := nonceMap[n]
		if ok {
			t.Errorf("%s Nonce twice #%d", string(n), i)
		}
		nonceMap[n] = time.Now()
	}
}

func TestNonceTwice(t *testing.T) {
	noncer := nonce.NewNoncer()
	Convey("When a Nonce is created", t, func() {
		nonce := noncer.Next()
		Convey("It should be valid once", func() {
			So(noncer.Valid(nonce), ShouldBeTrue)
			Convey("But not twice", func() {
				So(noncer.Valid(nonce), ShouldBeFalse)
			})
		})
	})
}
