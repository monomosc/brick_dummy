/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package grpc

import (
	"brick/brickweb/external"
	"brick/core/log"
	"context"
	"crypto/x509"
	"runtime/debug"
	"time"

	"github.com/pkg/errors"

	"brick/brickweb/acme"
	"brick/core"
	"brick/core/berrors"
	proto "brick/grpc/corepb"
	"brick/grpc/vapb"

	opentracing "github.com/opentracing/opentracing-go"
	jose "gopkg.in/square/go-jose.v2"
)

//Account
func protoToAccount(rootCtx context.Context, p *proto.Account) (*core.Account, error) {
	span, _ := opentracing.StartSpanFromContext(rootCtx, "protoToAccount")
	span.SetTag("id", p.Id)
	defer span.Finish()
	var key jose.JSONWebKey
	err := key.UnmarshalJSON(p.Key)
	if err != nil {
		span.LogKV("event", "error", "error.object", err, "message", "Could not UnmarshalJSON the key")
		return nil, err
	}
	createdAtTime, err := time.Parse(time.RFC3339, p.CreatedAt)
	if err != nil {
		span.LogKV("event", "error", "error.object", err)
		return nil, err
	}
	return &core.Account{
		ID: p.Id,
		Account: acme.Account{
			Status:  p.Status,
			Contact: p.Contact,
			Orders:  "",
		},
		Key:                &key,
		CreatedAt:          createdAtTime,
		ExternalIdentifier: external.Identifier(p.ExternalIdentifier),
	}, nil
}
func accountToProto(rootCtx context.Context, a *core.Account) (*proto.Account, error) {
	span, _ := opentracing.StartSpanFromContext(rootCtx, "accountToProto")
	span.SetTag("id", a.ID)
	defer span.Finish()
	if a == nil {
		panic("Account nil in AccountToProto")
	}
	keyBytes, err := a.Key.MarshalJSON()
	if err != nil {
		return nil, err
	}

	return &proto.Account{
		Id:                 a.ID,
		Key:                keyBytes,
		Status:             a.Status,
		Contact:            a.Contact,
		CreatedAt:          a.CreatedAt.Format(time.RFC3339),
		ExternalIdentifier: string(a.ExternalIdentifier),
	}, nil
}

//Authorization
func protoToAuthorization(rootCtx context.Context, a *proto.Authorization) (*core.Authorization, error) {
	span, ctx := opentracing.StartSpanFromContext(rootCtx, "accountToProto")
	span.SetTag("id", a.Id)
	defer span.Finish()
	if a == nil {
		e := errors.New("Nil Pointer")
		span.SetTag("error", true)
		span.LogKV("event", "error", "error.object", e, "stack", debug.Stack())
		return nil, berrors.UnknownError(e)
	}

	expDate, err := time.Parse(time.RFC3339, a.ExpiresDate)
	if err != nil {
		span.SetTag("error", true)
		span.LogKV("event", "error", "error.object", err, "message", "Could not parse ProtoAuthorization Datetime", "error.message", err.Error())
		return nil, err
	}
	chalz := make([]*core.Challenge, len(a.Challenges))
	for i, chal := range a.Challenges {
		c, err := protoToChallenge(ctx, chal)
		if err != nil {
			return nil, err
		}
		chalz[i] = c
	}
	return &core.Authorization{
		ID:          a.Id,
		ExpiresDate: expDate,
		Status:      a.Status,
		Wildcard:    false,
		Challenges:  chalz,
		Identifier: acme.Identifier{
			Type:  a.Identifier.Type,
			Value: a.Identifier.Value,
		},
		AccountID: a.AccountId,
	}, nil
}

func authorizationToProto(ctx context.Context, a *core.Authorization) (*proto.Authorization, error) {
	var err error

	var challenges = make([]*proto.Challenge, len(a.Challenges))
	for i, c := range a.Challenges {
		challenges[i], err = challengeToProto(ctx, c)
		if err != nil {
			return nil, errors.Wrap(err, "could not marshal authorization to proto")
		}
	}
	return &proto.Authorization{
		Id:          a.ID,
		ExpiresDate: a.ExpiresDate.Format(time.RFC3339),
		Status:      a.Status,
		Identifier: &proto.Identifier{
			Value: a.Identifier.Value,
			Type:  a.Identifier.Type,
		},
		AccountId:  a.AccountID,
		Challenges: challenges,
	}, nil
}

