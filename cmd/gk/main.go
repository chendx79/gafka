package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/funkygao/gafka"
	"github.com/funkygao/gafka/ctx"
	"github.com/funkygao/gocli"
	"github.com/funkygao/log4go"
)

func main() {
	ctx.LoadFromHome()
	setupLogging()

	app := os.Args[0]
	args := os.Args[1:]
	for _, arg := range args {
		if arg == "-v" || arg == "--version" {
			newArgs := make([]string, len(args)+1)
			newArgs[0] = "version"
			copy(newArgs[1:], args)
			args = newArgs
			break
		}

		if arg == "--generate-bash-completion" {
			for name := range commands {
				fmt.Println(name)
			}
			for _, cmd := range ctx.Aliases() {
				fmt.Println(cmd)
			}
			return
		}
	}

	c := cli.NewCLI(app, gafka.Version+"-"+gafka.BuildId+"-"+gafka.BuiltAt)
	c.Args = os.Args[1:]
	if len(os.Args) > 1 {
		// command given, convert alias
		if alias, present := ctx.Alias(os.Args[1]); present {
			cargs := make([]string, 0)
			cargs = append(cargs, strings.Split(alias, " ")...)
			if len(os.Args) > 2 {
				cargs = append(cargs, os.Args[2:]...)
			}
			c.Args = cargs
		}
	}
	c.Commands = commands
	c.HelpFunc = func(m map[string]cli.CommandFactory) string {
		var buf bytes.Buffer
		buf.WriteString(fmt.Sprintf("Unified multi-datacenter multi-cluster kafka swiss-knife management console\n\n"))
		buf.WriteString(cli.BasicHelpFunc(app)(m))
		return buf.String()
	}

	exitCode, err := c.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%+v\n", err)
		os.Exit(1)
	} else if c.IsVersion() {
		os.Exit(0)
	}

	os.Exit(exitCode)
}

func setupLogging() {
	log.SetOutput(ioutil.Discard)

	level := log4go.DEBUG
	switch ctx.LogLevel() {
	case "info":
		level = log4go.INFO

	case "warn":
		level = log4go.WARNING

	case "error":
		level = log4go.ERROR

	case "debug":
		level = log4go.DEBUG

	case "trace":
		level = log4go.TRACE

	case "alarm":
		level = log4go.ALARM
	}

	for _, filter := range log4go.Global {
		filter.Level = level
	}

	log4go.AddFilter("stdout", level, log4go.NewConsoleLogWriter())
}
