using System;
using Microsoft.Extensions.Logging;
using ILogger = Grpc.Core.Logging.ILogger;

namespace Brick.Grpc
{
    public class GrpcAdapterLogger : ILogger
    {
        public ILoggerFactory Fac { get; }
        private Microsoft.Extensions.Logging.ILogger Logger { get; }

        public GrpcAdapterLogger(ILoggerFactory fac)
        {
            Fac = fac;
            Logger = fac.CreateLogger(typeof(global::Grpc.Core.GrpcEnvironment));
        }

        private GrpcAdapterLogger(ILoggerFactory fac, Microsoft.Extensions.Logging.ILogger l)
        {
            Logger = l;
            Fac = fac;
        }

        public ILogger ForType<T>()
        {
            return new GrpcAdapterLogger(Fac, Fac.CreateLogger(typeof(T)));
        }

        public void Debug(string message)
        {
            Logger.LogDebug(message);
        }

        public void Debug(string format, params object[] formatArgs)
        {
            Logger.LogDebug(format, formatArgs);
        }

        public void Info(string message)
        {
            Logger.LogInformation(message);
        }

        public void Info(string format, params object[] formatArgs)
        {
            Logger.LogInformation(format, formatArgs);
        }

        public void Warning(string message)
        {
            Logger.LogWarning(message);
        }

        public void Warning(string format, params object[] formatArgs)
        {
            Logger.LogWarning(format, formatArgs);
        }

        public void Warning(Exception exception, string message)
        {
            Logger.LogWarning(exception, message);
        }

        public void Error(string message)
        {
            Logger.LogError(message);
        }

        public void Error(string format, params object[] formatArgs)
        {
            Logger.LogError(format, formatArgs);
        }

        public void Error(Exception exception, string message)
        {
            Logger.LogError(exception, message);
        }
    }
}
