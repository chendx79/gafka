package mysql

import (
	"database/sql"
	"time"

	"github.com/funkygao/gafka/ctx"
	"github.com/funkygao/gafka/zk"
	log "github.com/funkygao/log4go"
	_ "github.com/funkygao/mysql"
)

type mysqlStore struct {
	cf     *config
	zkzone *zk.ZkZone

	refreshCh  chan struct{}
	shutdownCh chan struct{}

	allowUnregisteredGroup bool

	// mysql store, initialized on refresh
	// TODO https://github.com/hashicorp/go-memdb
	appClusterMap       map[string]string              // appid:cluster
	appSecretMap        map[string]string              // appid:secret
	appSubMap           map[string]map[string]struct{} // appid:subscribed topics
	appTopicsMap        map[string]map[string]bool     // appid:topics enabled
	appConsumerGroupMap map[string]map[string]struct{} // appid:groups
	shadowQueueMap      map[string]string              // hisappid.topic.ver.myappid:group
	deadPartitionMap    map[string]map[int32]struct{}  // topic:partitionId
}

func New(cf *config) *mysqlStore {
	if cf == nil || cf.Zone == "" {
		panic("empty zone")
	}
	zkAddrs := ctx.ZoneZkAddrs(cf.Zone)
	if len(zkAddrs) == 0 {
		panic("empty zookeeper addr")
	}

	return &mysqlStore{
		cf:                     cf,
		zkzone:                 zk.NewZkZone(zk.DefaultConfig(cf.Zone, zkAddrs)), // TODO session timeout
		shutdownCh:             make(chan struct{}),
		refreshCh:              make(chan struct{}),
		allowUnregisteredGroup: false,
	}
}

func (this *mysqlStore) Name() string {
	return "mysql"
}

func (this *mysqlStore) Start() error {
	if err := this.refreshFromMysql(); err != nil {
		// refuse to start if mysql conn fails
		return err
	}

	// TODO watch KatewayMysqlDsn znode

	go func() {
		ticker := time.NewTicker(this.cf.Refresh)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				this.refreshFromMysql()
				log.Info("manager refreshed from mysql")

			case <-this.refreshCh:
				this.refreshFromMysql()
				log.Info("manager forced to refresh from mysql")

			case <-this.shutdownCh:
				log.Info("mysql manager stopped")
				return
			}
		}
	}()

	return nil
}

func (this *mysqlStore) Stop() {
	close(this.shutdownCh)
}

func (this *mysqlStore) refreshFromMysql() error {
	dsn, err := this.zkzone.KatewayMysqlDsn()
	if err != nil {
		log.Error("mysql manager store fetching mysql dsn: %v", err)
		return err
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Error("mysql manager store: %v", err)
		return err
	}
	defer db.Close()

	// if mysql dies, keep old/stale manager records as it was
	if err = this.fetchApplicationRecords(db); err != nil {
		log.Error("apps: %v", err)
		return err
	}

	if err = this.fetchTopicRecords(db); err != nil {
		log.Error("topics: %v", err)
		return err
	}

	if err = this.fetchSubscribeRecords(db); err != nil {
		log.Error("subs: %v", err)
		return err
	}

	if err = this.fetchAppGroupRecords(db); err != nil {
		log.Error("app groups: %v", err)
		return err
	}

	if err = this.fetchDeadPartitions(db); err != nil {
		log.Error("dead partitions: %v", err)
		return err
	}

	if false {
		if err = this.fetchShadowQueueRecords(db); err != nil {
			log.Error("shadow queues: %v", err)
			return err
		}

	}

	return nil
}

func (this *mysqlStore) shadowKey(hisAppid, topic, ver, myAppid string) string {
	return hisAppid + "." + topic + "." + ver + "." + myAppid
}

func (this *mysqlStore) fetchDeadPartitions(db *sql.DB) error {
	rows, err := db.Query("SELECT KafkaTopic,PartitionId FROM dead_partition")
	if err != nil {
		return err
	}
	defer rows.Close()

	deadPartitionMap := make(map[string]map[int32]struct{})
	var dp deadPartitionRecord
	for rows.Next() {
		err = rows.Scan(&dp.KafkaTopic, &dp.PartitionId)
		if err != nil {
			log.Error("mysql manager store: %v", err)
			continue
		}

		if _, present := deadPartitionMap[dp.KafkaTopic]; !present {
			deadPartitionMap[dp.KafkaTopic] = make(map[int32]struct{})
		}
		deadPartitionMap[dp.KafkaTopic][dp.PartitionId] = struct{}{}
	}

	this.deadPartitionMap = deadPartitionMap
	return nil
}

