/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package policy_test

import (
	"brick/brickweb/policy"
	"testing"
)

func Test_JWKAlgos(t *testing.T) {
	algos := policy.GetAllowedJWSAlgorithms()
	ok := func() bool {
		for _, a := range algos {
			if a == "ES256" {
				return true
			}
		}
		return false
	}()
	if !ok {
		t.Error("ES256 should be supported but is not (See RFC Chapter 6.2)")
	}
}
