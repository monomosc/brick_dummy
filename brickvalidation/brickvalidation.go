package main

import (
	"brick/brickvalidation/va"
	"brick/config"
	"brick/core"
	"brick/core/log"
	brickgrpc "brick/grpc"
	"brick/grpc/vapb"
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
)

var (
	configFile = flag.String("config", "./run/default-config.json", "The Config File (JSON)")
)

func main() {
	flag.Parse()
	var c = config.BrickValidationConfig{}
	err := config.ReadJSON(*configFile, &c)
	if err != nil {
		panic(err)
	}
	verifyConfig(&c)
	config.SetWriter(c.File)
	config.SetStage(c.Stage)
	mainLogger := config.MakeStandardLogger("brickvalidation", "brickvalidation", true)
	mainLogger.Info("Starting up")
	if c.GrpcLogging {
		grpclog.SetLoggerV2(grpclog.NewLoggerV2WithVerbosity(os.Stdout, os.Stdout, os.Stdout, 4))
	}

	//Check if we should do runtime tracing
	if c.ProcessTracing {
		mainLogger.Warn("Doing Process-Level Tracing - DISABLE IF NOT TESTING")
		c := config.StartProcessTracing()
		c.Close()
	}

	//Check if we should do openTracing
	closer, err := config.SetupOpenTracing(c.Opentracing, "brickvalidation", config.MakeStandardLogger("brickvalidation", "opentracing", false), prometheus.DefaultRegisterer)
	if err != nil {
		panic(err)
	}
	defer closer.Close()

	storageCredentials := brickgrpc.GrpcCredentialsInsecure

	var tlsConf *tls.Config
	if c.TLSConfig.Enable {
		confs := c.TLSConfig.GenerateTLSConfigs()
		storageCredentials = []grpc.DialOption{grpc.WithTransportCredentials(credentials.NewTLS(confs["brickstorage"]))}
		tlsConf = confs["brickweb"]
	}
	//StorageWrapper
	storage, err := brickgrpc.NewStorageWrapper(c.Storage.Address, config.MakeStandardLogger("brickvalidation", "storage", c.JSONLogging), storageCredentials)
	if err != nil {
		panic(err)
	}
	verificationChan := make(chan core.VerificationRequest, 10)
	verificationAuthority := va.New(verificationChan, config.MakeStandardLogger("brickvalidation", "va", c.JSONLogging), storage)

	go verificationAuthority.Start()

	impl := va.NewValidationGrpcService(verificationChan, config.MakeStandardLogger("brickvalidation", "vagrpc", c.JSONLogging))

	//start server
	srv := grpc.NewServer()
	vapb.RegisterBrickValidationServer(srv, impl)
	var listener net.Listener
	if c.TLSConfig.Enable {
		listener, err = tls.Listen("tcp", fmt.Sprintf(":%d", c.Port), tlsConf)
		if err != nil {
			panic(err)
		}
	} else {
		listener, err = net.Listen("tcp", fmt.Sprintf(":%d", c.Port))
		if err != nil {
			panic(err)
		}
	}

	if err != nil {
		mainLogger.Panic(err)
	}

	//"Gracefully" exit on ctrl-c or docker stop
	//
	go catchSIGINT(srv)
	mainLogger.WithField("addr", fmt.Sprintf(":%d", c.Port)).Info("Starting Validation Server")
	err = srv.Serve(listener)

	mainLogger.WithError(err).Warning("Exit")
}

func catchSIGINT(srv *grpc.Server) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	for range c {
		fmt.Println("SIGINT caught. Shutting down validation authority ....")
		srv.GracefulStop()
	}
}

func verifyConfig(c *config.BrickValidationConfig) {
	if c.Port <= 0 {
		c.Port = 4242
	}
}

func withTracingInterceptor() grpc.ServerOption {
	return grpc.UnaryInterceptor(traceInterceptor)
}

func traceInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.InvalidArgument, "Retrieving Metadata failed")
	}
	traceHeader, ok := md["Datev-Trace-ID"]
	if !ok {
		return nil, status.Errorf(codes.InvalidArgument, "Missing Datev-Trace-ID")
	}
	ctx = log.SetTraceID(ctx, traceHeader[0])
	h, err := handler(ctx, req)
	return h, err
}
