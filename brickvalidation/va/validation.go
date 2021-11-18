/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package va

import (
	"brick/config"
	"brick/core/log"
	"runtime/debug"

	"github.com/pkg/errors"

	//TODO: Remove/ Refactor references to brickweb packages
	"brick/brickvalidation/va/http01"
	"brick/brickweb/acme"
	"brick/brickweb/db"
	"brick/core"
	"context"
	"math/rand"
	"time"

	"github.com/sirupsen/logrus"
)

const maxChallengeRetries = 5

var (
	verificationFuncs map[string]func(context.Context, core.VerificationRequest, logrus.FieldLogger) error
)

type storage interface {
	UpdateAuthorization(context.Context, *core.Challenge, string, string) error
}

//VA is the interface used for a VerificationAuthority
type VA interface {
	Start()
}

func New(ch chan core.VerificationRequest, l logrus.FieldLogger, s storage) VA {
	return &vaImpl{
		logger: l,
		c:      ch,
		db:     s,
	}
}

func (va *vaImpl) Start() {
	va.startLoop()
}

func init() {
	verificationFuncs = make(map[string]func(context.Context, core.VerificationRequest, logrus.FieldLogger) error)
	verificationFuncs["http-01"] = http01.VerifyHTTP01
}

type vaImpl struct {
	c      chan core.VerificationRequest
	logger logrus.FieldLogger
	db     storage
	config config.BrickValidationConfig
}

func (va *vaImpl) startLoop() {
	va.logger.Info("VA starting")
	go func() {
		for req := range va.c {
			go va.performValidation(req)
		}
	}()
}

func (va *vaImpl) performValidation(req core.VerificationRequest) {
	log.WithTraceID(va.logger, req.Context).WithField("type", req.Challenge.Type).Info("Starting Validation")
	defer recoverAndLogError(va.logger, req.Context) //This is done to prevent a complete crash of the application in case of faulty challenge validation code
	chal := req.Challenge
	verFunc := verificationFuncs[chal.Type]
	ctx := req.Context
	if va.config.Sleep {
		dur := time.Second * time.Duration(rand.Intn(4)+1)
		log.WithTraceID(va.logger, ctx).WithField("duration", dur).Debug("Sleeping for a while, because config.sleep is set")
		time.Sleep(dur)
	}
	if verFunc == nil {
		log.WithTraceID(va.logger, ctx).WithField("verFunc", chal.Type).Fatal("Verification Method does not exist")
		return
	}
	err := verFunc(ctx, req, log.WithTraceID(va.logger.WithField("ver-func", "http-01"), ctx))
	if err != nil {
		if req.Retries < maxChallengeRetries {
			req.Retries++
			log.WithTraceID(va.logger, ctx).WithError(err).WithField("retryCount", req.Retries).WithField("chalz", req.Challenge).WithField("authz", req.Authorization).Warn("Retrying Challenge Validation")
			va.c <- req
			return
		}
		log.WithTraceID(va.logger, ctx).WithField("chalz", req.Challenge).WithField("authz", req.Authorization).Warn("Setting Challenge Invalid")
		e := db.SetChallengeInvalid(ctx, va.db, req.Challenge, req.Authorization, getAcmeProblem(err))
		if e != nil {
			log.WithTraceID(va.logger, ctx).WithError(e).Error("Could not set Challenge Invalid")
			return
		}
		return
	}
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	log.WithTraceID(va.logger, ctx).WithField("chalz", req.Challenge).WithField("authz", req.Authorization).Info("Setting Challenge valid")
	err = db.SetChallengeValid(ctx, va.db, req.Challenge, req.Authorization)
	if err != nil {
		err = errors.Wrap(err, "Could not set ChallengeValid")
		log.Error(ctx, err, va.logger)
	}
	return
}

func recoverAndLogError(logger logrus.FieldLogger, ctx context.Context) {
	x := recover()
	if x != nil {
		log.WithTraceID(logger, ctx).WithField("stack", string(debug.Stack())).Error(x)
	}
}

func getAcmeProblem(err error) *acme.ProblemDetails {
	prob, ok := err.(*acme.ProblemDetails)
	if ok {
		return prob
	}
	return acme.InternalErrorProblem("An internal Error occured while verifying")
}
