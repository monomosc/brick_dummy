using System;
using System.Linq;
using System.Security.Cryptography.X509Certificates;
using Grpc.Core;
using X509Toolbox;
using X509Toolbox.ExtensionMethods;

namespace Brick.Grpc
{
    public static class VerifyPeerCallbackFactory
    {
        public static readonly Func<string, VerifyPeerCallback> VerifyPeer =
            requiredServiceName => context =>
                new X509Certificate2(Convert.FromBase64String(context.PeerPem
                    .Replace("-----BEGIN CERTIFICATE-----", "")
                    .Replace("-----END CERTIFICATE-----", "")))
                    .GetSubjectAsRdnSequence().Content
                    .Where(rdn => rdn.Oid == WellKnownOids.Rdn.CommonName)
                    .Any(rdn => rdn.Value == requiredServiceName);
    }
}
