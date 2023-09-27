package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli"
)

func main() {
	flags := []cli.Flag{
		&cli.StringFlag{
			Name:      "file",
			Value:     "config.json",
			Usage:     "the configuration file",
			TakesFile: true,
		},
		&cli.StringFlag{
			Name:      "cert",
			Value:     "",
			Usage:     "the tls certificate",
			TakesFile: true,
		},
		&cli.StringFlag{
			Name:      "key",
			Value:     "",
			Usage:     "the public key of the certificate",
			TakesFile: true,
		},
	}

	app := &cli.App{
		Name: "portal",
		Commands: []cli.Command{
			{
				Name:        "serve",
				Aliases:     []string{"s"},
				Usage:       "serve the routes",
				UsageText:   "portal serve --file config.json\n   portal serve --file config.json --cert tls.cert --key tls.key",
				Description: "serve",
				Flags:       flags,
				Action:      serve,
			}, {
				Name:        "interactive",
				Aliases:     []string{"i"},
				Usage:       "serve the routes interactive tui",
				UsageText:   "portal interactive --file config.json\n   portal interactive --file config.json --cert tls.cert --key tls.key",
				Description: "serve with an interactive tui",
				Flags:       flags,
				Action:      interactive,
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
	}
}

func serve(c *cli.Context) error {
	config, err := loadConfig(c.String("file"))
	if err != nil {
		return err
	}

	cert, key, err := tlsCerts(c)
	if err != nil {
		return err
	}

	return config.Serve(cert, key)
}

func interactive(c *cli.Context) error {
	config, err := loadConfig(c.String("file"))
	if err != nil {
		fmt.Println(err)
		return err
	}

	tui := NewTui(config)

	go tui.Start()

	cert, key, err := tlsCerts(c)
	if err != nil {
		return err
	}

	return config.Serve(cert, key)
}

func tlsCerts(c *cli.Context) (string, string, error) {
	var cert = c.String("cert")
	var key = c.String("key")

	// http
	if cert == "" && key == "" {
		return "", "", nil
	}

	if _, err := os.Stat(cert); err != nil {
		return "", "", err
	}
	if _, err := os.Stat(key); err != nil {
		return "", "", err
	}

	// https
	return cert, key, nil
}
