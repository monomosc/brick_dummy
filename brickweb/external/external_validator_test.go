/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package external_test

import (
	"brick/brickweb/external"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestCorrectValidator(t *testing.T) {
	Convey("When requesting a symmetric_token validator", t, func() {
		So(func() {
			external.GetValidator(external.SymmetricToken, map[string]interface{}{"key": "123456789123456789"})
		}, ShouldNotPanic)
	})
}
