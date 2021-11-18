/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package external

import (
	"context"
	"errors"
)

//NoopAccountValidator checks if externalAccountBinding is nil or empty, otherwise returns error
type NoopAccountValidator struct {
}

func (val *NoopAccountValidator) Validate(ctx context.Context, ext map[string]interface{}) (Identifier, error) {
	if ext == nil {
		return "", nil
	}
	if len(ext) == 0 {
		return "", nil
	}
	return "", errors.New("External Account Binding should be empty")
}
