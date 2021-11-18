/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package config

//BrickWebConfig represents the json-options in a passed --config flag for BrickWeb
type BrickWebConfig struct {
	JSONLogging    bool              `json:"json_logging"`
	Opentracing    OpentracingConfig `json:"opentracing"`
	ProcessTracing bool              `json:"process_tracing"`
	CA             struct {
		Address string `json:"address"`
	}
	Web struct {
		TLS      bool   `json:"tls_enabled"`
		KeyFile  string `json:"key"`
		CertFile string `json:"cert"`
	}
	Storage struct {
		Address string `json:"address"`
	}
	Nonce               NonceConfig           `json:"nonce"`
	AccountCreation     AccountCreationConfig `json:"account_creation"`
	ProhibitGetRequests bool                  `json:"prohibit_get"`
	BaseURL             string                `json:"base_url"`
	Port                int                   `json:"port"`
	VA                  struct {
		Address string `json:"address"`
	}
	GrpcLogging     bool          `json:"grpc_logging"`
	File            FileLogConfig `json:"file_logging"`
	Stage           string        `json:"stage"`
	TLSConfig       GrpcTlsConfig `json:"tls_config"`
	WaitForIssuance bool          `json:"wait_for_issuance"`
}

type AccountCreationConfig struct {
	RequireExternal bool                   `json:"require_external"`
	RedirectURI     string                 `json:"redirect_uri"`
	ValidatorName   string                 `json:"validator_name"`
	ValidatorConfig map[string]interface{} `json:"validator_config"`
}

type NonceConfig struct {
	Provider  string `json:"provider"`
	RedisAddr string `json:"redis_address"`
}
