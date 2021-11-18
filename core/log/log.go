package log

import (
	"brick/core/berrors"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"path"
	"runtime"
	"runtime/debug"

	"github.com/sirupsen/logrus"
)

func Error(ctx context.Context, err error, logger logrus.FieldLogger) {
	_, fileName, fileLine, ok := runtime.Caller(3)
	var s string
	if ok {
		s = fmt.Sprintf("%s:%d", path.Base(fileName), fileLine)
	} else {
		s = "Error determining source File"
	}
	message := err.Error()
	if _, ok := berrors.IsTimeoutError(err); ok {
		message = "Timeout"
	}
	WithTraceID(logger.WithError(err).WithField("stack", string(debug.Stack())).WithField("file", s), ctx).Error(message)
}

type DatevTraceID string
type TraceIdType string
type FieldType string

func GenerateDatevTraceID() string {
	data := make([]byte, 10)
	_, err := rand.Read(data)
	if err != nil {
		return "ERROR_CREATING_TRACE_ID"
	}
	return hex.EncodeToString(data)
}

func EnsureTraceID(ctx context.Context) context.Context {
	val := ctx.Value(TraceIdType("tid"))
	if val == nil {
		traceID := GenerateDatevTraceID()
		return context.WithValue(ctx, TraceIdType("tid"), DatevTraceID(traceID))
	}
	return ctx
}

func WithTraceID(logger logrus.FieldLogger, ctx context.Context) logrus.FieldLogger {
	var l = logger
	val := ctx.Value(TraceIdType("tid"))
	if val != nil {
		l = l.WithField("Datev_Trace", string(val.(DatevTraceID)))
	}
	val = ctx.Value(FieldType("RequestPath"))
	if val != nil {
		l = l.WithField("RequestPath", string(val.(string)))
	}
	return l
}

func WithField(ctx context.Context, key, value string) context.Context {
	return context.WithValue(ctx, FieldType(key), value)
}

func GetTraceID(ctx context.Context) string {
	val := ctx.Value(TraceIdType("tid"))
	if val != nil {
		return string(val.(DatevTraceID))
	}
	return ""
}

func SetTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, TraceIdType("tid"), DatevTraceID(traceID))
}
