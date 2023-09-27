package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"github.com/urfave/cli"
)

type Route struct {
	Source string `json:"source"`
	Dest   string `json:"dest"`
}

type Config struct {
	File   string `json:"-"`
	Port   int
	Routes []Route
}

var flags = []cli.Flag{
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

func main() {
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
	tui := Tui{
		config: config,
		width:  0,
		height: 0,
	}

	go tui.Start()

	cert, key, err := tlsCerts(c)
	if err != nil {
		return err
	}

	return config.Serve(cert, key)
}

func findDestination(routes []Route, source string) (string, error) {
	for _, route := range routes {
		if route.Source == source {
			return route.Dest, nil
		}
	}
	return "", errors.New("route not found")
}

func (c *Config) Serve(cert, key string) error {
	proxyHandler := func(w http.ResponseWriter, r *http.Request) {
		source := strings.ToLower(r.Host)

		dest, err := findDestination(c.Routes, source)
		if err != nil {
			http.Error(w, "Host not found", http.StatusNotFound)
			return
		}

		target, err := url.Parse(dest)
		if err != nil {
			http.Error(w, "Invalid URL format", http.StatusInternalServerError)
			return
		}

		proxy := httputil.NewSingleHostReverseProxy(target)
		proxy.ServeHTTP(w, r)
	}

	if cert != "" && key != "" {
		return http.ListenAndServeTLS(fmt.Sprintf(":%d", c.Port), cert, key, http.HandlerFunc(proxyHandler))
	} else {
		return http.ListenAndServe(fmt.Sprintf(":%d", c.Port), http.HandlerFunc(proxyHandler))
	}
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

func loadConfig(filepath string) (*Config, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	config.File = filepath

	return &config, nil
}

func (c Config) SaveToFile(filepath string) error {
	data, err := json.MarshalIndent(c, "", "    ")
	if err != nil {
		return err
	}

	err = os.WriteFile(filepath, data, 0644)
	if err != nil {
		return err
	}

	return nil
}
