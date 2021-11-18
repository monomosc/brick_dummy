package grpc

import (
	"brick/core"
	"brick/core/log"
	"brick/grpc/tracing"
	"brick/grpc/vapb"
	"context"
	"time"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type validation struct {
	client vapb.BrickValidationClient
	logger logrus.FieldLogger
}

//NewValidationWrapper todo.
func NewValidationWrapper(grpcAddr string, logger logrus.FieldLogger, grpcCredentials []grpc.DialOption) (*validation, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	LogStart := func(innerCtx context.Context, method string, e error) {
		if e != nil {
			log.WithTraceID(logger, innerCtx).WithError(e).WithField("grpc_method", method).Debugf("Executing Grpc Method %s", method)
		} else {
			log.WithTraceID(logger, innerCtx).WithField("grpc_method", method).Debugf("Executing Grpc Method %s", method)
		}
	}
	LogEnd := func(innerCtx context.Context, method string, e error) {
		if e != nil {
			log.WithTraceID(logger, innerCtx).WithError(e).WithField("grpc_method", method).Debugf("Finished Grpc Method %s", method)
		} else {
			log.WithTraceID(logger, innerCtx).WithField("grpc_method", method).Debugf("Finished Grpc Method %s", method)
		}
	}
	conn, err := grpc.DialContext(ctx, grpcAddr,
		append(grpcCredentials, grpc.WithStreamInterceptor(tracing.TraceIDStreamClientInterceptor("Datev-Trace-ID", LogStart)),
			grpc.WithUnaryInterceptor(tracing.TraceIDUnaryClientInterceptor("Datev-Trace-ID", LogStart, LogEnd)))...)

	if err != nil {
		return nil, err
	}
	logger.Infof("Connecting to %s", grpcAddr)
	client := vapb.NewBrickValidationClient(conn)
	return &validation{
		client: client,
		logger: logger,
	}, nil
}

func (v *validation) DoValidation(ctx context.Context, valRequest *core.VerificationRequest) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "DoValidation")
	defer span.Finish()
	msg, err := ValidationToProto(ctx, valRequest)
	if err != nil {
		return err
	}
	v.logger.Debug("brickvalidate: DoValidation")
	_, err = v.client.DoValidate(ctx, msg, callOptions...)

	return handleError(ctx, err)
}
