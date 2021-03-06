package kafka

import (
	"sync"
	"time"

	"github.com/funkygao/gafka/cmd/kguard/monitor"
	"github.com/funkygao/gafka/zk"
	"github.com/funkygao/go-metrics"
	log "github.com/funkygao/log4go"
)

func init() {
	monitor.RegisterWatcher("kafka.consumer", func() monitor.Watcher {
		return &WatchConsumers{
			Tick: time.Minute,
		}
	})
}

// WatchConsumers monitors num of kafka online consumer groups over the time.
type WatchConsumers struct {
	Zkzone *zk.ZkZone
	Stop   chan struct{}
	Tick   time.Duration
	Wg     *sync.WaitGroup
}

func (this *WatchConsumers) Init(zkzone *zk.ZkZone, stop chan struct{}, wg *sync.WaitGroup) {
	this.Zkzone = zkzone
	this.Stop = stop
	this.Wg = wg
}

func (this *WatchConsumers) Run() {
	defer this.Wg.Done()

	ticker := time.NewTicker(this.Tick)
	defer ticker.Stop()

	consumerGroupsOnline := metrics.NewRegisteredGauge("consumer.groups.online", nil)
	consumerGroupsOffline := metrics.NewRegisteredGauge("consumer.groups.offline", nil)
	for {
		select {
		case <-this.Stop:
			log.Info("kafka.consumer stopped")
			return

		case <-ticker.C:
			online, offline := this.report()
			consumerGroupsOffline.Update(offline)
			consumerGroupsOnline.Update(online)
		}
	}
}

func (this *WatchConsumers) report() (online, offline int64) {
	this.Zkzone.ForSortedClusters(func(zkcluster *zk.ZkCluster) {
		for _, cgInfo := range zkcluster.ConsumerGroups() {
			if len(cgInfo) > 0 {
				online += 1
			} else {
				offline += 1
			}
		}
	})
	return
}
