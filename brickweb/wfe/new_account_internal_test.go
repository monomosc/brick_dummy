/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package wfe

import (
	"brick/brickweb/acme"
	"brick/brickweb/external"
	"brick/core"
	"context"
	"reflect"
	"testing"

	jose "gopkg.in/square/go-jose.v2"
)

func Test_createNewAccountWithExternalBinding(t *testing.T) {
	type args struct {
		ctx                context.Context
		accountCreation    acme.AccountCreation
		key                *jose.JSONWebKey
		validator          external.AccountValidator
		externalAccountUri string
	}
	tests := []struct {
		name    string
		args    args
		want    *core.Account
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := createNewAccountWithExternalBinding(tt.args.ctx, tt.args.accountCreation, tt.args.key, tt.args.validator, tt.args.externalAccountUri)
			if (err != nil) != tt.wantErr {
				t.Errorf("createNewAccountWithExternalBinding() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("createNewAccountWithExternalBinding() = %v, want %v", got, tt.want)
			}
		})
	}
}
