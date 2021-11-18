/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package wfe

import (
	"brick/core/errgroup"
	"brick/core/log"
	"io"

	"github.com/husobee/vestigo"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"

	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/pprof"
	"time"
	"net"

	"brick/brickweb/acme"
	"brick/brickweb/external"
	"brick/brickweb/metrics"
	"brick/brickweb/wfe/nonce"
	"brick/brickweb/wfe/token"
	"brick/core"
)

const (
	// Note: We deliberately pick endpoint paths that differ from Boulder to
	// exercise clients processing of the /directory response
	directoryPath     = "/dir"
	noncePath         = "/new-nonce"
	newAccountPath    = "/new-acct"
	newAuthzPath      = "/new-authz"
	acctPath          = "/acct/"
	newOrderPath      = "/new-order"
	orderPath         = "/order/"
	orderFinalizePath = "/finalize-order/"
	authzPath         = "/authZ/"
	challengePath     = "/chalZ/"
	certPath          = "/certZ/"
	revokeCertPath    = "/revoke-cert"
	healthPath        = "/health"
)

//ca represents the CA to which Certificate Generation Requests are passed
type ca interface {
	CompleteOrder(context.Context, *core.Order, *x509.CertificateRequest) error
}

//statusRecorder wraps the passed http.ResponseWriter and records the status
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (rec *statusRecorder) WriteHeader(code int) {
	rec.status = code
	rec.ResponseWriter.WriteHeader(code)
}

type wfeHandlerFunc func(context.Context, http.ResponseWriter, *http.Request)

func (wfe *WebFrontEndImpl) handlerFunc(f wfeHandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		spanContext, err := opentracing.GlobalTracer().Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(r.Header))
		var span opentracing.Span
		if err != nil {
			span = opentracing.StartSpan(GetSpanNameFromRequest(r))
		} else {
			span = opentracing.StartSpan(GetSpanNameFromRequest(r), opentracing.ChildOf(spanContext))
		}
		defer span.Finish()

		ctx, cancel := context.WithTimeout(r.Context(), time.Second*4)
		ctx = opentracing.ContextWithSpan(ctx, span)
		addNoCacheHeader(w)
		//Span for nonce creation
		nonceSpan := opentracing.StartSpan("newNonce", opentracing.ChildOf(span.Context()))
		newNonce := string(wfe.Noncer.Next())
		nonceSpan.LogKV("newNonce", newNonce)
		w.Header().Set("Replay-Nonce", newNonce)
		nonceSpan.Finish()
		rec := &statusRecorder{w, 999}
		ctx = log.EnsureTraceID(ctx)
		ctx = log.WithField(ctx, "RequestPath", r.URL.Path)

		host, _, _ := net.SplitHostPort(r.RemoteAddr)
		log.WithTraceID(wfe.logger, ctx).WithFields(
			logrus.Fields{"event": "RequestStart",
			"http.method": r.Method,
			"http.url": r.RequestURI,
			"http.peer": host,
			"http.user-agent": r.UserAgent()}).Debugf("Request from %s with %s", r.RemoteAddr, r.UserAgent())

		defer func() {
			x := recover()
			if x != nil {
				log.Error(ctx, fmt.Errorf("Unhandled Panic in %s: %v", r.URL.Path, x), wfe.logger)
			}
		}()
		//Function Call
		f(ctx, rec, r)
		span.LogKV(
			"event", "http",
			"component", "wfe",
			"message", "Finished handling Http",
		)
		span.SetTag("http.status_code", rec.status)
		log.WithTraceID(wfe.logger, ctx).WithField("http.status_code", rec.status).Debug("Request finished")

		cancel() //Need to call cancel for cleanup of context
	}
}

func addNoCacheHeader(response http.ResponseWriter) {
	response.Header().Add("Cache-Control", "public, max-age=0, no-cache")
}

//WebFrontEndImpl represents the Webserver running
type WebFrontEndImpl struct {
	ca                        ca
	db                        storage
	Noncer                    nonce.NonceService
	logger                    logrus.FieldLogger
	BasePath                  string
	tokenizer                 token.Tokenizer
	AccountValidator          external.AccountValidator
	ProhibitGet               bool
	Validation                va
	WaitForIssuanceOnFinalize bool
}
type va interface {
	DoValidation(context.Context, *core.VerificationRequest) error
}

