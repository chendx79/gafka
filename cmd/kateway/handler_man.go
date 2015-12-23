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
	w.Header().Set(ContentTypeHeader, ContentTypeText)
	w.Write([]byte(strings.TrimSpace(fmt.Sprintf(`
pub server: %s
sub server: %s
man server: %s
dbg server: %s

pub:
POST /topics/:topic/:ver?key=mykey&async=1
POST /ws/topics/:topic/:ver
 GET /raw/topics/:topic/:ver

sub:
 GET /topics/:appid/:topic/:ver/:group?limit=1&reset=newest
 GET /ws/topics/:appid/:topic/:ver/:group
 GET /raw/topics/:appid/:topic/:ver

man:
 GET /ver
 GET /help
 GET /stat
 GET /ping
 GET /clusters

dbg:
 GET /debug/pprof
 GET /debug/vars
`,
		options.pubHttpAddr, options.subHttpAddr,
		options.manHttpAddr, options.debugHttpAddr))))

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
	w.Header().Set(ContentTypeHeader, ContentTypeText)
	w.Write([]byte(fmt.Sprintf("%s-%s", gafka.Version, gafka.BuildId)))
}

func (this *Gateway) clustersHandler(w http.ResponseWriter, r *http.Request,
	params httprouter.Params) {
	this.writeKatewayHeader(w)
	w.Header().Set(ContentTypeHeader, ContentTypeJson)
	w.WriteHeader(http.StatusOK)
	b, _ := json.Marshal(meta.Default.Clusters())
	w.Write(b)
}

func (this *Gateway) statHandler(w http.ResponseWriter, r *http.Request,
	params httprouter.Params) {
	this.writeKatewayHeader(w)
	w.Header().Set(ContentTypeHeader, ContentTypeJson)
	this.guard.Refresh()
	b, _ := json.Marshal(this.guard.cpuStat)
	w.Write(b)
}