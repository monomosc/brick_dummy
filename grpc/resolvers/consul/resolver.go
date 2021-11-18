/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

//Package consul implements The v2 Grpc Resolver API via consul's Http API
package consul

import (
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/hashicorp/consul/api"
	"golang.org/x/net/context"
	"google.golang.org/grpc/resolver"
)

func init() {
	resolver.Register(NewConsulHTTPResolverBuilder())
}

func log(s string) {
	if LogEnabled {
		io.WriteString(Logger, s+"\n")
	}
}

const (
	defaultFreq = time.Minute * 1
)

var (
	//Logger for debug purposes - use os.Stdout e.g.
	Logger     io.Writer
	LogEnabled bool = false
)

//NewConsulHTTPResolverBuilder creates a dnsBuilder which is used to factory DNS resolvers.
func NewConsulHTTPResolverBuilder() resolver.Builder {
	return &dnsBuilder{freq: defaultFreq}
}

type dnsBuilder struct {
	// frequency of polling the DNS server.
	freq time.Duration
}

// Build creates and starts a DNS resolver that watches the name resolution of the target.
func (b *dnsBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {

	log(fmt.Sprintf("Building the ConsulHttp resolver. target: %v\n", target))

	consul, err := api.NewClient(&api.Config{
		Scheme:  "http",
		Address: target.Authority,
	})
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	r := &consulHTTPResolver{
		consulClient:   consul,
		cc:             cc,
		t:              time.NewTicker(time.Second * 6),
		resolveNowChan: make(chan struct{}, 1),
		ctx:            ctx,
		cancel:         cancel,
		serviceName:    target.Endpoint,
	}
	go r.watcher()
	return r, nil
}

// Scheme returns the naming scheme of this resolver builder, which is "dns".
func (b *dnsBuilder) Scheme() string {
	return "consul-http"
}

type consulHTTPResolver struct {
	consulClient   *api.Client
	cc             resolver.ClientConn
	t              *time.Ticker
	resolveNowChan chan struct{}
	ctx            context.Context
	serviceName    string
	cancel         context.CancelFunc
}

// ResolveNow invoke an immediate resolution of the target that this dnsResolver watches.
func (r *consulHTTPResolver) ResolveNow(opt resolver.ResolveNowOptions) {
	log("ResolveNow called")
	r.resolveNowChan <- struct{}{}
}

// Close closes the dnsResolver.
func (r *consulHTTPResolver) Close() {
	r.cancel()
	r.t.Stop()
}

func (r *consulHTTPResolver) watcher() {
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

func (r *consulHTTPResolver) lookup() ([]resolver.Address, string) {
	log("Starting lookup")
	svcs, _, err := r.consulClient.Health().Service(r.serviceName, "", true, nil)
	if err != nil {
		log(fmt.Sprintf("Could not retrieve Service Info: %v", err))
		return nil, ""
	}

	newAddrs := make([]resolver.Address, 0)
	for _, svc := range svcs {
		newAddrs = append(newAddrs, resolver.Address{
			Addr:       svc.Service.Address + ":" + strconv.Itoa(int(svc.Service.Port)),
			Type:       resolver.Backend,
			ServerName: r.serviceName,
		})
	}
	log(fmt.Sprintf("%v", newAddrs))
	return newAddrs, "{\"loadBalancingPolicy\" : \"round_robin\" }"
}