//New constructs a new WebFrontEnd or panics
func New(logger logrus.FieldLogger, ca ca, storage storage, validation va) *WebFrontEndImpl {
	return &WebFrontEndImpl{
		ca:         ca,
		db:         storage,
		Noncer:     nonce.NewNoncer(),
		logger:     logger,
		BasePath:   "http://localhost",
		tokenizer:  token.New(),
		Validation: validation,
	}
}

func (wfe *WebFrontEndImpl) Handler() http.Handler {
	m := vestigo.NewRouter()

	gor := pprof.Handler("goroutine")
	heap := pprof.Handler("heap")
	threadcreate := pprof.Handler("threadcreate")
	block := pprof.Handler("block")

	handlerhandlerfunc := func(h http.Handler) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			h.ServeHTTP(w, r)
		}
	}
	//Add pprof
	m.HandleFunc("/debug/pprof", pprof.Index)
	m.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	m.HandleFunc("/debug/pprof/profile", pprof.Profile)
	m.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	m.HandleFunc("/debug/goroutine", handlerhandlerfunc(gor))
	m.HandleFunc("/debug/threadcreate", handlerhandlerfunc(threadcreate))
	m.HandleFunc("/debug/heap", handlerhandlerfunc(heap))
	m.HandleFunc("/debug/block", handlerhandlerfunc(block))

	promHandler := promhttp.Handler()
	m.Get("/metrics", func(w http.ResponseWriter, r *http.Request) {
		promHandler.ServeHTTP(w, r)
	})

	//Application Code
	m.Get(directoryPath, wfe.handlerFunc(wfe.Directory), vestigo.Middleware(metrics.FullInstrumentingHandlerFunc))
	m.Post(newAccountPath, wfe.handlerFunc(wfe.NewAccount), vestigo.Middleware(metrics.FullInstrumentingHandlerFunc))
	m.Post(newOrderPath, wfe.handlerFunc(wfe.NewOrder), vestigo.Middleware(metrics.FullInstrumentingHandlerFunc))
	m.Post(fmt.Sprintf("%s:id", acctPath), wfe.handlerFunc(wfe.UpdateAccount), vestigo.Middleware(metrics.FullInstrumentingHandlerFunc))
	m.Post(fmt.Sprintf("%s:id", orderFinalizePath), wfe.handlerFunc(wfe.FinalizeOrder), vestigo.Middleware(metrics.FullInstrumentingHandlerFunc))
	m.Post(revokeCertPath, wfe.handlerFunc(wfe.RevokeCert), vestigo.Middleware(metrics.FullInstrumentingHandlerFunc))
	m.Handle(fmt.Sprintf("%s:id", orderPath), wfe.handlerFunc(wfe.HandleOrder), vestigo.Middleware(metrics.FullInstrumentingHandlerFunc))
	m.Handle(fmt.Sprintf("%s:id", certPath), wfe.handlerFunc(wfe.PostOrGetCert), vestigo.Middleware(metrics.FullInstrumentingHandlerFunc))
	m.Handle(fmt.Sprintf("%s:id", authzPath), wfe.handlerFunc(wfe.PostOrGetAuthz), vestigo.Middleware(metrics.FullInstrumentingHandlerFunc))
	m.Handle(fmt.Sprintf("%s:id", challengePath), wfe.handlerFunc(wfe.PostChallenge), vestigo.Middleware(metrics.FullInstrumentingHandlerFunc))
	m.Get(healthPath, wfe.HealthHandler)
	m.Get(noncePath, wfe.handlerFunc(wfe.Nonce), ContentType("application/json"), vestigo.Middleware(metrics.FullInstrumentingHandlerFunc))

	return m
}

