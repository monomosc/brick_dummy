/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package config

//BrickRevokedConfig is the config struct for brickrevoked
type BrickRevokedConfig struct {
	JSONLogging    bool              `json:"json_logging"`
	Opentracing    OpentracingConfig `json:"opentracing"`
	ProcessTracing bool              `json:"process_tracing"`
	Port           int               `json:"port"`
	Storage        struct {
		Address string `json:"address"`
	}
	CA struct {
		Address string `json:"address"`
	}
}
