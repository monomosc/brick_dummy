/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

//Package static implements The v2 Grpc Resolver API via Static Addresses
package static

import (
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc/resolver"
)

func init() {
	resolver.Register(NewStaticResolverBuilder())
}

func log(s string) {
	if LogEnabled {
		Logger.Info(s)
	}
}

const (
	defaultFreq = time.Minute * 1
)

var (
	//Logger for debug purposes - use os.Stdout e.g.
	Logger     logrus.FieldLogger = logrus.New()
	LogEnabled bool               = true
)

//NewStaticResolverBuilder creates a staticBuilder which is used to factory DNS resolvers.
func NewStaticResolverBuilder() resolver.Builder {
	return &staticBuilder{}
}

type staticBuilder struct {
}

// Build creates and starts a Static resolver that watches the name resolution of the target.
func (b *staticBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {

	log(fmt.Sprintf("Building the Static resolver. target: %v\n", target))

	ctx, cancel := context.WithCancel(context.Background())
	t := strings.Split(target.Endpoint, ",")
	r := &staticResolver{
		targets:        t,
		resolveNowChan: make(chan struct{}, 1),
		ctx:            ctx,
		cc:             cc,
		t:              time.NewTicker(time.Second * 6),
		cancel:         cancel,
	}
	go r.watcher()
	return r, nil
}

// Scheme returns the naming scheme of this resolver builder, which is "static".
func (b *staticBuilder) Scheme() string {
	return "static"
}

type staticResolver struct {
	targets        []string
	resolveNowChan chan struct{}
	ctx            context.Context
	cc             resolver.ClientConn
	t              *time.Ticker
	cancel         context.CancelFunc
}

// ResolveNow invoke an immediate resolution of the target that this dnsResolver watches.
func (r *staticResolver) ResolveNow(opt resolver.ResolveNowOptions) {
	log("ResolveNow called")
	r.resolveNowChan <- struct{}{}
}

// Close closes the dnsResolver.
func (r *staticResolver) Close() {
	r.cancel()
	r.t.Stop()
}

func (r *staticResolver) watcher() {
	for {
		select {
		case <-r.ctx.Done():
			return
		case <-r.t.C:
		case <-r.resolveNowChan:
		}
		result, sc := r.lookup()
		r.cc.NewServiceConfig(sc)
		r.cc.NewAddress(result)
	}
}

func (r *staticResolver) lookup() ([]resolver.Address, string) {

	newAddrs := make([]resolver.Address, 0)
	for _, svc := range r.targets {
		newAddrs = append(newAddrs, resolver.Address{
			Addr:       svc,
			Type:       resolver.Backend,
			ServerName: svc,
		})
	}
	return newAddrs, "{\"loadBalancingPolicy\" : \"round_robin\" }"
}