func (this *mysqlStore) fetchShadowQueueRecords(db *sql.DB) error {
	rows, err := db.Query("SELECT HisAppId,TopicName,Version,MyAppid,GroupName FROM group_shadow WHERE Status=1")
	if err != nil {
		return err
	}
	defer rows.Close()

	shadowQueueMap := make(map[string]string)
	var shadow shadowQueueRecord
	for rows.Next() {
		err = rows.Scan(&shadow.HisAppId, &shadow.TopicName, &shadow.Ver, &shadow.MyAppid, &shadow.Group)
		if err != nil {
			log.Error("mysql manager store: %v", err)
			continue
		}

		shadowQueueMap[this.shadowKey(shadow.HisAppId, shadow.TopicName, shadow.Ver, shadow.MyAppid)] = shadow.Group
	}

	this.shadowQueueMap = shadowQueueMap
	return nil
}

func (this *mysqlStore) fetchAppGroupRecords(db *sql.DB) error {
	rows, err := db.Query("SELECT AppId,GroupName FROM application_group WHERE Status=1")
	if err != nil {
		return err
	}
	defer rows.Close()

	appGroupMap := make(map[string]map[string]struct{})
	var group appConsumerGroupRecord
	for rows.Next() {
		err = rows.Scan(&group.AppId, &group.GroupName)
		if err != nil {
			log.Error("mysql manager store: %v", err)
			continue
		}

		if _, present := appGroupMap[group.AppId]; !present {
			appGroupMap[group.AppId] = make(map[string]struct{})
		}

		appGroupMap[group.AppId][group.GroupName] = struct{}{}
	}

	this.appConsumerGroupMap = appGroupMap
	return nil
}

func (this *mysqlStore) fetchApplicationRecords(db *sql.DB) error {
	rows, err := db.Query("SELECT AppId,Cluster,AppSecret FROM application WHERE Status=1")
	if err != nil {
		return err
	}
	defer rows.Close()

	appClusterMap := make(map[string]string)
	appSecretMap := make(map[string]string)
	var app applicationRecord
	for rows.Next() {
		err = rows.Scan(&app.AppId, &app.Cluster, &app.AppSecret)
		if err != nil {
			log.Error("mysql manager store: %v", err)
			continue
		}

		appSecretMap[app.AppId] = app.AppSecret
		appClusterMap[app.AppId] = app.Cluster
	}

	this.appClusterMap = appClusterMap
	this.appSecretMap = appSecretMap
	return nil
}

func (this *mysqlStore) fetchSubscribeRecords(db *sql.DB) error {
	// FIXME a sub topic t, t disabled, this subscription entry should be disabled too
	rows, err := db.Query("SELECT AppId,TopicName FROM topics_subscriber WHERE Status=1")
	if err != nil {
		return err
	}
	defer rows.Close()

	m := make(map[string]map[string]struct{})
	var app appSubscribeRecord
	for rows.Next() {
		err = rows.Scan(&app.AppId, &app.TopicName)
		if err != nil {
			log.Error("mysql manager store: %v", err)
			continue
		}

		if _, present := m[app.AppId]; !present {
			m[app.AppId] = make(map[string]struct{})
		}

		m[app.AppId][app.TopicName] = struct{}{}
	}

	this.appSubMap = m

	return nil
}

func (this *mysqlStore) fetchTopicRecords(db *sql.DB) error {
	rows, err := db.Query("SELECT AppId,TopicName,Status FROM topics")
	if err != nil {
		return err
	}
	defer rows.Close()

	m := make(map[string]map[string]bool)
	var app appTopicRecord
	for rows.Next() {
		err = rows.Scan(&app.AppId, &app.TopicName, &app.Status)
		if err != nil {
			log.Error("mysql manager store: %v", err)
			continue
		}

		if _, present := m[app.AppId]; !present {
			m[app.AppId] = make(map[string]bool)
		}

		m[app.AppId][app.TopicName] = app.Status == "1"
	}

	this.appTopicsMap = m

	return nil
}
