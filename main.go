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
		Usage:    "gofwd <https://github.com/DavexPro/gofwd>",
		Version:  "v0.1.0",
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
