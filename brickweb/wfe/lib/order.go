/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package lib

import (
	"context"

	"brick/brickweb/acme"
	"brick/brickweb/wfe/token"
	"brick/core"

	"github.com/opentracing/opentracing-go"
)

//CreateDefaultChallenges builds some Challenges as mandated by Policy (@Moritz TODO)
//It is a Function, not a Method as it is pure - it will always return the same based on Args
func CreateDefaultChallenges(rootCtx context.Context, tokenizer token.Tokenizer) []core.AddChallenge {
	span, _ := opentracing.StartSpanFromContext(rootCtx, "CreateDefaultChallenges")
	defer span.Finish()
	var httpChal = core.AddChallenge{
		Type:  acme.ChallengeHTTP01,
		Token: tokenizer.NewToken(),
	}
	return []core.AddChallenge{httpChal} //For now we only support HTTP-Auth
}
