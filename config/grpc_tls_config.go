package config

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/ioutil"
)

type GrpcTlsConfig struct {
	Enable        bool     `json:"enable"`
	RootLocation  string   `json:"root_location"`
	ChainLocation string   `json:"chain_location"`
	KeyLocation   string   `json:"key_location"`
	AllowedPeers  []string `json:"allowed_peers"`
}

func (c GrpcTlsConfig) GenerateTLSConfigs() map[string]*tls.Config {
	if !c.Enable {
		panic(errors.New("cannot Generate TLS Confg if TLS is not enabled"))
	}
	rootPool := x509.NewCertPool()
	rootcert, err := ioutil.ReadFile(c.RootLocation)
	if err != nil {
		panic(err)
	}
	rootPool.AppendCertsFromPEM(rootcert)
	cert, err := tls.LoadX509KeyPair(c.ChainLocation, c.KeyLocation)
	if err != nil {
		panic(err)
	}
	var ret = make(map[string]*tls.Config)
	for _, serviceName := range c.AllowedPeers {
		var conf = &tls.Config{
			RootCAs:            rootPool,
			InsecureSkipVerify: false,
			ClientCAs:          rootPool,
			ClientAuth:         tls.RequireAndVerifyClientCert,
			Certificates:       []tls.Certificate{cert},
			ServerName:         serviceName,
		}
		ret[serviceName] = conf
	}
	return ret
}