func (wfe *WebFrontEndImpl) Nonce(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func (wfe *WebFrontEndImpl) Directory(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	span, _ := opentracing.StartSpanFromContext(ctx, "directory")
	defer span.Finish()
	w.Header().Set("content-type", "application/json")
	directoryEndpoints := map[string]interface{}{
		"newNonce":      wfe.RelativePath(noncePath),
		"newAccount":    wfe.RelativePath(newAccountPath),
		"newOrder":      wfe.RelativePath(newOrderPath),
		"revokeCert":    wfe.RelativePath(revokeCertPath),
		"newAuthz":      wfe.RelativePath(newAuthzPath),
		"random-string": wfe.tokenizer.NewToken(),
		"meta": struct {
			External bool `json:"externalAccountRequired"`
		}{
			External: wfe.AccountValidator != nil,
		},
	}
	enc := json.NewEncoder(w)
	enc.SetIndent(" ", "   ")
	enc.Encode(directoryEndpoints)
}

func (wfe *WebFrontEndImpl) UpdateAccount(rootCtx context.Context, response http.ResponseWriter, r *http.Request) {
	span, ctx := opentracing.StartSpanFromContext(rootCtx, "UpdateAccount")
	defer span.Finish()
	var err error
	postRequest, err := wfe.verifyPOST(ctx, r)
	if err != nil {
		wfe.handleError(ctx, err, response)
		return
	}
	var updateAccountReq struct { //TODO: Add ExternalAccBinding
		Contact []string `json:"contact"`
		Status  string   `json:"status"`
	}
	err = json.Unmarshal(postRequest.jwsBody, &updateAccountReq)
	if err != nil {
		wfe.handleError(ctx, err, response)
		return
	}

	// if this update contains no contacts or deactivated status,
	// simply return the existing account and return early.
	if len(updateAccountReq.Contact) == 0 && updateAccountReq.Status != acme.StatusDeactivated {
		err = wfe.writeJSONResponse(response, http.StatusOK, postRequest.account)
		if err != nil {
			wfe.handleError(ctx, err, response)
			return
		}
		return
	}

	newAcct := &core.Account{
		Account: acme.Account{
			Contact: postRequest.account.Contact,
			Status:  postRequest.account.Status,
			Orders:  postRequest.account.Orders,
		},
		Key:       postRequest.account.Key,
		ID:        postRequest.account.ID,
		CreatedAt: postRequest.account.CreatedAt.UTC(),
	}

	//Account State State Machine: see ACME-DRAFT Section 7.1
	switch {
	case updateAccountReq.Status == acme.StatusDeactivated:
		newAcct.Status = updateAccountReq.Status
	case updateAccountReq.Status != "" && updateAccountReq.Status != newAcct.Status:
		wfe.sendError(ctx,
			acme.MalformedProblem(fmt.Sprintf(
				"Invalid account status: %q", updateAccountReq.Status)), response)
		return
	case len(updateAccountReq.Contact) > 0:
		newAcct.Contact = updateAccountReq.Contact
		// Verify that the contact information provided is supported & valid
		prob := wfe.verifyContacts(ctx, newAcct.Contact)
		if prob != nil {
			wfe.sendError(ctx, prob, response)
			return
		}
	}
	span.LogKV("newAcct", newAcct)
	//Everything OK!
	err = wfe.db.UpdateAccount(ctx, newAcct)
	if err != nil {
		wfe.handleError(ctx, err, response)
		return
	}
	err = wfe.writeJSONResponse(response, http.StatusOK, newAcct)
	if err != nil {
		wfe.handleError(ctx, err, response)
		return
	}
	return
}

func (wfe *WebFrontEndImpl) NewOrder(rootCtx context.Context, response http.ResponseWriter, r *http.Request) {
	span, ctx := opentracing.StartSpanFromContext(rootCtx, "NewOrder")
	defer span.Finish()
	var err error
	postRequest, err := wfe.verifyPOST(ctx, r)
	if err != nil {
		wfe.handleError(ctx, err, response)
		return
	}
	var newOrderReq struct {
		Identifiers []acme.Identifier `json:"identifiers"`
		NotBefore   string            `json:"notBefore"`
		NotAfter    string            `json:"notAfter"`
	}
	err = json.Unmarshal(postRequest.jwsBody, &newOrderReq)
	if err != nil {
		wfe.handleError(ctx, err, response)
		return
	}
	if len(newOrderReq.Identifiers) == 0 {
		err := acme.MalformedProblem("Cannot have 0 Identifiers in a Certificate")
		wfe.handleError(ctx, err, response)
		return
	}
	span.LogKV("event", "ParseNewOrder", "order.NotBefore", newOrderReq.NotBefore, "order.NotAfter", newOrderReq.NotAfter)

	authzIDC := make(chan string, len(newOrderReq.Identifiers))
	authzIDs := make([]string, len(newOrderReq.Identifiers))

	eg := errgroup.New()
	//TODO: Check for duplicate Identifiers
	//Check if valid or pending Authorizations already exist - make it concurrent for style
	countOfIdentifiers := len(newOrderReq.Identifiers)
	for _, ident := range newOrderReq.Identifiers {
		var identifier = ident
		eg.Add(func() error {
			innerSpan, innerCtx := opentracing.StartSpanFromContext(ctx, "CheckForNewAuth")
			defer innerSpan.Finish()
			innerSpan.SetTag("identifier", identifier.Value)
			innerSpan.LogKV("identifier", identifier.Value)
			id, err := wfe.createOrGetAuthorization(innerCtx, postRequest.account, identifier)
			if err != nil {
				span.SetTag("error", true)
				span.LogKV("event", "error", "error.object", err)
				return err
			}
			authzIDC <- id
			return nil
		})
	}
	eg.Go()
	err = eg.Wait()
	if err != nil {
		wfe.handleError(ctx, err, response)
		close(authzIDC)
		return
	}
	var i = 0
	for {
		select {
		case a := <-authzIDC:
			authzIDs[i] = a
			i++
		}
		if i == countOfIdentifiers {
			break
		}
	}
	close(authzIDC)
	//Alright: Got a Slice of Authorization IDs

	newOrderGrpcThing := core.AddOrderRequest{
		ExpiresDate:            time.Now().UTC().Add(time.Hour * 200).Format(time.RFC3339),
		RequestedNotBeforeDate: newOrderReq.NotBefore,
		RequestedNotAfterDate:  newOrderReq.NotAfter,
		AccountID:              postRequest.account.ID,
		Authz:                  authzIDs,
	}

	newOrderID, err := wfe.db.AddOrder(ctx, newOrderGrpcThing)
	if err != nil {
		wfe.handleError(ctx, err, response)
		return
	}
	newOrder, err := wfe.db.GetOrderByID(ctx, newOrderID)
	//newOrder is not complete, as there is wfe-specific data that can only be supplied here
	newOrder.Finalize = wfe.RelativePath(fmt.Sprintf("%s%s", orderFinalizePath, newOrderID))
	authzPaths := make([]string, len(newOrder.AuthzIDs))
	for i, a := range newOrder.AuthzIDs {
		authzPaths[i] = wfe.RelativePath(fmt.Sprintf("%s%s", authzPath, a))
	}
	newOrder.Authorizations = authzPaths

	response.Header().Set("Location", wfe.RelativePath(fmt.Sprintf("%s%s", orderPath, newOrderID)))
	err = wfe.writeJSONResponse(response, 201, newOrder.Order)
	if err != nil {
		wfe.handleError(ctx, err, response)
		return
	}
	return
}

func (wfe *WebFrontEndImpl) HandleOrder(rootCtx context.Context, response http.ResponseWriter, r *http.Request) {
	span, ctx := opentracing.StartSpanFromContext(rootCtx, "GetOrder")
	defer span.Finish()
	var err error
	orderID := vestigo.Param(r, "id")
	span.SetTag("id", orderID)
	o, accID, err := wfe.orderForDisplay(ctx, orderID)
	if err != nil {
		wfe.handleError(ctx, err, response)
		return
	}
	switch r.Method {
	case "POST":
		postRequest, err := wfe.verifyPOST(ctx, r)
		if err != nil {
			wfe.handleError(ctx, err, response)
			return
		}
		if postRequest.account.ID != accID {
			wfe.handleError(ctx, acme.UnauthorizedProblem("Not your Order"), response)
			return
		}
		break
	case "GET":
		span.LogKV(
			"event", "depcrecation warning",
			"message", "GET on POST-as-GET",
			"resource", "order",
		)
		metrics.DepcrecationWarnings.WithLabelValues("order", "POST-as-GET").Inc()
		break
	default:
		wfe.handleError(ctx, acme.MethodNotAllowed(), response)
		return
	}
	if o.Status == acme.StatusProcessing {
		response.Header().Add("Retry-After", "2")
	}
	err = wfe.writeJSONResponse(response, 200, o)
	if err != nil {
		wfe.handleError(ctx, err, response)
		return
	}
	return
}

//FinalizeOrder checks Preconditions and sends the Order/CSR to CA for final Issuance
func (wfe *WebFrontEndImpl) FinalizeOrder(rootCtx context.Context, response http.ResponseWriter, r *http.Request) {
	span, ctx := opentracing.StartSpanFromContext(rootCtx, "FinalizeOrder")
	defer span.Finish()
	var err error
	//TODO: Implement POST-AS-GET Order
	orderID := vestigo.Param(r, "id")
	span.SetTag("id", orderID)
	postRequest, err := wfe.verifyPOST(ctx, r)
	if err != nil {
		log.WithTraceID(wfe.logger, ctx).WithError(err).Debug("Did not Verify PostRequest")
		wfe.handleError(ctx, err, response)
		return
	}
	order, err := wfe.db.GetOrderByID(ctx, orderID)
	if err != nil {
		wfe.handleError(ctx, err, response)
		return
	}
	if order.AccountID != postRequest.account.ID {
		wfe.handleError(ctx, acme.UnauthorizedProblem("AccountID and Order AccountID do not match"), response)
	}
	prob := validateOrderForIssuance(ctx, order, postRequest.account)
	if prob != nil {
		log.WithTraceID(wfe.logger, ctx).WithError(prob).Debug("Could not validate Order for Issuance")
		wfe.sendError(ctx, prob, response)
		return
	}

	var finalizeMessage struct {
		CSR string
	}
	err = json.Unmarshal(postRequest.jwsBody, &finalizeMessage)
	if err != nil {
		wfe.handleError(ctx, err, response)
		return
	}
	csrBytes, err := base64.RawURLEncoding.DecodeString(finalizeMessage.CSR)
	if err != nil {
		wfe.handleError(ctx, acme.MalformedProblem("Error decoding Base64url-encoded CSR: "+err.Error()), response)
		return
	}
	parsedCSR, err := x509.ParseCertificateRequest(csrBytes)
	if err != nil {
		wfe.handleError(ctx, acme.MalformedProblem("Error parsing Base64url-encoded CSR: "+err.Error()), response)
		return
	}
	//Update Order to State Processing before starting Complete Order
	order.Status = acme.StatusProcessing
	err = wfe.db.UpdateOrder(ctx, order)
	if err != nil {
		wfe.handleError(ctx, err, response)
		return
	}
	err = wfe.ca.CompleteOrder(ctx, order, parsedCSR)
	if err != nil {
		wfe.handleError(ctx, err, response)
		return
	}
	//If the feature flag "WaitForIssuance" is set we wait until the certificate is issued - that is until the order is valid
	if wfe.WaitForIssuanceOnFinalize {
		for {
			log.WithTraceID(wfe.logger, ctx).Info("Waiting for Certificate Issuance until returning")
			//if the order never progresses to valid oder invalid we eventually run into this context deadline and getOrderById will return an error
			waitCtx, c := context.WithTimeout(r.Context(), time.Second*5)
			defer c()
			order, err = wfe.db.GetOrderByID(waitCtx, orderID)
			if err != nil {
				wfe.handleError(ctx, err, response)
				return
			}
			if order.Status == acme.StatusValid || order.Status == acme.StatusInvalid {
				log.WithTraceID(wfe.logger, ctx).WithField("acme_status", order.Status).Info("Order reached terminal Status, quit polling")
				break
			}
			time.Sleep(time.Millisecond * 100)
		}
	}
	//Ignore orderAccountID: We checked somewhere up higher
	o, _, err := wfe.orderForDisplay(ctx, orderID)
	if err != nil {
		wfe.handleError(ctx, err, response)
		return
	}
	response.Header().Set("Retry-After", "3")
	err = wfe.writeJSONResponse(response, 200, o)
	if err != nil {
		wfe.handleError(ctx, err, response)
		return
	}
	return
}

func (wfe *WebFrontEndImpl) PostOrGetAuthz(ctx context.Context, response http.ResponseWriter, r *http.Request) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "PostOrGetAuthz")
	defer span.Finish()
	id := vestigo.Param(r, "id")
	var err error
	switch r.Method {
	case "POST":
		var postRequest *postRequest
		postRequest, err = wfe.verifyPOST(ctx, r)
		if err != nil {
			wfe.handleError(ctx, err, response)
			return
		}
		authz, err := wfe.getAuthzJSON(ctx, id, postRequest.account.ID)
		if err != nil {
			wfe.handleError(ctx, err, response)
			return
		}
		wfe.writeJSONResponse(response, 200, authz)
		return
	case "GET":
		if wfe.ProhibitGet {
			wfe.handleError(ctx, acme.Malformed405("Use POST-as-GET"), response)
			return
		}
		log.WithTraceID(wfe.logger, ctx).Debug("GET on Post-As-Get")
		metrics.DepcrecationWarnings.WithLabelValues("authz", "POST-as-GET").Inc()
		authz, err := wfe.getAuthzJSON(ctx, id, "")
		if err != nil {
			wfe.handleError(ctx, err, response)
			return
		}
		wfe.writeJSONResponse(response, 200, authz)
		return
	default:
		wfe.handleError(ctx, acme.MethodNotAllowed(), response)
		return
	}
}

