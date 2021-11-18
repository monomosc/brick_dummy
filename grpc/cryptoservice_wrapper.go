/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package grpc

import (
	"brick/grpc/cryptoservice"
	"context"
	"crypto/tls"
	"crypto/x509"
	"time"

	"google.golang.org/grpc/credentials"

	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"

	"google.golang.org/grpc"
)

type cryptoS struct {
	client cryptoservice.CryptoGearClient
	logger logrus.FieldLogger
	keyId  string
}
type cryptoSOptions struct {
	GrpcOptions
}

type CryptoServiceOption func(*cryptoSOptions)

var WithKeyId = func(kid string) CryptoServiceOption {
	return func(o *cryptoSOptions) {
		o.keyId = kid
	}
}
var WithLogger = func(logger logrus.FieldLogger) CryptoServiceOption {
	return func(o *cryptoSOptions) {
		o.logger = logger
	}
}
var WithGrpcAddr = func(addr string) CryptoServiceOption {
	return func(o *cryptoSOptions) {
		o.grpcAddr = addr
	}
}
var WithoutTLS = func() CryptoServiceOption {
	return func(o *cryptoSOptions) {
		o.tls = false
	}
}
var WithTLS = func(clientcert tls.Certificate, rootcert *x509.Certificate, allowedCNs []string) CryptoServiceOption {
	rootcertPool := x509.NewCertPool()
	rootcertPool.AddCert(rootcert)
	return func(o *cryptoSOptions) {
		o.tls = true
		if o.tlsConf == nil {
			o.tlsConf = &tls.Config{}
		}
		o.tlsConf.Certificates = []tls.Certificate{clientcert}
		o.tlsConf.InsecureSkipVerify = true
		o.tlsConf.VerifyPeerCertificate = customVerify(rootcertPool, allowedCNs)

	}
}

//NewStorageWrapper returns a Grpc Client Wrapper intending to implement the Storage interface of different subcomponents
//,e.g. VA, CA, WFE, ...
func NewCryptoserviceWrapper(opts ...CryptoServiceOption) (*cryptoS, error) {
	o := &cryptoSOptions{}
	for _, f := range opts {
		f(o)
	}
	logger := o.logger
	if logger == nil {
		logger = logrus.New()
	}
	logger.Infof("Connecting to %s", o.grpcAddr)
	dialOptions := []grpc.DialOption{grpc.WithTimeout(time.Second * 5), grpc.WithStreamInterceptor(otgrpc.OpenTracingStreamClientInterceptor(opentracing.GlobalTracer())),
		grpc.WithUnaryInterceptor(otgrpc.OpenTracingClientInterceptor(opentracing.GlobalTracer()))}
	if o.tls {
		creds := credentials.NewTLS(o.tlsConf)
		dialOptions = append(dialOptions, grpc.WithTransportCredentials(creds))
	} else {
		dialOptions = append(dialOptions, grpc.WithInsecure())
	}
	conn, err := grpc.Dial(o.grpcAddr, dialOptions...)
	if err != nil {
		return nil, err
	}
	logger.WithField("keyId", o.keyId).Infof("Cryptogear using KeyID")
	client := cryptoservice.NewCryptoGearClient(conn)
	return &cryptoS{
		client: client,
		logger: logger,
		keyId:  o.keyId,
	}, nil
}

func (s *cryptoS) Sign(ctx context.Context, data []byte) ([]byte, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "Cryptoservice.Sign")
	defer span.Finish()
	return s.SignWithKeyLabel(ctx, data, s.keyId, "SHA256")
}

func (s *cryptoS) SignWithKeyLabel(ctx context.Context, data []byte, keyLabel string, HashAlgorithm string) ([]byte, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "SignWithKeyLabel")
	defer span.Finish()
	span.SetTag("keyLabel", keyLabel)
	span.SetTag("data.size", len(data))
	span.SetTag("HashAlgorithm", HashAlgorithm)
	sig, err := s.client.CreateSignature(ctx, &cryptoservice.DataToSign{
		KeyLabel:      keyLabel,
		RawData:       data,
		HashAlgorithm: HashAlgorithm,
		Padding:       cryptoservice.PaddingType_Pkcs,
	})
	if err != nil {
		return nil, handleError(ctx, err)
	}
	return sig.RawData, nil
}
