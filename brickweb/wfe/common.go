/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package wfe

import (
	"brick/core/log"
	"context"
	"crypto"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/opentracing/opentracing-go"

	"brick/brickweb/acme"
	"brick/brickweb/wfe/lib"
	"brick/brickweb/wfe/nonce"
	"brick/brickweb/wfe/wfecrypto"
	"brick/core"
	"brick/core/berrors"

	jose "gopkg.in/square/go-jose.v2"
)

const (
	//ACNE Draft#14 says POST Requests MUST have content type application/jose+json
	expectedJWSContentType = "application/jose+json"
)

//keyExtractor is a function that returns the JSONWebKey (Account) used for a JSONWebSignature
//For example by Lookup in a Storage system (for existing Keys) or by fully parsing the signature
type keyExtractor func(context.Context, *http.Request, *jose.JSONWebSignature) (*jose.JSONWebKey, *acme.ProblemDetails)

type postRequest struct {
	isPostAsGet bool
	r           *http.Request
	account     *core.Account
	jwsBody     []byte
}

func ContentType(contentType string) func(http.HandlerFunc) http.HandlerFunc {
	return func(f http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", contentType)
			f(w, r)
		}
	}
}

func (wfe *WebFrontEndImpl) dumpBody(ctx context.Context, bodyBytes []byte) {
	log.WithTraceID(wfe.logger, ctx).WithField("jws", string(bodyBytes)).Debug("JWS Body Dump")
}

//verifyPOST only works with already registered accounts!
func (wfe *WebFrontEndImpl) verifyPOST(
	ctx context.Context,
	request *http.Request) (*postRequest, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, "verifyPOST")
	defer span.Finish()
	if prob := wfe.validPOST(ctx, request); prob != nil {
		log.WithTraceID(wfe.logger, ctx).WithError(prob).Debug("Did not validate POST")
		return nil, prob
	}
	if request.Body == nil {
		log.WithTraceID(wfe.logger, ctx).Debug("Weird Request: No Body on POST")
		return nil, acme.MalformedProblem("no body on POST")
	}
	bodyBytes, err := ioutil.ReadAll(request.Body)
	if err != nil {
		log.Error(ctx, err, wfe.logger)
		return nil, acme.InternalErrorProblem("unable to read request body")
	}
	parsedJWS, err := wfe.ParseJWS(ctx, bodyBytes)
	if err != nil {
		wfe.dumpBody(ctx, bodyBytes)
		log.WithTraceID(wfe.logger, ctx).WithError(err).Warn("could not parse jws")
		return nil, acme.MalformedProblem(err.Error())
	}
	account, prob := wfe.lookupJWK(ctx, request, parsedJWS)
	if prob != nil {
		wfe.dumpBody(ctx, bodyBytes)
		log.Error(ctx, prob, wfe.logger)
		return nil, prob
	}

	jwsPayload, prob := wfe.verifyJWS(ctx, account.Key, parsedJWS, request)
	if prob != nil {
		log.Error(ctx, prob, wfe.logger)
		return nil, prob
	}
	if jwsPayload == nil || len(jwsPayload) == 0 {
		return &postRequest{
			isPostAsGet: true,
			r:           request,
			account:     account,
			jwsBody:     nil,
		}, nil
	}
	return &postRequest{
		isPostAsGet: false,
		r:           request,
		account:     account,
		jwsBody:     jwsPayload,
	}, nil
}

func (wfe *WebFrontEndImpl) validPOST(rootCtx context.Context, request *http.Request) *acme.ProblemDetails {
	span, _ := opentracing.StartSpanFromContext(rootCtx, "validPOST")
	defer span.Finish()
	// Section 6.2 says to reject JWS requests without the expected Content-Type
	// using a status code of http.UnsupportedMediaType
	if _, present := request.Header["Content-Type"]; !present {
		return acme.UnsupportedMediaTypeProblem(
			`missing Content-Type header on POST. ` +
				`Content-Type must be "application/jose+json"`)
	}
	if contentType := request.Header.Get("Content-Type"); contentType != expectedJWSContentType {
		return acme.UnsupportedMediaTypeProblem(
			`Invalid Content-Type header on POST. ` +
				`Content-Type must be "application/jose+json"`)
	}
	// Per 6.4.1  "Replay-Nonce" clients should not send a Replay-Nonce header in
	// the HTTP request, it needs to be part of the signed JWS request body
	if _, present := request.Header["Replay-Nonce"]; present {
		return acme.MalformedProblem("HTTP requests should NOT contain Replay-Nonce header. Use JWS nonce field")
	}

	return nil
}

