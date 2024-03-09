package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/urfave/cli/v2"
)

var (
	date      = "not provided (use build.sh instead of 'go build')"
	magic     = "not provided (use build.sh instead of 'go build')"
	startTime = time.Now()

	commands = []*cli.Command{
		{
			Name:        "udp",
			Description: "转发 udp 端口",
			Action:      fwdUDP,
			Flags:       udpFlags,
		},
		{
			Name:   "version",
			Action: version,
		},
	}
)

const (
	bufferSize = 8192
)

func main() {
	app := &cli.App{
		Name:     "gofwd",
		Usage:    "gofwd <https://github.com/davexpro/gofwd>",
		Version:  "v0.1.1",
		Writer:   os.Stdout,
		Flags:    nil,
		Commands: commands,
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Println(err.Error())
	}
}

func version(c *cli.Context) error {
	fmt.Println("date", date)
	fmt.Println("magic", magic)
	return nil
}

func fwdUDP(c *cli.Context) error {
	f := NewUDPFwd(c.String("listen"), c.String("target"), time.Minute*5, c.Bool("verbose"))
	if err := f.prepare(); err != nil {
		log.Println("failed to prepare:", err)
		return err
	}
	return f.run()
}
