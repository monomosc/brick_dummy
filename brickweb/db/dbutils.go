/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package db

import (
	"brick/brickweb/acme"
	"brick/core"
	"context"
	"time"

	opentracing "github.com/opentracing/opentracing-go"
)

type authUpdater interface {
	UpdateAuthorization(context.Context, *core.Challenge, string, string) error
}

type challengeUpdater interface {
	UpdateChallengeStatus(context.Context, string, string) error
	UpdateAuthorization(context.Context, *core.Challenge, string, string) error
}

//SetChallengeInvalid sets a Challenge and its Authorization to invalid and stores the Problem
func SetChallengeInvalid(ctx context.Context, db authUpdater, chal *core.Challenge, auth *core.Authorization, problem *acme.ProblemDetails) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "SetChallengeInvalid")
	defer span.Finish()
	var err error
	span.SetTag("problem", problem.Type)
	span.LogKV(
		"message", "Setting Challenge Invalid",
		"authz", auth,
		"chal", chal,
		"problem", problem,
	)
	chal.Error = problem
	err = db.UpdateAuthorization(ctx, chal, auth.ID, acme.StatusInvalid)
	if err != nil {
		span.LogKV("event", "error", "error.object", err)
		return err
	}
	return nil
}

//SetChallengeValid sets a Challenge and its Authorization to valid
func SetChallengeValid(ctx context.Context, db authUpdater, chal *core.Challenge, auth *core.Authorization) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "SetChallengeValid")
	defer span.Finish()
	chal.Status = acme.StatusValid
	chal.ValidatedAt = time.Now().UTC()
	span.LogKV(
		"message", "Setting Challenge Valid",
		"authz", auth,
		"chal", chal,
	)
	var err error
	err = db.UpdateAuthorization(ctx, chal, auth.ID, acme.StatusValid)
	if err != nil {
		span.LogKV("event", "error", "error.object", err)
		return err
	}
	return nil
}

//SetChalProcessing sets a challenge and its (passed) auth to processing
func SetChalProcessing(ctx context.Context, db challengeUpdater, chalID string) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "SetChalAndAuthProcessing")
	defer span.Finish()
	span.SetTag("chalID", chalID)
	err := db.UpdateChallengeStatus(ctx, chalID, acme.StatusProcessing)
	if err != nil {
		span.LogKV("event", "error", "error.object", err, "message", "Could not set Challenge processing")
		span.SetTag("error", true)
		return err
	}
	return nil
}
