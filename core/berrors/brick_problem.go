/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package berrors

import "fmt"

//BrickProblem represents an internal Error or unclear Condition
type BrickProblem struct {
	Type   errorType
	Detail string
}

type errorType string

const (
	notFoundType       = "NotFound"
	notImplementedType = "NotImplemented"
	timeoutType        = "Timeout"
	unknownType        = "Unknown"
)

func (b *BrickProblem) Error() string {
	return fmt.Sprintf("%s :: %s", b.Type, b.Detail)
}

//IsNotFoundError returns nil if err is not a BrickProblem or not a NotFound Error
//and the original error converted otherwise
func IsNotFoundError(err error) (*BrickProblem, bool) {
	b, ok := err.(*BrickProblem)
	if !ok {
		return nil, false
	}
	if b.Type == notFoundType {
		return b, true
	}
	return nil, false
}

//NotFoundError constructs a new NotFoundError
func NotFoundError(detail string) *BrickProblem {
	return &BrickProblem{
		Detail: detail,
		Type:   notFoundType,
	}
}

//NotImplementedError constructs a new NotImplementedError
func NotImplementedError(detail string) *BrickProblem {
	return &BrickProblem{
		Detail: detail,
		Type:   notImplementedType,
	}
}

func IsNotImplementedError(err error) (*BrickProblem, bool) {
	b, ok := err.(*BrickProblem)
	if !ok {
		return nil, false
	}
	if b.Type == notImplementedType {
		return b, true
	}
	return nil, false
}

//TimeoutError constructs a new NotImplementedError
func TimeoutError() *BrickProblem {
	return &BrickProblem{
		Detail: "A Subrequest timed out",
		Type:   timeoutType,
	}
}

func IsTimeoutError(err error) (*BrickProblem, bool) {
	b, ok := err.(*BrickProblem)
	if !ok {
		return nil, false
	}
	if b.Type == timeoutType {
		return b, true
	}
	return nil, false
}

func UnknownError(err error) *BrickProblem {
	return &BrickProblem{
		Detail: fmt.Sprintf("A deeper error occured: %s", err.Error()),
		Type:   unknownType,
	}
}