func (wfe *WebFrontEndImpl) ParseJWS(rootCtx context.Context, body []byte) (*jose.JSONWebSignature, error) {
	// Parse the raw JWS JSON to check that:
	// * the unprotected Header field is not being used.
	// * the "signatures" member isn't present, just "signature".
	//
	// This must be done prior to `jose.parseSigned` since it will strip away
	// these headers.
	span, _ := opentracing.StartSpanFromContext(rootCtx, "parseJWS")
	defer span.Finish()
	var unprotected struct {
		Header     map[string]string
		Signatures []interface{}
	}
	if err := json.Unmarshal(body, &unprotected); err != nil {
		return nil, errors.Wrap(err, "Parse error reading JWS")
	}

	// ACME v2 never uses values from the unprotected JWS header. Reject JWS that
	// include unprotected headers.
	if unprotected.Header != nil {
		return nil, errors.New(
			"JWS \"header\" field not allowed. All headers must be in \"protected\" field")
	}

	// ACME v2 never uses the "signatures" array of JSON serialized JWS, just the
	// mandatory "signature" field. Reject JWS that include the "signatures" array.
	if len(unprotected.Signatures) > 0 {
		return nil, errors.New(
			"JWS \"signatures\" field not allowed. Only the \"signature\" field should contain a signature")
	}

	parsedJWS, err := jose.ParseSigned(string(body))
	if err != nil {
		return nil, errors.New("Parse error reading JWS")
	}

	if len(parsedJWS.Signatures) > 1 {
		return nil, errors.New("Too many signatures in POST body")
	}

	if len(parsedJWS.Signatures) == 0 {
		return nil, errors.New("POST JWS not signed")
	}
	return parsedJWS, nil
}

func (wfe *WebFrontEndImpl) verifyJWS(
	rootCtx context.Context,
	pubKey *jose.JSONWebKey,
	parsedJWS *jose.JSONWebSignature,
	request *http.Request) ([]byte, *acme.ProblemDetails) {

	span, ctx := opentracing.StartSpanFromContext(rootCtx, "verifyJWS")
	defer span.Finish()
	payload, err := parsedJWS.Verify(pubKey)
	if err != nil {
		span.LogKV("event", "error", "error.object", err, "error.message", err.Error(), "message", "JWS Verification Error")
		span.SetTag("error", true)
		wfe.logger.WithError(err).Info("JWS Verification Error")
		return nil, acme.MalformedProblem("JWS verification error")
	}
	alg := parsedJWS.Signatures[0].Header.Algorithm
	span.SetTag("alg", alg)
	err = wfecrypto.CheckAlgorithm(ctx, pubKey, parsedJWS)
	if err != nil {
		span.SetTag("error", true)
		span.LogKV("event", "error", "error.object", err, "message", "CheckAlgorithm failed")
		wfe.logger.WithError(err).Info("JWS Algorithm bad")
		return nil, acme.BadSignatureAlgorithmProblem(fmt.Sprintf("Bad Signature Algorithm : %s", err.Error()))
	}

	non := parsedJWS.Signatures[0].Header.Nonce
	if len(non) == 0 {
		prob := acme.BadNonceProblem("JWS has no anti-replay nonce")
		span.SetTag("error", true)
		span.LogKV("event", "error", "error.object", prob, "message", "CheckAlgorithm failed", "error.type", prob.Type)
		return nil, prob
	}

	// If the nonce is not valid fail with an error
	if !wfe.Noncer.Valid(nonce.Nonce(non)) {
		prob := acme.BadNonceProblem(fmt.Sprintf("JWS has an invalid anti-replay nonce: %s", non))
		span.SetTag("error", true)
		span.LogKV("event", "error", "error.object", prob, "message", "CheckAlgorithm failed", "error.type", prob.Type)
		return nil, prob
	}

	headerURL, ok := parsedJWS.Signatures[0].Header.ExtraHeaders[jose.HeaderKey("url")].(string)
	if !ok || len(headerURL) == 0 {
		span.LogKV("message", "JWS header param 'url' required")
		return nil, acme.MalformedProblem("JWS header parameter 'url' required.")
	}
	expectedURL := url.URL{
		Scheme: "https",
		Host:   request.Host,
		Path:   request.RequestURI,
	}
	expectedURLHttp := url.URL{ //TODO; Disable for non-testing
		Scheme: "http",
		Host:   request.Host,
		Path:   request.RequestURI,
	}
	if expectedURL.String() != headerURL && expectedURLHttp.String() != headerURL {
		span.LogKV("message", "JWS header parameter 'url' incorrect", "url.expected", expectedURL.String(), "url.actual", headerURL)
		return nil, acme.MalformedProblem(fmt.Sprintf(
			"JWS header parameter 'url' incorrect. Expected %q, got %q",
			expectedURL.String(), headerURL))
	}
	log.WithTraceID(wfe.logger, ctx).Debug("JWS Verification successful")
	return payload, nil
}

