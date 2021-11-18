/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package core

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
)

//LoadPEMCertificate is a helper function loading a PEM from a file
func LoadPEMCertificate(filename string) (*x509.Certificate, error) {
	pemBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	pemBlock, _ := pem.Decode(pemBytes)
	if pemBlock == nil || len(pemBlock.Bytes) == 0 {
		return nil, fmt.Errorf("File %s PEM UNPARSEABLE", filename)
	}
	if pemBlock.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("File %s does not contain a PEM Certificate - PEM Type was: %s", filename, pemBlock.Type)
	}
	cert, err := x509.ParseCertificate(pemBlock.Bytes)
	if err != nil {
		return nil, err
	}
	return cert, nil
}