// ProtoToValidation transforms the wireformat an authorization into a usable VerificationRequest struct
func ProtoToValidation(ctx context.Context, valMsg *vapb.ValidationMessage) (*core.VerificationRequest, error) {
	authorization, err := protoToAuthorization(ctx, valMsg.Authorization)
	if err != nil {
		return nil, errors.Wrap(err, "Error reading Authorization from request")
	}

	challenge, err := protoToChallenge(ctx, valMsg.Challenge)
	if err != nil {
		return nil, errors.Wrap(err, "Error reading Challenge from request")
	}

	k := new(jose.JSONWebKey)
	err = k.UnmarshalJSON([]byte(valMsg.AccountJWK))
	if err != nil {
		return nil, errors.Wrap(err, "Error reading JSON Webkey from request")
	}
	var newCtx = context.Background()
	newCtx = log.SetTraceID(newCtx, log.GetTraceID(ctx))
	verificationRequest := core.VerificationRequest{Context: newCtx, Challenge: challenge, Authorization: authorization, AccountJWK: k, Retries: 0}

	return &verificationRequest, nil
}

// ValidationToProto transforms a validation struct into wireformat
func ValidationToProto(ctx context.Context, validation *core.VerificationRequest) (*vapb.ValidationMessage, error) {
	wireAuthorization, err := authorizationToProto(ctx, validation.Authorization)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to transform authorization in a validation request into wireformat")
	}

	wireChallenge, err := challengeToProto(ctx, validation.Challenge)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to transform challenge in a validation request into wireformat")
	}

	webKey, err := validation.AccountJWK.MarshalJSON()
	if err != nil {
		return nil, errors.Wrap(err, "Unable to transform JWK in a validation request into wireformat")
	}

	result := new(vapb.ValidationMessage)
	result.AccountJWK = string(webKey)
	result.Authorization = wireAuthorization
	result.Challenge = wireChallenge

	return result, nil
}

func protoToChallenge(ctx context.Context, chal *proto.Challenge) (*core.Challenge, error) {
	var valTime time.Time
	var err error
	if chal.Validated != "" {
		valTime, err = time.Parse(time.RFC3339, chal.Validated)
		if err != nil {
			return nil, err
		}
	}
	var e *acme.ProblemDetails
	if chal.Problem != nil {
		if chal.Problem.Type != "" {
			e = &acme.ProblemDetails{
				Type:   chal.Problem.Type,
				Detail: chal.Problem.Detail,
			}
		}
	}
	return &core.Challenge{
		ID:          chal.Id, //TODO: rethink data structure: this is an exceptionally ugly solution
		Token:       chal.Token,
		Status:      chal.Status,
		ValidatedAt: valTime,
		Type:        chal.Type,
		Error:       e, //TODO: Handle Problem/Error
	}, nil
}

func challengeToProto(ctx context.Context, c *core.Challenge) (*proto.Challenge, error) {
	var val = ""
	if !c.ValidatedAt.IsZero() {
		val = c.ValidatedAt.Format(time.RFC3339)
	}
	var problem *proto.Problem
	if c.Error != nil {
		problem = &proto.Problem{
			Type:   c.Error.Type,
			Detail: c.Error.Detail,
		}
	}
	return &proto.Challenge{
		Id:        c.ID,
		Token:     c.Token,
		Status:    c.Status,
		Type:      c.Type,
		Validated: val,
		Problem:   problem,
	}, nil
}