func (wfe *WebFrontEndImpl) extractJWK(rootCtx context.Context, _ *http.Request, jws *jose.JSONWebSignature) (*jose.JSONWebKey, *acme.ProblemDetails) {
	span, _ := opentracing.StartSpanFromContext(rootCtx, "extractJWK")
	defer span.Finish()
	header := jws.Signatures[0].Header
	key := header.JSONWebKey
	if key == nil {
		return nil, acme.MalformedProblem("No JWK in JWS header")
	}
	if !key.Valid() {
		return nil, acme.MalformedProblem("Invalid JWK in JWS header")
	}
	if header.KeyID != "" {
		return nil, acme.MalformedProblem("jwk and kid header fields are mutually exclusive.")
	}
	return key, nil
}

func marshalIndent(v interface{}) ([]byte, error) {
	return json.MarshalIndent(v, "", "   ")
}

func (wfe *WebFrontEndImpl) RelativePath(p string) string {
	return fmt.Sprintf("%s%s", wfe.BasePath, p)
}

func (wfe *WebFrontEndImpl) writeJSONResponse(response http.ResponseWriter, status int, v interface{}) error {
	jsonReply, err := marshalIndent(v)
	if err != nil {
		return err // All callers are responsible for handling this error
	}

	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)

	// Don't worry about returning an error from Write() because the caller will
	// never handle it.
	_, _ = response.Write(jsonReply)
	return nil
}

