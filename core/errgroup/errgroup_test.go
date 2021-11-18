/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package errgroup_test

import (
	"brick/core/errgroup"
	"errors"
	"math/rand"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestOneNoError(t *testing.T) {
	g := errgroup.New()
	Convey("When one nothrow is added to an errorgroup", t, func() {
		g.Add(defaultTestFunc(false))
		Convey("A call to wait should return no Error", func() {
			g.Go()
			So(g.Wait(), ShouldBeNil)
		})
	})
}

func TestOneError(t *testing.T) {
	g := errgroup.New()
	Convey("When one throw is added to an errorgroup", t, func() {
		g.Add(defaultTestFunc(true))
		Convey("A call to wait should return an Error", func() {
			g.Go()
			So(g.Wait(), ShouldNotBeNil)
		})
	})
}

func TestMultipleNonErrors(t *testing.T) {
	g := errgroup.New()
	Convey("When many nothrows are added to an errorgroup", t, func() {
		g.Add(defaultTestFunc(false))
		g.Add(defaultTestFunc(false))
		g.Add(defaultTestFunc(false))
		g.Add(defaultTestFunc(false))
		g.Add(defaultTestFunc(false))
		g.Add(defaultTestFunc(false))
		Convey("A call to wait should return no Error", func() {
			g.Go()
			So(g.Wait(), ShouldBeNil)
		})
	})
}

func TestMultipleNonErrorsWithOneError(t *testing.T) {
	g := errgroup.New()
	Convey("When many nothrows and one throw are added to an errorgroup", t, func() {
		g.Add(defaultTestFunc(false))
		g.Add(defaultTestFunc(false))
		g.Add(defaultTestFunc(false))
		g.Add(defaultTestFunc(true))
		g.Add(defaultTestFunc(false))
		g.Add(defaultTestFunc(false))
		Convey("A call to wait should return an Error", func() {
			g.Go()
			So(g.Wait(), ShouldNotBeNil)
		})
	})
}

func TestMultipleErrors(t *testing.T) {
	g := errgroup.New()
	Convey("When many nothrows and one throw are added to an errorgroup", t, func() {
		g.Add(defaultTestFunc(true))
		g.Add(defaultTestFunc(true))
		g.Add(defaultTestFunc(true))
		g.Add(defaultTestFunc(true))
		g.Add(defaultTestFunc(true))
		g.Add(defaultTestFunc(true))
		g.Add(defaultTestFunc(true))
		g.Add(defaultTestFunc(true))
		Convey("A call to wait should return an Error", func() {
			g.Go()
			So(g.Wait(), ShouldNotBeNil)
		})
	})
}

func TestNoFuncs(t *testing.T) {
	g := errgroup.New()
	Convey("When absolutely no functions are added", t, func() {
		g.Go()
		Convey("No error should be returned by Wait", func() {
			So(g.Wait(), ShouldBeNil)
		})
	})
}
func TestForgetToCallGo(t *testing.T) {
	g := errgroup.New()
	Convey("When absolutely no functions are added", t, func() {
		Convey("and Wait is Called before Go", func() {
			So(func() { g.Wait() }, ShouldPanic)
		})
	})
}
func defaultTestFunc(shouldError bool) func() error {
	return func() error {
		ms := rand.Intn(1000)
		time.Sleep(time.Duration(ms) * time.Millisecond)
		if shouldError {
			return errors.New("Requested Error")
		}
		return nil
	}
}
