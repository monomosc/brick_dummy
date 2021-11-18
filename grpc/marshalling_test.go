/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package grpc_test

import (
	"brick/grpc"
	"context"
	"crypto/x509"
	"crypto/x509/pkix"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGetACMEIdentifiers(t *testing.T) {
	testCases := []struct {
		desc           string
		cert           *x509.Certificate
		expectedLength int
	}{
		{
			desc: "Test Cert with only CommonName",
			cert: &x509.Certificate{
				Subject: pkix.Name{
					CommonName: "localhost",
				},
			},
			expectedLength: 1,
		},
		{
			desc: "Test without CommonName and 2 SANs",
			cert: &x509.Certificate{
				DNSNames: []string{"localhost1", "localhost2"},
			},
			expectedLength: 2,
		},
		{
			desc: "Test with CommonName and matching SAN and another SAN",
			cert: &x509.Certificate{
				Subject: pkix.Name{
					CommonName: "localhost1",
				},
				DNSNames: []string{"localhost1", "localhost2"},
			},
			expectedLength: 2,
		},
		{
			desc: "Test with CommonName and 2 non-matching SANs",
			cert: &x509.Certificate{
				Subject: pkix.Name{
					CommonName: "localhost",
				},
				DNSNames: []string{"localhost1", "localhost2"},
			},
			expectedLength: 3,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			Convey("The amount of Identifiers in a x509 Certificate should be correct", t, func() {
				idents := grpc.GetIdentifiersFromCert(context.Background(), tC.cert)
				Convey(tC.desc, func() {
					So(len(idents), ShouldEqual, tC.expectedLength)
				})
			})
		})
	}
}
