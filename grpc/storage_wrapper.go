/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package grpc

import (
	"brick/core/log"
	"brick/grpc/corepb"
	"context"
	"crypto/sha1"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"errors"
	"math/big"
	"time"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"brick/brickweb/acme"
	"brick/core"
	"brick/core/berrors"
	proto "brick/grpc/corepb"
	"brick/grpc/sapb"
	"brick/grpc/tracing"
)

type storage struct {
	client sapb.StorageAuthorityClient
	logger logrus.FieldLogger
}

//NewStorageWrapper returns a Grpc Client Wrapper intending to implement the Storage interface of different subcomponents
//,e.g. VA, CA, WFE, ...
func NewStorageWrapper(grpcAddr string, logger logrus.FieldLogger, grpcCredentials []grpc.DialOption) (*storage, error) {
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
		/*		grpc.WithUnaryInterceptor(grpc_prometheus.UnaryClientInterceptor),
				grpc.WithStreamInterceptor(grpc_prometheus.StreamClientInterceptor), */
		append(grpcCredentials, grpc.WithUnaryInterceptor(tracing.TraceIDUnaryClientInterceptor("Datev-Trace-ID", LogStart, LogEnd)),
			grpc.WithStreamInterceptor(tracing.TraceIDStreamClientInterceptor("Datev-Trace-ID", LogStart)))...)

	if err != nil {
		return nil, err
	}
	logger.Infof("Connecting to %s", grpcAddr)
	client := sapb.NewStorageAuthorityClient(conn)
	return &storage{
		client: client,
		logger: logger,
	}, nil
}

func (s *storage) GetAccountByID(rootCtx context.Context, id string) (*core.Account, error) {
	span, ctx := opentracing.StartSpanFromContext(rootCtx, "GetAccountByID")
	defer span.Finish()
	log.WithTraceID(s.logger, ctx).WithField("grpc_method", "GetAccountByID").Debug("GRPC Call")
	protoAcct, err := s.client.GetAccount(ctx, &proto.IdRequest{Id: id}, callOptions...)
	if err != nil {
		return nil, s.handleError(ctx, err)
	}
	acct, err := protoToAccount(ctx, protoAcct)
	if err != nil {
		return nil, s.handleError(ctx, err)
	}
	return acct, nil
}

func (s *storage) AddAccount(rootCtx context.Context, acct *core.Account) error {
	span, ctx := opentracing.StartSpanFromContext(rootCtx, "AddAccount")
	defer span.Finish()
	log.WithTraceID(s.logger, ctx).WithField("grpc_method", "AddAccount").Debug("GRPC Call")
	var protoAcct, err = accountToProto(ctx, acct)
	if err != nil {
		s.logger.WithError(err).Error("Error marshaling Account to ProtoFormat")
		return err //TODO: Better error
	}
	_, err = s.client.AddAccount(ctx, protoAcct, callOptions...)
	if err != nil {
		if status.Code(err) != codes.NotFound {
			s.logger.WithError(err).WithField("rpc", "AddAccount").Warn("RPC Error occured")
		}
		return s.handleError(ctx, err)
	}
	return nil
}

//UpdateAccount updates the account with passed data.
func (s *storage) UpdateAccount(rootCtx context.Context, acct *core.Account) error {
	span, ctx := opentracing.StartSpanFromContext(rootCtx, "UpdateAccount")
	defer span.Finish()

	var protoAcct, err = accountToProto(ctx, acct)
	if err != nil {
		return s.handleError(ctx, err)
	}
	_, err = s.client.UpdateAccount(ctx, protoAcct, callOptions...)
	if err != nil {
		if status.Code(err) != codes.NotFound {
			s.logger.WithError(err).WithField("rpc", "UpdateAccount").Warn("RPC Error occured")
		}
		return s.handleError(ctx, err)
	}
	return nil
}

