package config

import (
	"io"
	"io/ioutil"
	"os"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	jaeger "github.com/uber/jaeger-client-go"
	jaegerconf "github.com/uber/jaeger-client-go/config"
	jaegerprom "github.com/uber/jaeger-lib/metrics/prometheus"
)

//SetupOpenTracing sets up Opentracing (enabled with jaeger, or disabled with NoopTracer)
func SetupOpenTracing(c OpentracingConfig, serviceName string, logger logrus.FieldLogger, reg prometheus.Registerer) (io.Closer, error) {
	if !c.Enable {
		opentracing.SetGlobalTracer(opentracing.NoopTracer{})
		return ioutil.NopCloser(os.Stdin), nil // ugliest hack in the history of hacks
	}
	conf, err := jaegerconf.FromEnv()
	if err != nil {
		panic(err)
	}
	logger.Info("Enabling Opentracing Jaeger Instrumentation")
	samplerConfig := conf.Sampler
	samplerConfig.Type = "const"
	samplerConfig.Param = 1.0
	metricsFac := jaeger.NewMetrics(jaegerprom.New(), map[string]string{})
	sampler, err := samplerConfig.NewSampler(serviceName, metricsFac)
	if err != nil {
		panic(err)
	}
	remoteReporterConf := conf.Reporter
	remoteReporterConf.LogSpans = false
	remoteReporter, err := remoteReporterConf.NewReporter(serviceName, metricsFac, jaeger.NullLogger)
	if err != nil {
		panic(err)
	}
	tracer, closer := jaeger.NewTracer(serviceName, sampler, remoteReporter, jaeger.TracerOptions.Metrics(metricsFac)) //Zipkin Headers
	opentracing.SetGlobalTracer(tracer)
	return closer, nil
}

type jaegerLogger struct {
	logger logrus.FieldLogger
}

func (l *jaegerLogger) Error(msg string) {
	l.logger.Error(msg)
}
func (l *jaegerLogger) Infof(msg string, args ...interface{}) {
	l.logger.Infof(msg, args...)
}

//spanLoggingReporter implements jaeger.Reporter
type spanLoggingReporter struct {
	logger logrus.FieldLogger
}

func (r *spanLoggingReporter) Report(span *jaeger.Span) {
	r.logger.Infof("Finished operation %s", span.OperationName())
}

func (r *spanLoggingReporter) Close() {
	//NOOP
}
