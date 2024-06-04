package main

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/uphy/go-revealjs"
	"github.com/urfave/cli"
)

var version string = "dev"

func main() {
	app := cli.NewApp()
	app.Version = version
	app.Usage = "presentation slide generator using reveal.js"
	app.Description = "revealcli is a cli tool to generate the presentation slide using reveal.js."
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
				cli.BoolFlag{
					Name: "config,c",
				},
				cli.BoolFlag{
					Name: "html,t",
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
				return fs.Generate(server.DataDirectory(), &revealjs.GenerateOptions{
					Force:                ctx.Bool("overwrite"),
					GenerateConfig:       ctx.Bool("config"),
					GenerateHTMLTemplate: ctx.Bool("html"),
				})
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
				signal.Notify(signalc, os.Interrupt, syscall.SIGTERM)
				<-signalc
				os.Exit(0)
				return nil
			},
		},
		{
			Name:  "build",
			Usage: "Generate static slide files",
			Action: func(ctx *cli.Context) error {
				server.EmbedHTML = true
				server.EmbedMarkdown = true
				return server.Build()
			},
		},
	}
	if err := app.Run(os.Args); err != nil {
		fmt.Println("failed to execute: ", err)
	}
}
