package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/urfave/cli"
)

var styleSelected = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("#7D56F4"))

var styleHelp = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#777777"))

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
	config, err := NewConfigFromFile(c.String("file"))
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
	config, err := NewConfigFromFile(c.String("file"))
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

type Tui struct {
	config    *Config
	width     int
	height    int
	selection int
}

func (t Tui) Init() tea.Cmd {
	return nil
}

func (t Tui) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {

		case "ctrl+c", "q", "esc":
			return t, tea.Quit
		case "down":
			if t.selection < len(t.config.Routes)-1 {
				t.selection++
			}
			return t, nil
		case "up":
			if t.selection > 0 {
				t.selection--
			}
			return t, nil
		case "d":
			t.config.Routes = slices.Delete(t.config.Routes, t.selection, t.selection+1)
			if err := t.config.SaveToFile(t.config.File); err != nil {
				fmt.Println(err)
			}
			return t, nil
		}
	case tea.WindowSizeMsg:
		t.width = msg.Width
		t.height = msg.Height
	}
	return t, nil
}

func (t Tui) View() string {
	r := fmt.Sprintf("Port: %d", t.config.Port)
	r += "\n\n"
	r += "Routes:\n\n"

	width := 0
	for _, route := range t.config.Routes {
		width = max(width, len(route.Source))

	}
	width = min(width, t.width/2-8)

	var routeLines []string
	for _, route := range t.config.Routes {
		rl := lipgloss.PlaceHorizontal(width, lipgloss.Left, route.Source)
		rl += " -> "
		rl += lipgloss.PlaceHorizontal(width, lipgloss.Left, route.Dest)
		routeLines = append(routeLines, rl)
	}

	for i, routeLine := range routeLines {
		var line string
		if t.selection == i {
			line += "  > "
			line += routeLine
			line = styleSelected.Render(line)
			line += "\n"
		} else {
			line += "    "
			line += routeLine
			line += "\n"
		}

		r += line
	}

	r += strings.Repeat("\n", max(t.height-2-strings.Count(r, "\n"), 10))

	r += styleHelp.Render("up   ‑ go up            d ‑ delete route") + "\n"
	r += styleHelp.Render("down ‑ go down")

	return r
}

func (t *Tui) Start() error {
	p := tea.NewProgram(t, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
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

func NewConfigFromFile(filepath string) (*Config, error) {
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
