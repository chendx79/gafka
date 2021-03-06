package kafka

import (
	l "log"
	"os"
	"sync"

	"github.com/Shopify/sarama"
	"github.com/funkygao/gafka/cmd/kateway/store"
	"github.com/funkygao/gafka/ctx"
	"github.com/funkygao/golib/color"
	log "github.com/funkygao/log4go"
)

type subStore struct {
	shutdownCh   chan struct{}
	closedConnCh <-chan string // remote addr
	wg           *sync.WaitGroup
	hostname     string

	subPool *subPool
}

func NewSubStore(wg *sync.WaitGroup, closedConnCh <-chan string, debug bool) *subStore {
	if debug {
		sarama.Logger = l.New(os.Stdout, color.Blue("[Sarama]"),
			l.LstdFlags|l.Lshortfile)
	}

	return &subStore{
		hostname:     ctx.Hostname(),
		wg:           wg,
		shutdownCh:   make(chan struct{}),
		closedConnCh: closedConnCh,
	}
}

func (this *subStore) Name() string {
	return "kafka"
}

func (this *subStore) Start() (err error) {
	this.wg.Add(1)
	defer this.wg.Done()

	this.subPool = newSubPool()

	go func() {
		var remoteAddr string
		for {
			select {
			case <-this.shutdownCh:
				log.Trace("sub store[%s] stopped", this.Name())
				return

			case remoteAddr = <-this.closedConnCh:
				this.subPool.killClient(remoteAddr)

			}
		}
	}()

	return
}

func (this *subStore) Stop() {
	this.subPool.Stop()
	close(this.shutdownCh)
}

func (this *subStore) Fetch(cluster, topic, group, remoteAddr,
	resetOffset string, permitStandby bool) (store.Fetcher, error) {
	cg, err := this.subPool.PickConsumerGroup(cluster, topic, group, remoteAddr, resetOffset, permitStandby)
	if err != nil {
		return nil, err
	}

	return &consumerFetcher{
		ConsumerGroup: cg,
		remoteAddr:    remoteAddr,
		store:         this,
	}, nil
}

func (this *subStore) IsSystemError(err error) bool {
	switch err {
	case store.ErrTooManyConsumers:
		return false

	default:
		return true
	}
}