func (wfe *WebFrontEndImpl) PostChallenge(ctx context.Context, response http.ResponseWriter, r *http.Request) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "PostOrGetChallenge")
	defer span.Finish()
	id := vestigo.Param(r, "id")
	if r.Method != "GET" && r.Method != "POST" {
		wfe.handleError(ctx, acme.MalformedProblem("Method not allowed"), response)
		return
	}
	if r.Method == "POST" {
		postRequest, prob := wfe.verifyPOST(ctx, r)
		if prob != nil {
			wfe.handleError(ctx, prob, response)
			return
		}
		if !postRequest.isPostAsGet {
			wfe.HandleStartChallenge(ctx, postRequest, id, response)
			return
		}
		// request is not POST with '{}' Content as mandated by ACME 7.5.1 but POST-AS-GET with  '' Content
		//We allow this for now, but Clients should update
		//This is not documented anywhere in the ACME specification
		log.WithTraceID(wfe.logger, ctx).WithField("path", r.URL.Path).Warn("Unusual Request: POST-AS-GET Request to Challenge URL. ACME Standard mandates '{}' Payload instead of ''")
		chal, err := wfe.getChallengeJSON(ctx, id)
		if err != nil {
			wfe.handleError(ctx, err, response)
			return
		}
		wfe.writeJSONResponse(response, 200, chal)
		return
	} else if r.Method == "GET" { //deprecated GET
		if wfe.ProhibitGet {
			wfe.handleError(ctx, acme.MalformedProblem("Method not allowed"), response)
			return
		}
		chal, owningAccID, authzID, err := wfe.db.GetChallengeByID(ctx, id)
		if err != nil {
			wfe.handleError(ctx, err, response)
			return
		}
		//Check Expiry:
		authz, err := wfe.db.GetAuthorizationByID(ctx, authzID)
		if err != nil {
			wfe.handleError(ctx, err, response)
			return
		}
		if authz.ExpiresDate.Before(time.Now().UTC()) {
			//TODO: SET authz invalid
			wfe.handleError(ctx, acme.MalformedProblem(fmt.Sprintf("The Authorization for %s is already expired", authz.Identifier.Value)), response)
			return
		}
		acc, err := wfe.db.GetAccountByID(ctx, owningAccID)
		if err != nil {
			wfe.handleError(ctx, err, response)
			return
		}
		//TODO: Investigate if there are challenge types which need updates here
		err = wfe.QueueValidateChallenge(ctx, chal, authz, acc.Key)
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
	}
	return
}