func (s *storage) AddOrder(rootCtx context.Context, o core.AddOrderRequest) (string, error) {
	span, ctx := opentracing.StartSpanFromContext(rootCtx, "AddOrder")
	defer span.Finish()

	newOrder := &sapb.NewOrder{
		ExpiresDate:            o.ExpiresDate,
		RequestedNotBeforeDate: o.RequestedNotBeforeDate,
		RequestedNotAfterDate:  o.RequestedNotAfterDate,
		AccountId:              o.AccountID,
	}
	span.LogKV("event", "NewOrder", "order.go", o, "order.proto", newOrder)
	newOrder.AuthzIDs = o.Authz
	ID, err := s.client.AddOrder(ctx, newOrder, callOptions...)
	if err != nil {
		return "", s.handleError(ctx, err)
	}
	return ID.Id, nil
}

func (s *storage) GetAuthFromIdent(rootCtx context.Context, identifier acme.Identifier, account *core.Account) (*core.Authorization, error) {
	span, ctx := opentracing.StartSpanFromContext(rootCtx, "GetAuthFromIdent")
	defer span.Finish()
	protoAuth, err := s.client.GetActiveAuthorization(ctx, &sapb.AccountAndIdent{
		AccountId: account.ID,
		Identifier: &proto.Identifier{
			Type:  identifier.Type,
			Value: identifier.Value,
		},
	}, callOptions...)
	if err != nil {
		return nil, s.handleError(ctx, err)
	}
	auth, err := protoToAuthorization(ctx, protoAuth)
	if err != nil {
		return nil, s.handleError(ctx, err)
	}
	return auth, nil
}

func (s *storage) AddAuthorization(rootCtx context.Context, a core.AddAuthz) (string, error) {
	span, ctx := opentracing.StartSpanFromContext(rootCtx, "AddAuthorization")
	defer span.Finish()

	var newAuthz = &sapb.NewAuthz{
		Identifier: &proto.Identifier{
			Type:  a.Identifier.Type,
			Value: a.Identifier.Value,
		},
		ExpiresDate: a.ExpiresDate,
		AccountId:   a.AccountID,
	}
	newAuthz.Challenges = make([]*proto.Challenge, len(a.Challenges))
	for i, chal := range a.Challenges {
		newAuthz.Challenges[i] = &proto.Challenge{
			Type:  chal.Type,
			Token: chal.Token,
		}
	}

	ID, err := s.client.AddAuthorization(ctx, newAuthz, callOptions...)
	if err != nil {
		return "", s.handleError(ctx, err)
	}
	return ID.Id, nil
}

func (s *storage) GetOrderByID(rootCtx context.Context, ID string) (*core.Order, error) {
	span, ctx := opentracing.StartSpanFromContext(rootCtx, "GetOrderByID")
	span.SetTag("id", ID)
	defer span.Finish()
	order, err := s.client.GetOrder(ctx, &proto.IdRequest{Id: ID}, callOptions...)
	if err != nil {
		return nil, s.handleError(ctx, err)
	}
	coreOrder, err := protoToOrder(ctx, order)
	if err != nil {
		return nil, s.handleError(ctx, err)
	}
	return coreOrder, nil
}

func (s *storage) GetAuthorizationByID(ctx context.Context, ID string) (*core.Authorization, error) {
	span, _ := opentracing.StartSpanFromContext(ctx, "GetAuthorizationByID")
	defer span.Finish()
	auth, err := s.client.GetAuthorization(ctx, &proto.IdRequest{Id: ID}, callOptions...)
	if err != nil {
		return nil, s.handleError(ctx, err)
	}
	authz, err := protoToAuthorization(ctx, auth)
	if err != nil {
		return nil, s.handleError(ctx, err)
	}
	return authz, nil
}

