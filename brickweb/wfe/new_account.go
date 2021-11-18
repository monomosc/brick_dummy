/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package wfe

import (
	"brick/brickweb/acme"
	"brick/brickweb/external"
	"brick/brickweb/wfe/wfecrypto"
	"brick/core"
	"brick/core/berrors"
	"brick/core/log"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"time"

	opentracing "github.com/opentracing/opentracing-go"
	jose "gopkg.in/square/go-jose.v2"
)

func (wfe *WebFrontEndImpl) NewAccount(rootCtx context.Context, response http.ResponseWriter, r *http.Request) {
	span, ctx := opentracing.StartSpanFromContext(rootCtx, "newAccount")
	defer span.Finish()
	key, body, err := wfe.verifyPOSTNewAccount(ctx, r)
	if err != nil {
		wfe.handleError(ctx, err, response)
		return
	}

	var newAcctReq acme.AccountCreation
	err = json.Unmarshal(body, &newAcctReq)
	if err != nil {
		wfe.sendError(ctx, acme.MalformedProblem("Error Unmarshaling JSON"), response)
		wfe.logger.Info("Incapable of parsing Json")
		return
	}
	keyID, err := wfecrypto.KeyToID(key)
	if err != nil {
		SpanError(span, err)
		wfe.logger.Error(err)
		wfe.sendError(ctx, acme.MalformedProblem("Error reading Key"), response)
		return
	}
	l := log.WithTraceID(wfe.logger, ctx)
	l.WithField("email", newAcctReq.Contact).Info("Received NewAccount Request")
	existingAcct, err := wfe.db.GetAccountByID(ctx, keyID)
	if err != nil {
		_, ok := berrors.IsNotFoundError(err)
		if !ok { //NotFound is not an Error - anything else is!
			wfe.logger.WithError(err).Error("Could not lookup Account")
			SpanError(span, err)
			wfe.sendError(ctx, acme.InternalErrorProblem("Error looking up Account"), response)
			return
		}
	} else {
		//NO Error has occured, thus the account exists:
		acctURL := wfe.RelativePath(fmt.Sprintf("%s%s", acctPath, existingAcct.ID))
		response.Header().Set("Location", acctURL)
		wfe.writeJSONResponse(response, http.StatusOK, nil)
		return
	}
	//Check if onlyReturnExisting is set, return error if so (Acme-Draft-14 Sec 7.3.1)
	if newAcctReq.OnlyReturnExisting {
		prob := acme.AccountDoesNotExistProblem("OnlyReturnExisting was set and Account does not exist")
		span.LogKV("problem", prob)
		wfe.sendError(ctx, prob, response)
		return
	}
	var acct *core.Account
	if wfe.AccountValidator == nil {
		//Create a new Account
		acct, err = createNewAccount(ctx, newAcctReq, key)
		if err != nil {
			wfe.handleError(ctx, err, response)
			return
		}
	} else {
		acct, err = createNewAccountWithExternalBinding(ctx, newAcctReq, key, wfe.AccountValidator, "go/acme")
		if err != nil {
			wfe.handleError(ctx, err, response)
			return
		}
	}
	err = wfe.db.AddAccount(ctx, acct)
	if err != nil {
		wfe.handleError(ctx, err, response)
		return
	}

	//Get the newly created account for return
	acctURL := wfe.RelativePath(fmt.Sprintf("%s%s", acctPath, acct.ID))
	acmeAcct, err := wfe.getACMEAccount(ctx, acct.ID)
	if err != nil {
		wfe.handleError(ctx, err, response)
		return
	}
	response.Header().Set("Location", acctURL)
	wfe.writeJSONResponse(response, 201, acmeAcct)
	time.Sleep(time.Millisecond * time.Duration(rand.Intn(900)))
	return
}

//CreateNewAccount creates a new Core.Account from an AccountRequest
func createNewAccount(ctx context.Context, accountCreation acme.AccountCreation, key *jose.JSONWebKey) (*core.Account, error) {
	span, _ := opentracing.StartSpanFromContext(ctx, "CreateNewAccount")
	defer span.Finish()
	keyID, err := wfecrypto.KeyToID(key)
	if err != nil {
		return nil, err
	}
	newAcct := core.Account{
		Account: acme.Account{
			Contact: accountCreation.Contact,
			Status:  acme.StatusValid,
		},
		Key:       key,
		ID:        keyID,
		CreatedAt: time.Now().UTC(),
	}
	//TODO: Verify Account details (AcmeDraft #14 Sec 7.1)
	return &newAcct, nil
}

