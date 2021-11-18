/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package token_test

import (
	"brick/brickweb/wfe/token"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

var tok string

func Test_UnequalConsecutive(t *testing.T) {
	Convey("Given a standard token.Tokenizer", t, func() {
		t := token.New()
		Convey("Consecutive Tokens should be unequal", func() {
			t1 := t.NewToken()
			t2 := t.NewToken()
			So(t1, ShouldNotEqual, t2)
		})
	})
}

func BenchmarkBunchOfTokens(b *testing.B) {
	t := token.New()
	var y string
	var x string
	for i := 0; i < b.N; i++ {
		x = t.NewToken()
	}
	y = x
	tok = y
}