func (s *storage) UpdateOrder(ctx context.Context, o *core.Order) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "UpdateOrder")
	defer span.Finish()

	protoAuthz := make([]*corepb.Authorization, len(o.Authorizations))
	//TODO: Consider removing getting authz here
	for i, au := range o.Authorizations {
		protoAuth, err := s.client.GetAuthorization(ctx, &corepb.IdRequest{Id: au}, callOptions...)
		if err != nil {
			return s.handleError(ctx, err)
		}
		protoAuthz[i] = protoAuth
	}
	protoOrder, err := orderToProto(ctx, o, protoAuthz)
	_, err = s.client.UpdateOrder(ctx, protoOrder, callOptions...)
	if err != nil {
		return s.handleError(ctx, err)
	}
	return nil
}

//Check is a simple Application-Layer Healthcheck
//TODO: Implement
func (s *storage) Check(ctx context.Context) bool {
	span, _ := opentracing.StartSpanFromContext(ctx, "Check")
	defer span.Finish()
	span.SetTag("error", true)
	span.LogKV("event", "error", "error.object", errors.New("NYI"))
	return true
}

func (s *storage) StoreCertificate(ctx context.Context, cert *x509.Certificate, orderID string, cacertID string) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "StoreCertificate")
	defer span.Finish()
	idents := GetIdentifiersFromCert(ctx, cert)
	identStrings := make([]string, len(idents))
	for i, ident := range idents {
		identStrings[i] = ident.Value
	}
	newCertReq := &sapb.NewCert{
		Identifiers: identStrings,
		CertDER:     cert.Raw,
		OrderId:     orderID,
		CaCertId:    cacertID,
		Serial:      cert.SerialNumber.Bytes(),
	}
	order, err := s.client.GetOrder(ctx, &corepb.IdRequest{Id: orderID}, callOptions...)
	if err != nil {
		return s.handleError(ctx, err)
	}
	span.LogKV("NewCertReq.OrderId", orderID, "NewCertReq.CaCertId", cacertID)
	id, err := s.client.AddCertificate(ctx, newCertReq, callOptions...)
	if err != nil {
		return s.handleError(ctx, err)
	}
	span.LogKV(
		"message", "Inserted Certificate",
		"certID", id.Id,
	)
	order.Status = acme.StatusValid
	order.CertificateId = id.Id
	_, err = s.client.UpdateOrder(ctx, order, callOptions...)
	if err != nil {
		return s.handleError(ctx, err)
	}
	return nil
}
func (s *storage) StoreCaCertificate(ctx context.Context, cacert *x509.Certificate, cacertID string, willIssue bool) (string, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "Store")
	defer span.Finish()

	nameHash := sha1.Sum(cacert.RawSubject)
	id, err := s.client.AddCaCertificate(ctx, &corepb.CaCertificate{
		NameHash:   nameHash[:],
		WillIssue:  willIssue,
		CertDER:    cacert.Raw,
		CommonName: cacert.Subject.CommonName,
		CaCertId:   cacertID,
	}, callOptions...)
	if err != nil {
		return "", s.handleError(ctx, err)
	}
	return id.Id, nil
}

func (s *storage) CancelOrder(ctx context.Context, orderID string, problem *acme.ProblemDetails) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "CancelOrder")
	defer span.Finish()
	protoOrder, err := s.client.GetOrder(ctx, &corepb.IdRequest{Id: orderID}, callOptions...)
	if err != nil {
		return s.handleError(ctx, err)
	}
	protoOrder.Error = &corepb.Problem{
		Type:   problem.Type,
		Detail: problem.Error(),
	}
	protoOrder.Status = acme.StatusInvalid
	_, err = s.client.UpdateOrder(ctx, protoOrder, callOptions...)
	if err != nil {
		return s.handleError(ctx, err)
	}
	return nil
}

func (s *storage) GetCertificateStatusBySerial(ctx context.Context, serial *big.Int, issuerNameHash []byte) (*core.Certificate, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "GetCertificateStatusBySerial")
	defer span.Finish()
	panic("NYI")
}

