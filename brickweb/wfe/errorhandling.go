/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package wfe

import (
	"brick/core/log"
	"context"
	"errors"
	"net/http"
	"runtime/debug"

	"brick/brickweb/acme"
	"brick/core/berrors"

	opentracing "github.com/opentracing/opentracing-go"
)

func SpanError(span opentracing.Span, err error) {
	span.SetTag("error", true)
	span.LogKV("event", "error", "error.object", err)
}

func (wfe *WebFrontEndImpl) sendError(ctx context.Context, prob *acme.ProblemDetails, response http.ResponseWriter) {
	if prob == nil {
		log.WithTraceID(wfe.logger, ctx).WithField("stack", string(debug.Stack())).Error("OUCH! Problem document in sendError is nil!")
		response.WriteHeader(http.StatusInternalServerError)
		return
	}
	prob.TraceID = log.GetTraceID(ctx)
	problemDoc, err := marshalIndent(prob)
	if err != nil {
		problemDoc = []byte("{\"detail\": \"Problem marshalling error message.\"}")
	}

	response.Header().Set("Content-Type", "application/problem+json")
	response.WriteHeader(prob.HTTPStatus)
	response.Write(problemDoc)
	log.WithTraceID(wfe.logger, ctx).WithError(prob).Info("Returned Problem")
}

//handleError handles the generic Error err, checking whether it is an acme.ProblemDetails, a berrors.BrickProblem or an unknown Error
//generally call it and return in http handler methods
func (wfe *WebFrontEndImpl) handleError(ctx context.Context, err error, response http.ResponseWriter) {
	if err == nil {
		log.Error(ctx, errors.New("Error in handleError is nil"), wfe.logger)
		return
	}
	log.WithTraceID(wfe.logger, ctx).Infof("Error: %v", err)
	switch e := err.(type) {
	case *acme.ProblemDetails:
		log.WithTraceID(wfe.logger, ctx).Infof("ProblemDetails: %v", e)
		wfe.sendError(ctx, e, response)
		return
	case *berrors.BrickProblem:
		wfe.sendError(ctx, acme.InternalErrorProblem("Something went wrong internally"), response)
		log.Error(ctx, err, wfe.logger)
		return
	default:
		wfe.logger.WithError(err).Error("An unexpected Error has occured")
		log.Error(ctx, err, wfe.logger)
		wfe.sendError(ctx, acme.InternalErrorProblem("Something went very wrong internally"), response)
		return
	}
}
