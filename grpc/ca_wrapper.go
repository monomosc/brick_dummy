/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package grpc

import (
	"brick/grpc/tracing"
	"context"
	"crypto/x509"
	"fmt"
	"time"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"brick/core"
	"brick/core/berrors"
	"brick/core/log"
	"brick/grpc/capb"
	proto "brick/grpc/corepb"
)

type certificateAuthority struct {
	client capb.CertificateAuthorityClient
	logger logrus.FieldLogger
	CNToID map[string]string
}

//NewCAWrapper returns a Grpc Wrapper to the CA-Interface required by other components
//like WFE
func NewCAWrapper(grpcAddr string, logger logrus.FieldLogger, grpcCredentials []grpc.DialOption) (*certificateAuthority, error) {
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
	conn, err := grpc.Dial(grpcAddr,
		append(grpcCredentials, grpc.WithStreamInterceptor(tracing.TraceIDStreamClientInterceptor("Datev-Trace-ID", LogStart)),
			grpc.WithUnaryInterceptor(tracing.TraceIDUnaryClientInterceptor("Datev-Trace-ID", LogStart, LogEnd)))...)

	if err != nil {
		return nil, err
	}
	logger.Infof("Connecting to %s", grpcAddr)
	client := capb.NewCertificateAuthorityClient(conn)

	var certAuth = new(certificateAuthority)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*9)
		span, ctx := opentracing.StartSpanFromContext(ctx, "GetCaCertificates")
		defer span.Finish()
		certList, err := client.GetCaCertificates(ctx, &proto.Empty{}, grpc.FailFast(false))
		if err != nil {
			panic(err)
		}
		cnToIDMap := make(map[string]string)
		for _, cacert := range certList.CaCerts {
			if cacert.WillIssue {
				cnToIDMap[cacert.CommonName] = cacert.Id
			}
		}
		certAuth.CNToID = cnToIDMap
		cancel()
	}()
	certAuth.client = client
	certAuth.logger = logger
	return certAuth, nil
}

func (ca *certificateAuthority) CompleteOrder(rootCtx context.Context, order *core.Order, parsedCSR *x509.CertificateRequest) error {
	span, ctx := opentracing.StartSpanFromContext(rootCtx, "CompleteOrder")
	defer span.Finish()

	span.SetTag("id", order.ID)
	protoOrder, err := orderToProto(ctx, order, nil)
	_, err = ca.client.CompleteOrder(ctx, &capb.CompleteOrderRequest{
		Csr:   parsedCSR.Raw,
		Order: protoOrder,
	}, callOptions...)
	if err != nil {
		return handleError(ctx, err)
	}
	return nil
}

func (ca *certificateAuthority) GetAvailableCertificates(rootCtx context.Context) ([]*core.CaCertificate, error) {
	span, ctx := opentracing.StartSpanFromContext(rootCtx, "GetAvailableCertificates")
	defer span.Finish()
	cacertList, err := ca.client.GetCaCertificates(ctx, &proto.Empty{}, callOptions...)
	if err != nil {
		return nil, handleError(ctx, err)
	}
	cacerts := make([]*core.CaCertificate, len(cacertList.CaCerts))
	for i, c := range cacertList.CaCerts {
		cacert, err := protoToCaCert(ctx, c)
		if err != nil {
			return nil, handleError(ctx, err)
		}
		cacerts[i] = cacert
	}
	return cacerts, nil
}

func (ca *certificateAuthority) GenerateCRL(ctx context.Context, caName string) ([]byte, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "GetCRL")
	defer span.Finish()
	span.SetTag("ca.name", caName)
	caID, ok := ca.CNToID[caName]
	if !ok {
		//Requested CA - CN does not exist - if name is empty AND only one cacert registered, return CRL for that one.
		//TODO: Investigate possible buggy behaviour if cas are swapped, added or removed during runtime
		span.SetTag("len(CNToID)", len(ca.CNToID))
		switch len(ca.CNToID) {
		case 1:
			for _, v := range ca.CNToID {
				caID = v
			}
		case 0:
			caID = "" //empty map?
		default:
			err := berrors.NotFoundError(fmt.Sprintf("CA-CN %s not found", caName))
			span.LogKV("event", "error", "error.object", err, "error.message", "ca.GetCRL impossible, CN does not exist")
			return nil, err
		}
	}
	crl, err := ca.client.GenerateCRL(ctx, &proto.IdRequest{Id: caID}, callOptions...)
	if err != nil {
		return nil, handleError(ctx, err)
	}
	return crl.CRL, nil
}

func (ca *certificateAuthority) GenerateOCSP(ctx context.Context, cert *core.Certificate, nonce []byte) ([]byte, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "GenerateOCSP")
	defer span.Finish()
	ocsp, err := ca.client.GenerateOCSPResponseByID(ctx, &capb.OCSPIdRequest{
		Id:    cert.ID,
		Nonce: nonce,
	})
	if err != nil {
		return nil, handleError(ctx, err)
	}
	return ocsp.OCSP, nil
}
