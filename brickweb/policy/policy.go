/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

//Package policy holds all the information about policies in package level variables
package policy

import (
	"gopkg.in/square/go-jose.v2"
)

//TODO: Use any of this
var (
	//BlacklistEnable controls whether the Domain Blacklist is used
	BlacklistEnable bool
	//Blacklist are domain suffixes we refuse to issue certificates to.
	//Example: []string{".gov", "example.com"} will cause WFE to refuse to accept orders for www.example.com
	Blacklist []string
	//WhitelistEnable controls whether the Domain Whitelist is used
	WhitelistEnable bool
	//Whitelist are domain suffixes we exclusively issue certificates for
	Whitelist []string
)

//GetAllowedJWSAlgorithms returns the JWK Algorithms that are allowed by Policy, and Standard
func GetAllowedJWSAlgorithms() []string {
	return []string{string(jose.ES256), string(jose.ES384), string(jose.ES512), string(jose.RS256)}
}
