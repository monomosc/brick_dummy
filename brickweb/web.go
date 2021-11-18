/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package main

import (
	"brick/brickweb/external"
	"brick/brickweb/wfe/nonce"
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"

	"google.golang.org/grpc/credentials"

	"google.golang.org/grpc/grpclog"

	"brick/brickweb/wfe"
	"brick/config"
	"brick/core"
	"brick/grpc"
	_ "brick/grpc/resolvers/consul"
	_ "brick/grpc/resolvers/static"

	"github.com/prometheus/client_golang/prometheus"
	googlegrpc "google.golang.org/grpc"
)

var (
	configFile = flag.String("config", "./run/default-config.json", "The Config File (JSON)")
)

func main() {
	flag.Parse()
	var c = config.BrickWebConfig{}
	err := config.ReadJSON(*configFile, &c)
	if err != nil {
		panic(err)
	}

	verifyConfig(&c)
	config.SetWriter(c.File)
	config.SetStage(c.Stage)

	mainLogger := config.MakeStandardLogger("brickweb", "brickweb", true)
	mainLogger.Info("Starting up")
	if c.GrpcLogging {
		grpclog.SetLoggerV2(grpclog.NewLoggerV2WithVerbosity(os.Stdout, os.Stdout, os.Stdout, 3))
	}

	//Check if we should do runtime tracing
	if c.ProcessTracing {
		mainLogger.Warn("Doing Process-Level Tracing - DISABLE IF NOT TESTING")
		c := config.StartProcessTracing()
		c.Close()
	}
	//Check if we should do openTracing
	closer, err := config.SetupOpenTracing(c.Opentracing, "brickweb", config.MakeStandardLogger("brickweb", "opentracing", false), prometheus.DefaultRegisterer)
	if err != nil {
		panic(err)
	}
	defer closer.Close()

	caCredentials := grpc.GrpcCredentialsInsecure
	validationCredentials := grpc.GrpcCredentialsInsecure
	storageCredentials := grpc.GrpcCredentialsInsecure

	if c.TLSConfig.Enable {
		confs := c.TLSConfig.GenerateTLSConfigs()
		caCredentials = []googlegrpc.DialOption{googlegrpc.WithTransportCredentials(credentials.NewTLS(confs["brickca"]))}
		validationCredentials = []googlegrpc.DialOption{googlegrpc.WithTransportCredentials(credentials.NewTLS(confs["brickvalidation"]))}
		storageCredentials = []googlegrpc.DialOption{googlegrpc.WithTransportCredentials(credentials.NewTLS(confs["brickstorage"]))}
	}
	//CaWrapper
	ca, err := grpc.NewCAWrapper(c.CA.Address, config.MakeStandardLogger("brickweb", "ca", c.JSONLogging), caCredentials)
	if err != nil {
		panic(err)
	}
	storage, err := grpc.NewStorageWrapper(c.Storage.Address, config.MakeStandardLogger("brickweb", "storage", c.JSONLogging), storageCredentials)
	if err != nil {
		panic(err)
	}
	validationAuthority, err := grpc.NewValidationWrapper(c.VA.Address, config.MakeStandardLogger("brickweb", "va", c.JSONLogging), validationCredentials)
	if err != nil {
		panic(err)
	}
	wfe := wfe.New(config.MakeStandardLogger("brickweb", "wfe", c.JSONLogging), ca, storage, validationAuthority)

	//Initialize Nonce Subsystem
	switch c.Nonce.Provider {
	case "redis":
		mainLogger.WithField("address", c.Nonce.RedisAddr).Infof("Using Redis Provider")
		noncer := nonce.NewRedisNoncer(c.Nonce.RedisAddr)
		wfe.Noncer = noncer
	case "none":
		mainLogger.Info("Using None Nonce Provider")
		wfe.Noncer = nonce.NewNoneNoncer()
	default:
		wfe.Noncer = nonce.NewNoncer()
	}
	//Initialize HTTP Server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", c.Port),
		Handler: wfe.Handler(),
	}
	if c.AccountCreation.RequireExternal {
		wfe.AccountValidator = external.GetValidator(c.AccountCreation.ValidatorName, c.AccountCreation.ValidatorConfig)
	} else {
		wfe.AccountValidator = nil
	}
	wfe.BasePath = c.BaseURL
	wfe.WaitForIssuanceOnFinalize = c.WaitForIssuance

	//"Gracefully" exit on ctrl-c or docker stop
	go catchSIGINT(srv)
	//start server
	mainLogger.Infof("Starting listening on %s", srv.Addr)
	if c.Web.TLS {
		mainLogger.Warn("Running with TLS enabled is not recommended; try using a reverse proxy instead")
		srv.Addr = ":443"
		err = srv.ListenAndServeTLS(c.Web.CertFile, c.Web.KeyFile)
	} else {
		err = srv.ListenAndServe()
	}
	mainLogger.WithError(err).Warning("Exit")
}

func catchSIGINT(srv *http.Server) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	for range c {
		srv.Shutdown(context.Background())
	}
}

type cacertLister interface {
	GetAvailableCaCertificates(rootCtx context.Context) ([]*core.CaCertificate, error)
}

func verifyConfig(c *config.BrickWebConfig) {
	if c.BaseURL == "" {
		panic("BrickWeb BasePath cannot be unset (base_url)")
	}
	if c.Port == 0 {
		c.Port = 80
	}
}
