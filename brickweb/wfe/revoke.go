package wfe

import (
	"brick/brickweb/acme"
	"brick/core"
	"brick/core/berrors"
	"brick/core/log"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"net/http"

	opentracing "github.com/opentracing/opentracing-go"
)

//RevokeCert ist the Implementation of RFC 8555 7.6 Certificate Revocation
func (wfe *WebFrontEndImpl) RevokeCert(ctx context.Context, response http.ResponseWriter, r *http.Request) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "RevokeCert")
	defer span.Finish()
	var err error
	key, body, err := wfe.verifyPOSTNewAccount(ctx, r)
	if err != nil {
		wfe.handleError(ctx, err, response)
		return
	}
	var certRevocationReq struct {
		Certificate string `json:"certificate"`
	}
	err = json.Unmarshal(body, &certRevocationReq)
	if err != nil {
		wfe.handleError(ctx, err, response)
		return
	}
	certRaw, err := base64.RawURLEncoding.DecodeString(certRevocationReq.Certificate)
	if err != nil {
		wfe.handleError(ctx, err, response)
		return
	}
	cert, err := x509.ParseCertificate(certRaw)
	if err != nil {
		wfe.handleError(ctx, err, response)
		return
	}
	var issuerNameHash = sha1.Sum(cert.RawIssuer)
	coreCert, err := wfe.db.GetCertificateBySerial(ctx, cert.SerialNumber, issuerNameHash[:])
	if err != nil {
		if _, ok := berrors.IsNotFoundError(err); ok {
			wfe.handleError(ctx, acme.NotFoundProblem("This Certificate was not issued here"), response)
			return
		}
		wfe.handleError(ctx, err, response)
		return
	}
	if !coreCert.RevocationTime.IsZero() {
		wfe.handleError(ctx, acme.AlreadyRevokedProblem(coreCert.RevocationTime), response)
		return
	}

	jwsKey := key.Key.(crypto.PublicKey)
	mayRevoke, err := wfe.checkRevocationAuthorization(ctx, jwsKey, coreCert)
	if !mayRevoke {
		log.WithTraceID(wfe.logger, ctx).Debug("Unauthorized RevokeCert Request")
		wfe.handleError(ctx, acme.UnauthorizedProblem("Cert Key does not match JWS Key"), response)
		return
	}
	log.WithTraceID(wfe.logger, ctx).WithField("commonname", coreCert.Cert.Subject.CommonName).Info("Revoking Cert")
	err = wfe.db.RevokeCertificate(ctx, coreCert.ID, 0)
	if err != nil {
		wfe.handleError(ctx, err, response)
		return
	}
	response.WriteHeader(200)
	return
}

func (wfe *WebFrontEndImpl) checkRevocationAuthorization(ctx context.Context, key crypto.PublicKey, cert *core.Certificate) (bool, error) {
	switch jwsKey := key.(type) {
	case *rsa.PublicKey:
		rsaCertKey, ok := cert.Cert.PublicKey.(*rsa.PublicKey)
		if !ok {
			log.WithTraceID(wfe.logger, ctx).Debug("Key Type of JWS PubKey is RSA but Cert Key Type is not")
			break
		}
		return rsaCertKey.N.Cmp(jwsKey.N) == 0, nil
	case *ecdsa.PublicKey:
		ecCertKey, ok := cert.Cert.PublicKey.(*ecdsa.PublicKey)
		if !ok {
			log.WithTraceID(wfe.logger, ctx).Debug("Key Type of JWS PubKey is ECDSA but Cert Key Type is not")
			break
		}
		return ecCertKey.Params().Name == jwsKey.Params().Name && ecCertKey.X.Cmp(jwsKey.X) == 0 && ecCertKey.Y.Cmp(jwsKey.Y) == 0, nil
	default:
		log.WithTraceID(wfe.logger, ctx).WithField("key", key).Warn("Intersting Key")
		break
	}

	//TODO: Account Key
	return false, nil
}
