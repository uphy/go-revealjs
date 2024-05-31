package main // import "github.com/uphy/revealjs-docker/bootstrap"

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/uphy/go-revealjs"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "dir,d",
			Value: ".",
			Usage: "path to the slide data directory",
		},
	}

	var server *revealjs.RevealJS
	app.Before = func(ctx *cli.Context) error {
		dir := ctx.String("dir")
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			return errors.New("`dir` not exist")
		}
		var err error
		server, err = revealjs.NewRevealJS(dir)
		if err != nil {
			return fmt.Errorf("failed to initialize app: %s", err)
		}
		return nil
	}
	app.Commands = []cli.Command{
		{
			Name:  "init",
			Usage: "Generate config file and slide files",
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name: "overwrite,o",
				},
			},
			ArgsUsage: fmt.Sprintf("[%s]", strings.Join(revealjs.PresetNames, "|")),
			Action: func(ctx *cli.Context) error {
				var name string
				if ctx.NArg() == 0 {
					name = revealjs.PresetNames[0]
				} else {
					name = ctx.Args().First()
				}
				fs, err := revealjs.NewPreset(name)
				if err != nil {
					return err
				}
				return fs.Generate(server.DataDirectory(), ctx.Bool("overwrite"))
			},
		},
		{
			Name:  "start",
			Usage: "Start reveal.js server",
			Action: func(ctx *cli.Context) error {
				if err := server.Start(); err != nil {
					return fmt.Errorf("failed to start server: %s", err)
				}

				signalc := make(chan os.Signal, 1)
				signal.Notify(signalc, os.Interrupt)
				<-signalc
				return nil
			},
		},
		{
			Name: "build",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name: "output,o",
				},
			},
			Action: func(ctx *cli.Context) error {
				var output string
				if ctx.IsSet("output") {
					output = ctx.String("output")
				} else {
					output = filepath.Join(server.DataDirectory(), "build")
				}
				server.EmbedHTML = true
				server.EmbedMarkdown = true
				return server.Build(output)
			},
		},
	}
	if err := app.Run(os.Args); err != nil {
		fmt.Println("failed to execute: ", err)
	}
}
