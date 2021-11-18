/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package acme

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

//AcmeDraft 14 introduced a bunch of new Errors
const (
	errNS                      = "urn:ietf:params:acme:error:"
	serverInternalErr          = errNS + "serverInternal"
	malformedErr               = errNS + "malformed"
	badNonceErr                = errNS + "badNonce"
	agreementReqErr            = errNS + "agreementRequired"
	connectionErr              = errNS + "connection"
	unauthorizedErr            = errNS + "unauthorized"
	invalidContactErr          = errNS + "invalidContact"
	unsupportedContactErr      = errNS + "unsupportedContact"
	accountDoesNotExistErr     = errNS + "accountDoesNotExist"
	badRevocationReasonErr     = errNS + "badRevocationReason"
	alreadyRevokedErr          = errNS + "alreadyRevoked"
	badCSRErr                  = errNS + "badCSR"
	badSignatureAlgorithmErr   = errNS + "badSignatureAlgorithm"
	caaErr                     = errNS + "caa"
	compoundErr                = errNS + "compound"
	dnsErr                     = errNS + "dns"
	externalAccountRequiredErr = errNS + "externalAccountRequired"
	incorrectResponseErr       = errNS + "incorrectResponse"
	orderNotReadyErr           = errNS + "orderNotReady"
)

type ProblemDetails struct {
	Type        string            `json:"type,omitempty"`
	Detail      string            `json:"detail,omitempty"`
	HTTPStatus  int               `json:"status,omitempty"`
	Subproblems []*ProblemDetails `json:"subproblems,omitempty"`
	TraceID     string            `json:"trace"`
}

func (pd *ProblemDetails) Error() string {
	if pd.Subproblems == nil {
		return fmt.Sprintf("%s :: %s", pd.Type, pd.Detail)
	}
	if len(pd.Subproblems) >= 1 {
		errorStrings := make([]string, len(pd.Subproblems))
		for i, e := range pd.Subproblems {
			errorStrings[i] = e.Error()
		}
		return fmt.Sprintf("%s :: [%s]", pd.Type, strings.Join(errorStrings, ", "))
	}
	return fmt.Sprintf("%s :: %s", pd.Type, pd.Detail)
}

func InternalErrorProblem(detail string) *ProblemDetails {
	return &ProblemDetails{
		Type:       serverInternalErr,
		Detail:     detail,
		HTTPStatus: http.StatusInternalServerError,
	}
}

func AlreadyRevokedProblem(t time.Time) *ProblemDetails {
	return &ProblemDetails{
		Type:       alreadyRevokedErr,
		Detail:     fmt.Sprintf("Cert was already revoked on %v", t),
		HTTPStatus: http.StatusBadRequest,
	}
}

func ExternalAccountRequiredProblem(externalUri string) *ProblemDetails {
	return &ProblemDetails{
		Type:       externalAccountRequiredErr,
		Detail:     fmt.Sprintf(`This ACME Server is configured to require a binding to another account. Visit %s to find out how to do that`, externalUri),
		HTTPStatus: http.StatusBadRequest,
	}
}

func BadSignatureAlgorithmProblem(detail string) *ProblemDetails {
	return &ProblemDetails{
		Type:       badSignatureAlgorithmErr,
		Detail:     detail,
		HTTPStatus: http.StatusBadRequest,
	}
}

func Malformed405(detail string) *ProblemDetails {
	return &ProblemDetails{
		Type:       malformedErr,
		Detail:     detail,
		HTTPStatus: http.StatusMethodNotAllowed,
	}
}
func MalformedProblem(detail string) *ProblemDetails {
	return &ProblemDetails{
		Type:       malformedErr,
		Detail:     detail,
		HTTPStatus: http.StatusBadRequest,
	}
}

func NotFoundProblem(detail string) *ProblemDetails {
	return &ProblemDetails{
		Type:       malformedErr,
		Detail:     detail,
		HTTPStatus: http.StatusNotFound,
	}
}

func OrderNotReadyProblem(detail string) *ProblemDetails {
	return &ProblemDetails{
		Type:       orderNotReadyErr,
		Detail:     detail,
		HTTPStatus: http.StatusForbidden,
	}
}

func MethodNotAllowed() *ProblemDetails {
	return &ProblemDetails{
		Type:       malformedErr,
		Detail:     "Method not allowed",
		HTTPStatus: http.StatusMethodNotAllowed,
	}
}

func BadNonceProblem(detail string) *ProblemDetails {
	return &ProblemDetails{
		Type:       badNonceErr,
		Detail:     detail,
		HTTPStatus: http.StatusBadRequest,
	}
}

func Conflict(detail string) *ProblemDetails {
	return &ProblemDetails{
		Type:       malformedErr,
		Detail:     detail,
		HTTPStatus: http.StatusConflict,
	}
}

func AgreementRequiredProblem(detail string) *ProblemDetails {
	return &ProblemDetails{
		Type:       agreementReqErr,
		Detail:     detail,
		HTTPStatus: http.StatusForbidden,
	}
}

func ConnectionProblem(detail string) *ProblemDetails {
	return &ProblemDetails{
		Type:       connectionErr,
		Detail:     detail,
		HTTPStatus: http.StatusBadRequest,
	}
}

func UnauthorizedProblem(detail string) *ProblemDetails {
	return &ProblemDetails{
		Type:       unauthorizedErr,
		Detail:     detail,
		HTTPStatus: http.StatusForbidden,
	}
}

func InvalidContactProblem(detail string) *ProblemDetails {
	return &ProblemDetails{
		Type:       invalidContactErr,
		Detail:     detail,
		HTTPStatus: http.StatusBadRequest,
	}
}

func UnsupportedContactProblem(detail string) *ProblemDetails {
	return &ProblemDetails{
		Type:       unsupportedContactErr,
		Detail:     detail,
		HTTPStatus: http.StatusBadRequest,
	}
}

func AccountDoesNotExistProblem(detail string) *ProblemDetails {
	return &ProblemDetails{
		Type:       accountDoesNotExistErr,
		Detail:     detail,
		HTTPStatus: http.StatusBadRequest,
	}
}

func UnsupportedMediaTypeProblem(detail string) *ProblemDetails {
	return &ProblemDetails{
		Type:       malformedErr,
		Detail:     detail,
		HTTPStatus: http.StatusUnsupportedMediaType,
	}
}

func BadRevocationReasonProblem(detail string) *ProblemDetails {
	return &ProblemDetails{
		Type:       badRevocationReasonErr,
		Detail:     detail,
		HTTPStatus: http.StatusBadRequest,
	}
}

func BadCSRProblem(detail string) *ProblemDetails {
	return &ProblemDetails{
		Type:       badCSRErr,
		Detail:     detail,
		HTTPStatus: http.StatusBadRequest,
	}
}

func CompoundProblem(detail string, subproblems ...*ProblemDetails) *ProblemDetails {
	return &ProblemDetails{
		Type:        compoundErr,
		Detail:      detail,
		Subproblems: subproblems,
		HTTPStatus:  http.StatusInternalServerError,
	}
}
