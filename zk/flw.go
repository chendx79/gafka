package zk

import (
	"time"

	"github.com/funkygao/golib/color"
)

// Returns {zkHost: outputLines}
func (this *ZkZone) RunZkFourLetterCommand(cmd string) map[string]string {
	servers := this.conf.ZkServers()
	r := make(map[string]string, len(servers))
	for _, server := range servers {
		b, err := zkFourLetterWord(server, cmd, time.Minute)
		if err != nil {
			color.Red("%s: %v", server, err)
		}

		r[server] = string(b)
	}

	return r
}