//GetCert downloads the Certificate
func (wfe *WebFrontEndImpl) PostOrGetCert(ctx context.Context, response http.ResponseWriter, r *http.Request) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "GetCert")
	defer span.Finish()
	var err error
	if r.Method != "POST" && r.Method != "GET" { //sanity check
		wfe.handleError(ctx, acme.MethodNotAllowed(), response)
		return
	}
	if r.Method == "GET" && wfe.ProhibitGet {
		wfe.handleError(ctx, acme.Malformed405("Use POST-As-Get Requests"), response)
		return
	}
	if r.Method == "POST" {
		postRequest, err := wfe.verifyPOST(ctx, r)
		if err != nil {
			wfe.handleError(ctx, err, response)
			return
		}
		if !postRequest.isPostAsGet {
			wfe.handleError(ctx, acme.MalformedProblem("JWS singed Payload should be empty for POST-AS-GET"), response)
			return
		}
		//TODO: Investigate if checking of certificate-account ownership should be implemented
		//I think not, because certificates might be pushed to CT-logs and are a public resource in any case.
	} else {
		span.LogKV(
			"event", "depcrecation warning",
			"message", "GET on POST-as-GET",
			"resource", "cert",
		)
		metrics.DepcrecationWarnings.WithLabelValues("cert", "POST-as-GET").Inc()
	}
	certID := vestigo.Param(r, "id")
	_, chain, err := wfe.db.GetCertificateAndChain(ctx, certID)
	if err != nil {
		wfe.handleError(ctx, err, response)
		return
	}
	c := core.CertificateChain(chain)
	response.Header().Set("content-type", "application/pem-certificate-chain")
	response.WriteHeader(200)
	_, err = response.Write(c.PEM(true, log.WithTraceID(wfe.logger, ctx)))
	if err != nil {
		wfe.handleError(ctx, err, response)
		return
	}
}

func (wfe *WebFrontEndImpl) HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	io.WriteString(w, "{\"result\" : \"ok\"}")
}
