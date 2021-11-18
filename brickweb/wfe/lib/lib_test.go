/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package lib_test

import (
	"brick/brickweb/wfe/lib"
	"brick/brickweb/wfe/token"
	"context"
	"testing"
)

func TestAtLeastHTTPChallenge(t *testing.T) {
	chalZ := lib.CreateDefaultChallenges(context.TODO(), token.New())
	ok := func() bool {
		for _, chal := range chalZ {
			if chal.Type == "http-01" {
				return true
			}
		}
		return false
	}()
	if !ok {
		t.Error("No Http Challenge has been created by defaultchalz")
	}
}
