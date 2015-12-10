package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/funkygao/gafka"
	"github.com/funkygao/gafka/cmd/kateway/meta"
	"github.com/julienschmidt/httprouter"
)

func (this *Gateway) helpHandler(w http.ResponseWriter, r *http.Request,
	params httprouter.Params) {
	this.writeKatewayHeader(w)
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(strings.TrimSpace(fmt.Sprintf(`
 GET /ver
 GET /help
 GET /stat
 GET /ping
 GET /clusters

POST /topics/:topic/:ver?key=mykey&async=1
 GET /raw/topics/:appid/:topic/:ver
 GET /topics/:appid/:topic/:ver/:group?limit=1&reset=newest
`))))

}

func (this *Gateway) pingHandler(w http.ResponseWriter, r *http.Request,
	params httprouter.Params) {
	this.writeKatewayHeader(w)
	w.Write([]byte(fmt.Sprintf("ver:%s, build:%s, uptime:%s",
		gafka.Version, gafka.BuildId, time.Since(this.startedAt))))
}

func (this *Gateway) versionHandler(w http.ResponseWriter, r *http.Request,
	params httprouter.Params) {
	this.writeKatewayHeader(w)
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(fmt.Sprintf("%s-%s", gafka.Version, gafka.BuildId)))
}

func (this *Gateway) clustersHandler(w http.ResponseWriter, r *http.Request,
	params httprouter.Params) {
	this.writeKatewayHeader(w)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	b, _ := json.Marshal(meta.Default.Clusters())
	w.Write(b)
}

func (this *Gateway) statHandler(w http.ResponseWriter, r *http.Request,
	params httprouter.Params) {
	this.writeKatewayHeader(w)
	w.Header().Set("Content-Type", "application/json")
	this.guard.Refresh()
	b, _ := json.Marshal(this.guard.cpuStat)
	w.Write(b)
}