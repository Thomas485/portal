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
			}, {
				Name:        "generate",
				Aliases:     []string{"g"},
				Usage:       "generate a new template config file",
				UsageText:   "portal generate --file config.json",
				Description: "generate a new template config file",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:      "file",
						Value:     "config.json",
						Usage:     "the file for the new configuration",
						TakesFile: true,
					},
				},
				Action: generateConfig,
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

func generateConfig(c *cli.Context) error {
	file := c.String("file")
	config := Config{
		Port: 8080,
		Routes: []Route{
			{Source: "localhost:8080", Dest: "http://localhost:12345", Active: true},
			{Source: "127.0.0.1:8080", Dest: "http://localhost:56789", Active: false},
		},
	}

	err := config.SaveToFile(file)
	if err == nil {
		fmt.Printf("A new configuration template is written into file \"%s\"\n", file)
	}
	return err
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
