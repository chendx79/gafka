package command

import (
	"flag"
	"fmt"
	"strings"

	"github.com/funkygao/gafka/ctx"
	"github.com/funkygao/gafka/zk"
	"github.com/funkygao/gocli"
	"github.com/funkygao/golib/color"
	"github.com/funkygao/golib/gofmt"
)

type Kguard struct {
	Ui  cli.Ui
	Cmd string

	zone string
}

func (this *Kguard) Run(args []string) (exitCode int) {
	cmdFlags := flag.NewFlagSet("kguard", flag.ContinueOnError)
	cmdFlags.Usage = func() { this.Ui.Output(this.Help()) }
	cmdFlags.StringVar(&this.zone, "z", ctx.ZkDefaultZone(), "")
	if err := cmdFlags.Parse(args); err != nil {
		return 2
	}

	zkzone := zk.NewZkZone(zk.DefaultConfig(this.zone, ctx.ZoneZkAddrs(this.zone)))
	data, stat, err := zkzone.Conn().Get("/" + zk.KguardLeaderPath)
	if err != nil {
		this.Ui.Error(fmt.Sprintf("%s %v", zk.KguardLeaderPath, err.Error()))
		return
	}

	children, _, _ := zkzone.Conn().Children("/" + zk.KguardLeaderPath)

	this.Ui.Output(fmt.Sprintf("%s(out of %d candidates) up: %s",
		color.Green(string(data)),
		len(children),
		gofmt.PrettySince(zk.ZkTimestamp(stat.Mtime).Time())))

	return
}

func (*Kguard) Synopsis() string {
	return "List online kguard instances"
}

func (this *Kguard) Help() string {
	help := fmt.Sprintf(`
Usage: %s kateway -z zone [options]

    List online kguard instances

`, this.Cmd)
	return strings.TrimSpace(help)
}
