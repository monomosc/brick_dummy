/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */
package http01

import (
	"brick/brickweb/acme"
	"brick/brickweb/metrics"
	"brick/core"
	"brick/core/log"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"runtime"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

func userAgent() string {
	return fmt.Sprintf("Brick (%s, %s)", runtime.GOOS, runtime.GOARCH)
}

func VerifyHTTP01(ctx context.Context, req core.VerificationRequest, logger logrus.FieldLogger) error {
	body, err := fetchToken(ctx, req.Authorization.Identifier.Value, req.Challenge.Token, logger)
	if err != nil {
		log.Error(ctx, err, logger)
		return convertToAcmeProblem(err)
	}
	expectedKeyAuthorization, err := acme.ExpectedKeyAuthorization(ctx, req.Challenge.Token, req.AccountJWK)
	if err != nil {
		log.Error(ctx, err, logger)
		return convertToAcmeProblem(err)
	}
	payload := strings.TrimRight(string(body), "\n\r\t")
	if payload != expectedKeyAuthorization {
		log.WithTraceID(logger, ctx).WithField("expected", expectedKeyAuthorization).WithField("actual", payload).Warn("Expected Key Authorization did not match")
		return acme.UnauthorizedProblem(fmt.Sprintf("The key authorization file from server did not match this challenge: %q != %q", expectedKeyAuthorization, payload))
	}
	return nil
}

func fetchToken(ctx context.Context, identifier string, token string, outerLogger logrus.FieldLogger) ([]byte, error) {
	url := &url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("%s:%s", identifier, "80"), //Challenge Validation hardcoded to port 80
		Path:   fmt.Sprintf("%s%s", acme.HTTP01BaseURL, token),
	}
	logger := outerLogger.WithField("url", url.String())
	httpRequest, err := http.NewRequest(http.MethodGet, url.String(), nil)
	if err != nil {
		log.Error(ctx, err, logger)
		return nil, err
	}

	httpRequest.Header.Set("user-agent", userAgent())
	httpRequest.Header.Set("accept", "*/*")
	httpRequest.Header.Set("x-random-header-value", "abc")
	transport := &http.Transport{
		DisableKeepAlives: true,                                                   //we only do one roundtrip, ever
		TLSNextProto:      map[string]func(string, *tls.Conn) http.RoundTripper{}, //As per https://godoc.org/net/http this causes no upgrades to http/2
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   time.Second * 2,
	}
	if bytes, err := httputil.DumpRequestOut(httpRequest, true); err == nil {
		logger.WithField("request", string(bytes)).Info("Sending http request")
	}
	logger.WithField("request", httpRequest).Info("Sending http request")
	resp, err := client.Do(httpRequest)
	if err != nil {
		log.Error(ctx, err, logger)
		metrics.ChallengeHttpRequests.WithLabelValues("0").Inc()
		return nil, acme.ConnectionProblem(fmt.Sprintf("Could not connect to url %s", url.String()))
	}
	metrics.ChallengeHttpRequests.WithLabelValues(fmt.Sprintf("%d", resp.StatusCode)).Inc()
	logger.WithField("response", resp).Info("Received Response")
	if resp.StatusCode != http.StatusOK {
		//ACME RFC mandates status 200 on challenge GET
		return nil, acme.UnauthorizedProblem(fmt.Sprintf("Non-200 status code from GET %s: %d", url.String(), resp.StatusCode))
	}
	defer resp.Body.Close()
	bodyReader := io.LimitReader(resp.Body, 1000)
	body, _ := ioutil.ReadAll(bodyReader) // I don't believe there is an error that could happen here
	return body, nil
}

func convertToAcmeProblem(err error) *acme.ProblemDetails {
	prob, ok := err.(*acme.ProblemDetails)
	if ok {
		return prob
	}
	return acme.InternalErrorProblem("An Internal Error happened")
}
