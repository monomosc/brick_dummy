package config

//BrickValidationConfig represents the json-options in a passed --config flag for BrickWeb
type BrickValidationConfig struct {
	JSONLogging    bool              `json:"json_logging"`
	Opentracing    OpentracingConfig `json:"opentracing"`
	ProcessTracing bool              `json:"process_tracing"`
	Storage        struct {
		Address string `json:"address"`
	}
	Port        int           `json:"port"`
	File        FileLogConfig `json:"file_logging"`
	GrpcLogging bool          `json:"grpc_logging"`
	Sleep       bool          `json:"sleep"` //Whether to sleep before attempting validation
	Stage       string        `json:"stage"`
	TLSConfig   GrpcTlsConfig `json:"tls_config"`
}
