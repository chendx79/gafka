package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/funkygao/gafka/cmd/kateway/api"
	"github.com/funkygao/gafka/cmd/kateway/meta"
	"github.com/funkygao/gafka/ctx"
)

var (
	addr    string
	zone    string
	cluster string
	cf      string
)

func init() {
	flag.StringVar(&addr, "addr", "http://localhost:9192", "sub kateway addr")
	flag.StringVar(&zone, "z", "", "zone name")
	flag.StringVar(&cluster, "c", "", "cluster name")
	flag.StringVar(&cf, "cf", "/etc/gafka.cf", "config file")
	flag.Parse()

	if zone == "" || cluster == "" {
		panic("-z and -c required")
	}

	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func main() {
	ctx.LoadConfig(cf)

	// curl -i -XPOST -H "Pubkey: mypubkey" -d '{"cmd":"createTopic", "topic": "hello"}' "http://10.1.82.201:9191/topics/v1/_kateway"
	m := meta.NewZkMetaStore(zone, cluster, 0)

	cf := api.DefaultConfig()
	cf.Debug = true
	c := api.NewClient(cf)
	c.Connect(addr)
	for {
		err := c.Subscribe("v1", "_kateway", "_addtopic", func(statusCode int, cmd []byte) (err error) {
			if statusCode != http.StatusOK {
				log.Printf("err[%d] backoff 10s: %s", statusCode, string(cmd))
				time.Sleep(time.Second * 10)
				return nil
			}

			v := make(map[string]interface{})
			err = json.Unmarshal(cmd, &v)
			if err != nil {
				log.Printf("%s: %v", string(cmd), err)
				return nil
			}

			topic := v["topic"].(string)
			var lines []string
			lines, err = m.ZkCluster().AddTopic(topic, 1, 1) // TODO
			if err != nil {
				log.Printf("add: %v", err)
				time.Sleep(time.Second * 10)
				return nil
			}

			for _, l := range lines {
				log.Printf("add topic[%s]: %s", topic, l)
			}

			return nil
		})

		log.Printf("backoff 10s for: %s", err)
		time.Sleep(time.Second * 10)
	}

}