func (s *storage) GetCertificateAndChain(ctx context.Context, certID string) (*core.Certificate, []*x509.Certificate, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "GetCertificateAndChain")
	defer span.Finish()
	protoCertificate, err := s.client.GetCertificate(ctx, &corepb.IdRequest{Id: certID}, callOptions...)
	cacerts := make([]*corepb.CaCertificate, 0)
	nextCaCert := protoCertificate.CaCert
	cacerts = append(cacerts, nextCaCert)
	//Loop until we receive a self-signed ("root") Cert
	for nextCaCert.CaCertId != nextCaCert.Id {
		nextCaCert, err = s.client.GetCaCertificate(ctx, &corepb.IdRequest{Id: nextCaCert.CaCertId}, callOptions...)
		if err != nil {
			return nil, nil, s.handleError(ctx, err)
		}
		cacerts = append(cacerts, nextCaCert)
	}

	leafCert, err := x509.ParseCertificate(protoCertificate.CertDER)
	if err != nil {
		return nil, nil, s.handleError(ctx, err)
	}
	var revocationTime time.Time
	if len(protoCertificate.RevocationTime) == 0 {
		revocationTime = time.Time{}
	} else {
		revocationTime, err = time.Parse(time.RFC3339, protoCertificate.RevocationTime)
		if err != nil {
			return nil, nil, s.handleError(ctx, err)
		}
	}
	coreCertificate := &core.Certificate{
		ID:             protoCertificate.Id,
		Cert:           leafCert,
		DER:            leafCert.Raw,
		IssuerID:       protoCertificate.Id,
		IssuerNameHash: protoCertificate.IssuerNameHash,
		RevocationTime: revocationTime,
		Serial:         leafCert.SerialNumber,
	}

	certChain := make([]*x509.Certificate, len(cacerts)+1)
	certChain[0] = leafCert
	for i, c := range cacerts {
		cert, err := x509.ParseCertificate(c.CertDER)
		if err != nil {
			return nil, nil, s.handleError(ctx, err)
		}
		certChain[i+1] = cert
	}
	return coreCertificate, certChain, nil
}

func (s *storage) RevokeCertificate(ctx context.Context, certID string, reason int) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "RevokeCertificate")
	defer span.Finish()
	_, err := s.client.RevokeCertificate(ctx, &sapb.RevokeCert{
		Id:               certID,
		RevocationReason: "Revoked by ACME",
	}, callOptions...)
	if err != nil {
		return s.handleError(ctx, err)
	}
	return nil
}

func (s *storage) GetCRL(ctx context.Context, caCN string) ([]byte, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "GetCRL")
	defer span.Finish()
	span.SetTag("error", true)
	span.LogKV("event", "error", "error.object", errors.New("NYI"), "error", true)
	return nil, berrors.NotFoundError("NYI") //Not Found, to cause caller to use CA instead
}

func (s *storage) StoreCRL(ctx context.Context, caCN string, crl []byte) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "GetCRL")
	defer span.Finish()
	span.SetTag("error", true)
	span.LogKV("event", "error", "error.object", errors.New("NYI"))
	return nil
}

func (s *storage) GetRevokedCerts(ctx context.Context, caID string) ([]pkix.RevokedCertificate, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "GetRevokedCerts")
	defer span.Finish()
	certs, err := s.client.GetRevokedCertificates(ctx, &corepb.IdRequest{Id: caID}, callOptions...)
	if err != nil {
		return nil, s.handleError(ctx, err)
	}
	revokedCerts := make([]pkix.RevokedCertificate, len(certs.RevokedCerts))
	for i, c := range certs.RevokedCerts {
		loopSpan, _ := opentracing.StartSpanFromContext(ctx, "ParseRevokedCert")
		bigInt := big.NewInt(0)
		bigInt.SetBytes(c.Serial)
		loopSpan.SetTag("serial", bigInt.Text(10))
		revocationTime, err := time.Parse(time.RFC3339, c.RevocationTime)
		if err != nil {
			span.SetTag("error", true)
			span.LogKV("event", "error", "error.object", err)
			loopSpan.Finish()
			return nil, s.handleError(ctx, err)
		}
		revokedCerts[i] = pkix.RevokedCertificate{
			SerialNumber:   bigInt,
			RevocationTime: revocationTime.UTC(),
		}
		loopSpan.Finish()
	}
	return revokedCerts, nil
}

