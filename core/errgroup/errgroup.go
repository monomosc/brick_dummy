/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package errgroup

import (
	"sync"
	"sync/atomic"
)

//ErrGroup represents a bunch of Work being done on multiple goroutines.
//It is similar to sync.WaitGroup, but it also collects errors
//Usage: Call Add a bunch of times, Call Go, and then call Wait
//Always in this order!
type ErrGroup struct {
	count    int64
	done     chan struct{}
	errMutex sync.Mutex
	err      error
	start    chan struct{}
	started  bool
}

//New returns a new ErrGroup ready for Use
func New() *ErrGroup {
	return &ErrGroup{
		count:   0,
		done:    make(chan struct{}, 1),
		err:     nil,
		start:   make(chan struct{}),
		started: false,
	}
}

//Add adds an error returning function to the Error Group
func (g *ErrGroup) Add(f func() error) {
	if g.started {
		panic("Do not call Add after Go")
	}
	atomic.AddInt64(&g.count, 1)
	go func() {
		<-g.start
		err := f()
		if err != nil {
			g.errMutex.Lock()
			if g.err == nil {
				g.err = err
				close(g.done)
			}
			g.errMutex.Unlock()
			return
		}
		c := atomic.AddInt64(&g.count, -1)
		if c == 0 {
			close(g.done)
		}
	}()
}

//Go starts all functions passed by the previous add calls
func (g *ErrGroup) Go() {
	g.started = true
	close(g.start)
	if g.count == 0 {
		close(g.done)
	}
}

//Wait blocks until all functions have returned and returns the first error encountered
//Wait panics if called before Go()
func (g *ErrGroup) Wait() error {
	if g.started == false {
		panic("ErrGroup has not been started yet")
	}
	<-g.done
	g.errMutex.Lock()
	defer g.errMutex.Unlock()
	return g.err
}
