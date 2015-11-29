package main

import (
	"os"
	"sync"
	"syscall"
	"time"

	_ "expvar"
	_ "net/http/pprof"

	"github.com/funkygao/golib/breaker"
	"github.com/funkygao/golib/ratelimiter"
	"github.com/funkygao/golib/signal"
	log "github.com/funkygao/log4go"
)

// Gateway is a distributed kafka Pub/Sub HTTP endpoint.
type Gateway struct {
	hostname string

	pubServer *pubServer
	subServer *subServer

	routes []route

	shutdownOnce sync.Once
	shutdownCh   chan struct{}
	wg           sync.WaitGroup

	leakyBucket *ratelimiter.LeakyBucket // TODO
	breaker     *breaker.Consecutive
	pubMetrics  *pubMetrics
	subMetrics  *subMetrics

	metaStore           MetaStore
	metaRefreshInterval time.Duration

	pubPool *pubPool
	subPool *subPool
}

func NewGateway(metaRefreshInterval time.Duration) *Gateway {
	this := &Gateway{
		shutdownCh:          make(chan struct{}),
		routes:              make([]route, 0),
		metaStore:           newZkMetaStore(options.zone, options.cluster),
		leakyBucket:         ratelimiter.NewLeakyBucket(1000*60, time.Minute),
		metaRefreshInterval: metaRefreshInterval,
	}

	this.breaker = &breaker.Consecutive{
		FailureAllowance: 10,
		RetryTimeout:     time.Second * 10,
	}
	this.hostname, _ = os.Hostname()

	if options.pubPort > 0 {
		this.pubServer = newPubServer(options.pubPort, options.maxClients,
			&this.wg, this.shutdownCh)
		this.pubMetrics = newPubMetrics(this)
	}
	if options.subPort > 0 {
		this.subServer = newSubServer(options.subPort, options.maxClients,
			&this.wg, this.shutdownCh)
		this.subMetrics = newSubMetrics(this)
	}

	return this
}

func (this *Gateway) Start() (err error) {
	signal.RegisterSignalHandler(syscall.SIGINT, func(sig os.Signal) {
		this.Stop()
	})
	signal.RegisterSignalHandler(syscall.SIGUSR2, func(sig os.Signal) {
		this.Stop()
	})

	this.metaStore.Start()
	log.Info("meta store started")

	this.buildRouting()

	if options.pubPort > 0 {
		this.pubPool = newPubPool(this, this.metaStore.BrokerList())
		go this.pubPool.Start()

		this.pubServer.Start()
	}
	if options.subPort > 0 {
		this.subPool = newSubPool(this)
		go this.subPool.Start()

		this.subServer.Start()
	}

	log.Info("gateway[%s] ready", this.hostname)

	return nil
}

func (this *Gateway) Stop() {
	this.shutdownOnce.Do(func() {
		log.Info("gateway[%s] shutting down", this.hostname)

		close(this.shutdownCh)

		// wait for all components shutdown
		this.wg.Wait()

		this.metaStore.Stop()

		log.Info("gateway[%s] shutdown complete", this.hostname)
	})

}

func (this *Gateway) ServeForever() {
	meteRefreshTicker := time.NewTicker(this.metaRefreshInterval)
	defer meteRefreshTicker.Stop()

	ever := true
	for ever {
		select {
		case <-this.shutdownCh:
			log.Info("gateway[%s] terminated", this.hostname)
			ever = false

		case <-meteRefreshTicker.C:
			this.metaStore.Refresh()
			if this.pubPool != nil {
				this.pubPool.RefreshBrokerList(this.metaStore.BrokerList())
			}
		}
	}

}
