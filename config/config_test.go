/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package config_test

import (
	"brick/config"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestBrickConfig(T *testing.T) {
	Convey("When a Config containing some values is loaded", T, func() {
		var c config.BrickWebConfig
		err := config.ReadJSON("./testConfig.json", &c)
		if err != nil {
			So(err, ShouldBeNil)
		}
		Convey("The values should match config values", func() {
			So(c.CA.Address, ShouldEqual, "consul-http://localhost:8500/brickca")
		})
	})
}