func (wfe *WebFrontEndImpl) getACMEAccount(rootCtx context.Context, id string) (*acme.Account, error) {
	span, ctx := opentracing.StartSpanFromContext(rootCtx, "getACMEAccount")
	span.SetTag("comp", "wfe")
	defer span.Finish()
	acct, err := wfe.db.GetAccountByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &acct.Account, nil
}
func (wfe *WebFrontEndImpl) lookupJWK(rootCtx context.Context, request *http.Request, jws *jose.JSONWebSignature) (*core.Account, *acme.ProblemDetails) {
	span, ctx := opentracing.StartSpanFromContext(rootCtx, "lookupJWK")
	defer span.Finish()

	header := jws.Signatures[0].Header
	//Request Authentication as per ACME-Draft-14 Sec 6.2 Request Authentication
	accountURL := header.KeyID
	prefix := fmt.Sprintf("%s", wfe.RelativePath(acctPath))
	if !strings.HasPrefix(accountURL, prefix) {
		span.LogKV("event", "error", "message", "Key ID (kid) in JWS header missing expected URL prefix",
			"expectedKeyIDPrefix", prefix, "actualKeyID", accountURL)
		return nil, acme.MalformedProblem("Key ID (kid) in JWS header missing expected URL prefix")
	}
	accountID := strings.TrimPrefix(accountURL, prefix)
	if accountID == "" {
		return nil, acme.MalformedProblem("No KID in JWS Header")
	}
	log.WithTraceID(wfe.logger, ctx).WithField("kid", accountID).Debug("Looking up JWK")
	account, err := wfe.db.GetAccountByID(ctx, accountID)
	if _, ok := berrors.IsNotFoundError(err); ok {
		log.Error(ctx, err, wfe.logger)
		return nil, acme.AccountDoesNotExistProblem("Account not found")
	}
	if err != nil {
		log.Error(ctx, err, wfe.logger)
		return nil, acme.InternalErrorProblem("Error looking up Account")
	}
	if header.JSONWebKey != nil {
		log.Error(ctx, err, wfe.logger)
		return nil, acme.MalformedProblem("jwk and kid header fields are mutually exclusive.")
	}
	return account, nil
}

func (wfe *WebFrontEndImpl) verifyContacts(rootCtx context.Context, contacts []string) *acme.ProblemDetails {
	span, _ := opentracing.StartSpanFromContext(rootCtx, "verifyContacts")
	defer span.Finish()
	return nil //TODO: Verify Contacts External Hook
}

func (wfe *WebFrontEndImpl) orderForDisplay(ctx context.Context, orderID string) (*acme.Order, string, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "orderForDisplay")
	defer span.Finish()
	order, err := wfe.db.GetOrderByID(ctx, orderID)
	if err != nil {
		if _, ok := berrors.IsNotFoundError(err); ok {
			log.Error(ctx, err, wfe.logger)
			return nil, "", acme.NotFoundProblem("Order does not exist")
		}
		return nil, "", err
	}
	order.Authorizations = make([]string, len(order.AuthzIDs))
	for i, authID := range order.AuthzIDs {
		order.Authorizations[i] = wfe.RelativePath(fmt.Sprintf("%s%s", authzPath, authID))
	}
	
	order.Finalize = wfe.RelativePath(fmt.Sprintf("%s%s", orderFinalizePath, order.ID))
	order.Certificate = wfe.RelativePath(fmt.Sprintf("%s%s", certPath, order.CertificateID))
	span.SetTag("certID", order.CertificateID)
	return &order.Order, order.AccountID, nil
}

func (wfe *WebFrontEndImpl) getAccountByKey(rootCtx context.Context, key crypto.PublicKey) (*core.Account, error) {
	span, ctx := opentracing.StartSpanFromContext(rootCtx, "GetAccountByKey")
	defer span.Finish()
	id, err := wfecrypto.KeyToID(key)
	if err != nil {
		return nil, errors.Wrap(err, "could not get account by key")
	}
	acct, err := wfe.db.GetAccountByID(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "could not get account by key")
	}
	return acct, nil
}

func validateOrderForIssuance(rootCtx context.Context, order *core.Order, acct *core.Account) *acme.ProblemDetails {
	span, _ := opentracing.StartSpanFromContext(rootCtx, "validateOrderForIssuance")
	defer span.Finish()
	if order.AccountID != acct.ID {
		return acme.MalformedProblem("You are not the Owner of this Order")
	}
	if order.Status != acme.StatusReady {
		return acme.OrderNotReadyProblem(fmt.Sprintf("Order Status is %s, not ready", order.Status))
	}
	return nil
}

//createOrGetAuthorization Builds an Authorization
func (wfe *WebFrontEndImpl) createOrGetAuthorization(rootCtx context.Context, acct *core.Account, identifier acme.Identifier) (string, error) {
	span, ctx := opentracing.StartSpanFromContext(rootCtx, "createOrGetAuthorization")
	span.LogKV("identifier", identifier.Value, "account", acct.ID)
	defer span.Finish()
	auth, err := wfe.db.GetAuthFromIdent(ctx, identifier, acct)
	if err != nil {
		if bpb, ok := berrors.IsNotFoundError(err); !ok { //It's not a Not Found Error!
			return "", bpb
		} //OK, Auth does not exist, let's create it!
		return wfe.createAuthorization(ctx, acct, identifier)
	}
	//Auth exists, return it
	return auth.ID, nil
}

