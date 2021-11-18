package tracing

import (
	"brick/core/log"
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TraceIDUnaryClientInterceptor(traceIDHeaderName string, LogStart, LogEnd func(innerCtx context.Context, method string, err error)) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		traceID := log.GetTraceID(ctx)
		LogStart(ctx, method, nil)
		ctx = metadata.AppendToOutgoingContext(ctx, traceIDHeaderName, traceID)
		err := invoker(ctx, method, req, reply, cc, opts...)
		LogEnd(ctx, method, err)
		return err
	}
}

func TraceIDStreamClientInterceptor(traceIDHeaderName string, LogStart func(innerCtx context.Context, method string, err error)) grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		traceID := log.GetTraceID(ctx)
		LogStart(ctx, method, nil)
		ctx = metadata.AppendToOutgoingContext(ctx, traceIDHeaderName, traceID)
		stream, err := streamer(ctx, desc, cc, method, opts...)
		return stream, err
	}
}