func (s *storage) GetChallengeByID(ctx context.Context, id string) (*core.Challenge, string, string, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "GetChallengeByID")
	defer span.Finish()
	enrichedChalZ, err := s.client.GetChallenge(ctx, &corepb.IdRequest{Id: id}, callOptions...)
	if err != nil {
		return nil, "", "", s.handleError(ctx, err)
	}
	chalZ := enrichedChalZ.Challenge
	chal, err := protoToChallenge(ctx, chalZ)
	if err != nil {
		return nil, "", "", s.handleError(ctx, err)
	}
	return chal, enrichedChalZ.AccountId, enrichedChalZ.AuthorizationId, nil
}

//UpdateAuthorization updates the Authorization
func (s *storage) UpdateAuthorization(ctx context.Context, chal *core.Challenge, ID string, newStatus string) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "UpdateAuthorization")
	defer span.Finish()
	var err error
	if chal != nil {
		updatedChallenge, err := challengeToProto(ctx, chal)
		if err != nil {
			return s.handleError(ctx, err)
		}
		var updateAuthz = &sapb.UpdateAuthz{
			Id:               ID,
			NewStatus:        newStatus,
			UpdatedChallenge: updatedChallenge,
		}
		_, err = s.client.UpdateAuthorization(ctx, updateAuthz, callOptions...)
		if err != nil {
			return s.handleError(ctx, err)
		}
	} else { //Challenge is nil, we just want to update the status
		var newStatusForID = &sapb.NewStatusForId{
			Id:     ID,
			Status: newStatus,
		}
		_, err = s.client.UpdateAuthorizationStatus(ctx, newStatusForID, callOptions...)
		if err != nil {
			return s.handleError(ctx, err)
		}
	}
	return nil
}

func (s *storage) UpdateChallengeStatus(ctx context.Context, ID string, newStatus string) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "UpdateChallengeStatus")
	defer span.Finish()
	_, err := s.client.UpdateChallengeStatus(ctx, &sapb.NewStatusForId{
		Id:     ID,
		Status: newStatus,
	}, callOptions...)
	if err != nil {
		return s.handleError(ctx, err)
	}
	return nil
}

func (s *storage) GetCertificateBySerial(ctx context.Context, serial *big.Int, issuerNameHash []byte) (*core.Certificate, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "GetCertificateBySerial")
	defer span.Finish()
	span.SetTag("serial", serial.String())
	span.SetTag("issuerNameHash", hex.EncodeToString(issuerNameHash))

	protoCertificate, err := s.client.GetCertificateBySerial(ctx, &sapb.CertBySerial{
		Serial:         serial.Bytes(),
		IssuerNameHash: issuerNameHash,
	}, callOptions...)
	if err != nil {
		return nil, s.handleError(ctx, err)
	}
	leafCert, err := x509.ParseCertificate(protoCertificate.CertDER)
	if err != nil {
		return nil, s.handleError(ctx, err)
	}
	t, err := parseTime(protoCertificate.RevocationTime)
	if err != nil {
		return nil, s.handleError(ctx, err)
	}
	coreCertificate := &core.Certificate{
		ID:             protoCertificate.Id,
		Cert:           leafCert,
		DER:            leafCert.Raw,
		IssuerID:       protoCertificate.Id,
		IssuerNameHash: protoCertificate.IssuerNameHash,
		RevocationTime: t,
		Serial:         serial,
		OrderID:        protoCertificate.OrderId,
	}
	return coreCertificate, nil
}