func (wfe *WebFrontEndImpl) createAuthorization(rootCtx context.Context, acct *core.Account, identifier acme.Identifier) (string, error) {
	span, ctx := opentracing.StartSpanFromContext(rootCtx, "createAuthorization")
	span.LogKV("identifier", identifier.Value, "account", acct.ID)
	defer span.Finish()
	
	//If The Identifier is "localhost.local", add a valid Challenge and set Auth Status to valid
	if identifier.Value == "localhost.local" {
		id, err := wfe.makeDefaultValidAuthz(ctx, acct.ID)
		if err != nil {
			return "", errors.Wrap(err, "could not make default valid auth for localhost.local")
		}
		return id, nil
	}
	addAuthz := core.AddAuthz{
		ExpiresDate: time.Now().UTC().Add(200 * time.Hour).Format(time.RFC3339),
		Identifier:  identifier,
		Challenges:  lib.CreateDefaultChallenges(ctx, wfe.tokenizer),
		AccountID:   acct.ID,
	}
	id, err := wfe.db.AddAuthorization(ctx, addAuthz)
	if err != nil {
		log.WithTraceID(wfe.logger, ctx).WithError(err).Error("could not add authorization")
		return "", errors.Wrap(err, "could not add authz")
	}
	return id, nil
}
var allPaths = []string{directoryPath, noncePath, newAccountPath, newAuthzPath, newOrderPath, acctPath, orderPath, orderFinalizePath, authzPath, challengePath, certPath, revokeCertPath}

func GetSpanNameFromRequest(r *http.Request) string {
	var uri = r.RequestURI
	actualPathFragment := ""
	for _, pathFragment := range allPaths {
		if strings.HasPrefix(uri, pathFragment) {
			actualPathFragment = pathFragment
		}
	}
	return fmt.Sprintf("%s %s", r.Method, actualPathFragment)
}

func (wfe *WebFrontEndImpl) HandleStartChallenge(ctx context.Context, postRequest *postRequest, id string, response http.ResponseWriter) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "HandleStartChallenge")
	defer span.Finish()
	var err error
	chal, owningAccID, authzID, err := wfe.db.GetChallengeByID(ctx, id)
	if err != nil {
		wfe.handleError(ctx, err, response)
		return
	}
	if postRequest.account.ID != owningAccID {
		wfe.handleError(ctx, acme.UnauthorizedProblem("Account authenticating Request is not owner of the challenge"), response)
		return
	}
	//Check Expiry:
	authz, err := wfe.db.GetAuthorizationByID(ctx, authzID)
	if err != nil {
		wfe.handleError(ctx, err, response)
		return
	}
	if authz.ExpiresDate.Before(time.Now().UTC()) {
		err = wfe.db.UpdateAuthorization(ctx, chal, authzID, acme.StatusExpired)
		wfe.handleError(ctx, acme.MalformedProblem(fmt.Sprintf("The Authorization for %s is already expired", authz.Identifier.Value)), response)
		return
	}
	//TODO: Investigate if there are challenge types which need updates here
	err = wfe.QueueValidateChallenge(ctx, chal, authz, postRequest.account.Key)
	if err != nil {
		wfe.handleError(ctx, err, response)
		return
	}
	//wait for a bit for the Challenge to really start processing
	time.Sleep(time.Millisecond * 300)
	//return challenge
	acmeChal, err := wfe.getChallengeJSON(ctx, id)
	if err != nil {
		wfe.handleError(ctx, err, response)
		return
	}
	err = wfe.writeJSONResponse(response, 200, acmeChal)
	if err != nil {
		wfe.handleError(ctx, err, response)
		return
	}
	return
}
