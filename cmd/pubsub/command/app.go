package command

import (
	"flag"
	"strings"

	"github.com/funkygao/gocli"
	"github.com/funkygao/golib/color"
)

type App struct {
	Ui cli.Ui
}

func (this *App) Run(args []string) (exitCode int) {
	var (
		id string
	)
	cmdFlags := flag.NewFlagSet("app", flag.ContinueOnError)
	cmdFlags.Usage = func() { this.Ui.Output(this.Help()) }
	cmdFlags.StringVar(&id, "init", "", "")
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	if id == "" {
		this.Ui.Error(color.Red("-init required"))
		this.Ui.Error(this.Help())
		return 2
	}

	// init
	if err := NewZk(DefaultConfig(id, ZkAddr)).Init(); err != nil {
		this.Ui.Error(color.Red("%v", err))
		return 1
	}

	this.Ui.Output(color.Green("app:%s initialized successfully", id))

	return

}

func (*App) Synopsis() string {
	return "Application initialization"
}

func (*App) Help() string {
	help := `
Usage: pubsub app -init appId

	Application initialization
`
	return strings.TrimSpace(help)
}