func protoToOrder(rootCtx context.Context, o *proto.Order) (*core.Order, error) {
	span, _ := opentracing.StartSpanFromContext(rootCtx, "protoToOrder")
	span.SetTag("id", o.Id)
	defer span.Finish()

	authzIDs := make([]string, len(o.Authz))
	Identifiers := make([]acme.Identifier, len(o.Authz))
	for i, a := range o.Authz {
		authzIDs[i] = a.Id
		Identifiers[i] = acme.Identifier{
			Type:  a.Identifier.Type,
			Value: a.Identifier.Value,
		}
	}
	_, err := time.Parse(time.RFC3339, o.ExpiresDate)
	if err != nil {
		span.SetTag("error", true)
		span.LogKV("event", "error", "error.kind", "OrderInvalid", "message", "Order set invalid", "error.object", err)
		return nil, err
	}
	//TODO: Add ProblemDetails
	return &core.Order{
		ID:            o.Id,
		AuthzIDs:      authzIDs,
		CertificateID: o.CertificateId,
		AccountID:     o.AccountId,
		Order: acme.Order{
			Status:         o.Status,
			Expires:        o.ExpiresDate,
			Identifiers:    Identifiers,
			Finalize:       "", //SET THIS IN WFE, NECESSARY INFO ONLY EXISTS THERE
			NotBefore:      o.RequestedNotBeforeDate,
			NotAfter:       o.RequestedNotAfterDate,
			Authorizations: nil, //SET THIS IN WFE
			Certificate:    "",  //SET THIS IN WFE
		},
	}, nil
}

func orderToProto(rootCtx context.Context, o *core.Order, authz []*proto.Authorization) (*proto.Order, error) {
	span, _ := opentracing.StartSpanFromContext(rootCtx, "orderToProto")
	span.SetTag("id", o.ID)
	defer span.Finish()
	if authz == nil {
		authz = []*proto.Authorization{}
	}
	return &proto.Order{
		Id:                     o.ID,
		Authz:                  authz,
		Status:                 o.Status,
		ExpiresDate:            o.Expires,
		CertificateId:          o.CertificateID,
		RequestedNotAfterDate:  o.NotAfter,
		RequestedNotBeforeDate: o.NotBefore,
		AccountId:              o.AccountID,
	}, nil
}

func protoToCaCert(rootCtx context.Context, c *proto.CaCertificate) (*core.CaCertificate, error) {
	span, ctx := opentracing.StartSpanFromContext(rootCtx, "protoToCaCert")
	defer span.Finish()
	handleError(ctx, errors.New("NYI"))
	panic("NYI")
}

func certToProto(ctx context.Context, c *core.Certificate) (*proto.Certificate, error) {
	span, _ := opentracing.StartSpanFromContext(ctx, "certToProto")
	defer span.Finish()
	panic("NYI")
}

//GetIdentifiersFromCert returns the acme Identifiers from a cert, by the following rules:
//all sans + common name; then remove duplicates
//TODO: Investigate if this function belongs in another package
func GetIdentifiersFromCert(ctx context.Context, cert *x509.Certificate) []acme.Identifier {
	span, _ := opentracing.StartSpanFromContext(ctx, "GetIdentifiersFromCert")
	defer span.Finish()
	var allNames = append([]string{}, cert.DNSNames...)
	if cert.Subject.CommonName != "" {
		allNames = append(allNames, cert.Subject.CommonName)
	}
	var elements = make(map[string]int)
	for i, el := range allNames {
		elements[el] = i
	}
	retSlice := make([]acme.Identifier, len(elements))
	i := 0
	for name := range elements {
		retSlice[i] = acme.Identifier{
			Type:  "dns",
			Value: name,
		}
		i++
	}
	return retSlice
}

type acmeInvalidError struct {
	innerErr error
}

func (aie *acmeInvalidError) Error() string {
	return aie.innerErr.Error()
}
func setAcmeInvalid(err error) error {
	return &acmeInvalidError{
		innerErr: err,
	}
}

//ShouldSetAcmeInvalid tells the caller, that the error is due to data corruption.
//The attempted call will never succed, so the corresponding ACME-Object should be set to Status Invalid.
func ShouldSetAcmeInvalid(err error) bool {
	_, ok := err.(*acmeInvalidError)
	return ok
}

func parseTime(timestamp string) (time.Time, error) {
	var t time.Time
	var err error
	if len(timestamp) == 0 {
		t = time.Time{}
	} else {
		t, err = time.Parse(time.RFC3339, timestamp)
		if err != nil {
			return time.Time{}, err
		}
	}
	return t, nil
}