func createNewAccountWithExternalBinding(ctx context.Context, accountCreation acme.AccountCreation, key *jose.JSONWebKey, validator external.AccountValidator, externalAccountUri string) (*core.Account, error) {
	span, _ := opentracing.StartSpanFromContext(ctx, "CreateNewAccountWithExternalBinding")
	defer span.Finish()
	keyID, err := wfecrypto.KeyToID(key)
	if err != nil {
		return nil, err
	}
	if validator == nil {
		span.SetTag("error", true)
		span.LogKV("event", "error", "error.message", "nil validator passed")
		return nil, berrors.UnknownError(errors.New("Nil Validator passed to createNewAccount"))
	}
	//Check external
	accIdentifier, err := validator.Validate(ctx, accountCreation.ExternalAccountBinding)
	if err != nil {
		span.SetTag("error", true)
		span.LogKV("event", "error", "error.object", err, "message", "External Validator failed", "error.message", err.Error())
		return nil, acme.MalformedProblem("The passed Token Value for external account-binding was incorrect")
	}
	span.LogKV(
		"message", "Retrieved valid ExternalAccountIdentifier",
		"identifier", accIdentifier,
	)
	return &core.Account{
		Account: acme.Account{
			Contact: accountCreation.Contact,
			Status:  acme.StatusValid,
		},
		Key:                key,
		ID:                 keyID,
		CreatedAt:          time.Now().UTC(),
		ExternalIdentifier: accIdentifier,
	}, nil
}

func (wfe *WebFrontEndImpl) verifyPOSTNewAccount(ctx context.Context, request *http.Request) (*jose.JSONWebKey, []byte, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "verifyPOSTNewAccount")
	defer span.Finish()
	var err error
	prob := wfe.validPOST(ctx, request)
	if prob != nil {
		log.Error(ctx, err, wfe.logger)
		span.SetTag("error", true)
		span.LogKV("event", "error", "error.object", prob, "message", "Request is an invalid POST-Request", "reason", prob.Error())
		return nil, nil, prob
	}
	if request.Body == nil {
		log.Error(ctx, err, wfe.logger)
		span.SetTag("error", true)
		span.LogKV("event", "error", "error.object", acme.MalformedProblem("no body on POST"))
		return nil, nil, acme.MalformedProblem("no body on POST")
	}
	bodyBytes, err := ioutil.ReadAll(request.Body)
	if err != nil {
		log.Error(ctx, err, wfe.logger)
		span.SetTag("error", true)
		span.LogKV("event", "error", "error.object", err, "message", "unable to read requestBody")
		return nil, nil, acme.InternalErrorProblem("unable to read request body")
	}
	defer request.Body.Close()

	body := string(bodyBytes)
	span.SetTag("body", body)
	parsedJWS, err := wfe.ParseJWS(ctx, bodyBytes)
	if err != nil {
		log.Error(ctx, err, wfe.logger)
		span.SetTag("error", true)
		span.LogKV("event", "error", "error.object", err, "message", "could not parse JWS", "error.message", err.Error())
		return nil, nil, acme.MalformedProblem(err.Error())
	}
	key, prob := wfe.extractJWK(ctx, request, parsedJWS)
	if prob != nil {
		log.Error(ctx, err, wfe.logger)
		span.SetTag("error", true)
		span.LogKV("event", "error", "error.object", prob, "message", "could not extract JWK", "error.message", prob.Error())
		return nil, nil, prob
	}
	var payload []byte
	payload, prob = wfe.verifyJWS(ctx, key, parsedJWS, request)
	if prob != nil {
		log.Error(ctx, prob, wfe.logger)
		span.SetTag("error", true)
		span.LogKV("event", "error", "error.object", prob, "message", "INVALID jws", "error.type", prob.Type)
		return nil, nil, prob
	}
	return key, payload, nil
}
