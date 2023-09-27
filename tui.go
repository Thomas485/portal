package main

import (
	"fmt"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var styleSelected = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("#7D56F4"))

var styleHelp = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#777777"))

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
