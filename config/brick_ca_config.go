/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package config

//BrickCAConfig represents the Json-options for BrickCA
type BrickCAConfig struct {
	JSONLogging    bool              `json:"json_logging"`
	Opentracing    OpentracingConfig `json:"opentracing"`
	ProcessTracing bool              `json:"process_tracing"`
	SerialPrefix   uint8             `json:"serial_prefix"`
	Storage        struct {
		Address string `json:"address"`
	}
	Port              int                    `json:"port"`
	CAProvider        string                 `json:"ca_provider"`
	CAProviderOptions map[string]interface{} `json:"ca_provider_config"`
	Policy            CAPolicy               `json:"policy"`
	Prometheus        PrometheusConfig       `json:"prometheus"`
	BasePath          string                 `json:"base_path"`
}

type CAPolicy struct {
	Whitelist    CAConstraints `json:"whitelist"`
	Blacklist    CAConstraints `json:"blacklist"`
	ValidityDays int           `json:"validity_days"`
}

type CAConstraints struct {
	DomainSuffixes []string
}
