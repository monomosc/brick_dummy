/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */
package wfe

import (
	"brick/brickweb/acme"
	"brick/brickweb/db"
	"brick/core"
	"context"
	"fmt"
	"time"

	opentracing "github.com/opentracing/opentracing-go"
	jose "gopkg.in/square/go-jose.v2"
)

func (wfe *WebFrontEndImpl) QueueValidateChallenge(ctx context.Context, challenge *core.Challenge, authz *core.Authorization, accountKey *jose.JSONWebKey) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "QueueValidateChallenge")
	defer span.Finish()
	err := db.SetChalProcessing(ctx, wfe.db, challenge.ID)
	wfe.Validation.DoValidation(ctx, &core.VerificationRequest{
		Context:       ctx,
		Challenge:     challenge,
		Authorization: authz,
		AccountJWK:    accountKey,
	})
	return err
}

func (wfe *WebFrontEndImpl) getChallengeJSON(ctx context.Context, ID string) (*acme.Challenge, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "getChallengeJSON")
	defer span.Finish()
	c, _, _, err := wfe.db.GetChallengeByID(ctx, ID)
	if err != nil {
		return nil, err
	}
	validated := ""
	if !c.ValidatedAt.IsZero() {
		validated = c.ValidatedAt.Format(time.RFC3339)
	}
	return &acme.Challenge{
		Type:      c.Type,
		URL:       wfe.RelativePath(fmt.Sprintf("%s%s", challengePath, ID)),
		Token:     c.Token,
		Status:    c.Status,
		Validated: validated,
		Error:     nil, //TODO: Implement error handling in challenge
	}, nil
}
