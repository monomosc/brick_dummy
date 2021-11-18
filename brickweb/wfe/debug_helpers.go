/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package wfe

import (
	"brick/brickweb/acme"
	"brick/brickweb/wfe/lib"
	"brick/core"
	"context"
	"time"

	"github.com/opentracing/opentracing-go"
)

func (wfe *WebFrontEndImpl) makeDefaultValidAuthz(ctx context.Context, accountID string) (string, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "makeDefaultValidAuthz")
	defer span.Finish()
	chalZ := lib.CreateDefaultChallenges(ctx, wfe.tokenizer)
	chalZ = append(chalZ, core.AddChallenge{
		Type:  "valid-01", //StorageAuthority knows about valid-01 and just adds it as valid
		Token: "",
	})
	addAuthz := core.AddAuthz{
		ExpiresDate: time.Now().UTC().Add(200 * time.Hour).Format(time.RFC3339),
		Identifier:  acme.Identifier{Type: "dns", Value: "localhost.local"},
		Challenges:  chalZ,
		AccountID:   accountID,
	}
	id, err := wfe.db.AddAuthorization(ctx, addAuthz)
	if err != nil {
		return "", err
	}
	return id, nil
}
