package main

import (
	"strings"
	"time"

	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"

	"github.com/c0deaddict/ledmatrix-fft/internal/client"
	"github.com/c0deaddict/ledmatrix-fft/internal/server"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	app := &cli.App{
		Name:  "ledmatrix-fft",
		Usage: "Ledmatrix FFT with Spotify track information",
		Flags: []cli.Flag{
			&cli.PathFlag{
				Name:    "socket",
				Value:   "$XDG_RUNTIME_DIR/ledmatrix-fft.sock",
				EnvVars: []string{"LEDMATRIX_FFT_SOCKET"},
			},
		},
		Commands: []*cli.Command{
			&cli.Command{
				Name:  "server",
				Usage: "server",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "always-on",
						Usage:   "always show FFT, by default it is only enabled when music is playing",
						Aliases: []string{"a"},
					},
					&cli.StringFlag{
						Name:    "hostname",
						Usage:   "hostname of the ledmatrix device",
						Value:   "ledmatrix",
						Aliases: []string{"t"},
					},
					&cli.IntFlag{
						Name:    "udp-port",
						Usage:   "UDP port of FFT listener on ledmatrix device",
						Value:   1337,
						Aliases: []string{"p"},
					},
				},
				Action: func(c *cli.Context) error {
					s, err := server.New(
						os.ExpandEnv(c.String("socket")),
						c.String("hostname"),
						c.Int("udp-port"),
						c.Bool("always-on"),
					)
					if err != nil {
						return err
					}
					return s.Run()
				},
			},
			&cli.Command{
				Name:  "client",
				Usage: "client",
				Subcommands: []*cli.Command{
					{
						Name:  "enable",
						Usage: "enable",
						Action: func(c *cli.Context) error {
							return sendCommand(c, "enable")
						},
					},
					{
						Name:  "disable",
						Usage: "disable",
						Action: func(c *cli.Context) error {
							return sendCommand(c, "disable")
						},
					},
					{
						Name:  "message",
						Usage: "message MESSAGE",
						Flags: []cli.Flag{
							&cli.DurationFlag{
								Name:    "show-time",
								Usage:   "show message duration",
								Value:   5 * time.Second,
								Aliases: []string{"t"},
							},
						},
						Action: func(c *cli.Context) error {
							message := strings.Join(c.Args().Slice(), " ")
							showTime := c.Duration("show-time")
							return sendCommand(c, client.MakeMessageCommand(message, showTime))
						},
					},
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal().Err(err).Msg("app")
	}
}

func sendCommand(ctx *cli.Context, command string) error {
	socketPath := os.ExpandEnv(ctx.String("socket"))
	c, err := client.New(socketPath)
	if err != nil {
		return err
	}

	err = c.Send(command)
	c.Close()
	return err
}
