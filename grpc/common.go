/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package grpc

import (
	"brick/core/berrors"
	"brick/core/log"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GrpcOptions struct {
	logger   logrus.FieldLogger
	grpcAddr string
	keyId    string
	tls      bool
	tlsConf  *tls.Config
}

//GrpcCredentialsInsecure are used for passing to NewXXXWrapper for No TLS
var GrpcCredentialsInsecure []grpc.DialOption = []grpc.DialOption{grpc.WithInsecure()} //grpc.WithInsecure()
//GrpcCredentialsServerAuth are not used yet :)
var GrpcCredentialsServerAuth []grpc.DialOption = []grpc.DialOption{}

var callOptions = []grpc.CallOption{grpc.FailFast(false)}

func (s *storage) handleError(ctx context.Context, err error) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "HandleError")
	status := status.Code(err)
	if status == codes.DeadlineExceeded {
		return berrors.TimeoutError()
	} else if status == codes.NotFound {
		return berrors.NotFoundError("The object does not exist")
	} else {
		span.SetTag("error", true)
		span.LogKV("event", "error", "error.object", err)
		log.Error(ctx, err, s.logger)
		return berrors.UnknownError(err)
	}
}

func handleError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	span, ctx := opentracing.StartSpanFromContext(ctx, "HandleError")
	span.SetTag("error", true)
	span.LogKV("event", "error", "error.object", err)
	status := status.Convert(err)
	if status.Code() == codes.DeadlineExceeded {
		return berrors.TimeoutError()
	} else if status.Code() == codes.NotFound {
		return berrors.NotFoundError(status.Message())
	} else {
		return berrors.UnknownError(err)
	}
}

func customVerify(rootcertPool *x509.CertPool, allowedCNs []string) func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	return func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
		//verified := false
		certs := make([]*x509.Certificate, 0)
		for _, rawCert := range rawCerts {
			cert, err := x509.ParseCertificate(rawCert)
			if err != nil {
				return err
			}
			certs = append(certs, cert)
		}
		intermediatePool := x509.NewCertPool()
		for _, intermediate := range certs[1 : len(certs)-1] {
			intermediatePool.AddCert(intermediate)
		}
		verified := true
		for _, allowedCN := range allowedCNs {
			_, err := certs[0].Verify(x509.VerifyOptions{
				DNSName:       allowedCN,
				Intermediates: intermediatePool,
				Roots:         rootcertPool,
				KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			})
			if err == nil {
				verified = true
				break
			}
			fmt.Println(err)
		}
		if !verified {
			return errors.New("Verification Failure")
		}
		return nil
	}
}
