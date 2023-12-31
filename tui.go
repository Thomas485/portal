package main

import (
	"fmt"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var styleSelected = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("#7D56F4"))

var styleHelp = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#777777"))

type Screen int

const (
	ScreenList Screen = iota
	ScreenAdd
	ScreenEdit
)

type TuiAdd struct {
	sourceInput textinput.Model
	destInput   textinput.Model
}

type Tui struct {
	config    *Config
	width     int
	height    int
	selection int
	screen    Screen
	AddScreen TuiAdd
}

func NewTui(config *Config) Tui {
	sourceInput := textinput.New()
	sourceInput.Prompt = "Source: "
	destInput := textinput.New()
	destInput.Prompt = "Destination: "
	return Tui{
		config: config,
		width:  0,
		height: 0,
		screen: ScreenList,
		AddScreen: TuiAdd{
			sourceInput: sourceInput,
			destInput:   destInput,
		},
	}
}

func (t *Tui) Start() error {
	p := tea.NewProgram(t, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}

func (t *Tui) Init() tea.Cmd {
	return nil
}

func (t *Tui) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch t.screen {
	case ScreenList:
		return t.UpdateList(msg)
	case ScreenAdd:
		return t.UpdateAdd(msg)
	}
	return t, nil
}

func (t Tui) View() string {
	switch t.screen {
	case ScreenList:
		return t.ViewList()
	case ScreenAdd:
		return t.ViewAdd()
	}
	return ""
}

func (t *Tui) UpdateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return t, tea.Quit
		case "down", "j":
			t.config.mu.RLock()
			if t.selection < len(t.config.Routes)-1 {
				t.selection++
			}
			t.config.mu.RUnlock()
			return t, nil
		case "up", "k":
			if t.selection > 0 {
				t.selection--
			}
			return t, nil
		case "d":
			t.config.mu.Lock()
			t.config.Routes = slices.Delete(t.config.Routes, t.selection, t.selection+1)
			if err := t.config.SaveToFile(t.config.File); err != nil {
				fmt.Println(err)
			}
			t.config.mu.Unlock()
			return t, nil
		case "D":
			t.config.mu.Lock()
			t.config.Routes[t.selection].Active = !t.config.Routes[t.selection].Active
			if err := t.config.SaveToFile(t.config.File); err != nil {
				fmt.Println(err)
			}
			t.config.mu.Unlock()
			return t, nil
		case "a":
			t.screen = ScreenAdd
			t.AddScreen.sourceInput.Reset()
			t.AddScreen.destInput.Reset()
			focus := t.AddScreen.sourceInput.Focus()
			return t, focus
		}
	case tea.WindowSizeMsg:
		t.width = msg.Width
		t.height = msg.Height
	}
	return t, nil
}

func (t Tui) ViewList() string {
	r := fmt.Sprintf("Port: %d", t.config.Port)
	r += "\n\n"
	r += "Routes:\n\n"

	width := 0
	t.config.mu.RLock()
	for _, route := range t.config.Routes {
		width = max(width, len(route.Source))

	}
	t.config.mu.RUnlock()
	width = min(width, t.width/2-8)

	var routeLines []string
	t.config.mu.RLock()
	for _, route := range t.config.Routes {
		var rl string
		if route.Active {
			rl += "A "
		} else {
			rl += "D "
		}
		rl += lipgloss.PlaceHorizontal(width, lipgloss.Left, route.Source)
		rl += " -> "
		rl += lipgloss.PlaceHorizontal(width, lipgloss.Left, route.Dest)
		routeLines = append(routeLines, rl)
	}
	t.config.mu.RUnlock()

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

	r += styleHelp.Render("up   ‑ go up            d ‑ delete route           D - (de)activate route") + "\n"
	r += styleHelp.Render("down ‑ go down          a - add route")

	return r
}

func (t *Tui) UpdateAdd(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return t, tea.Quit
		case "up":
			if t.AddScreen.destInput.Focused() {
				t.AddScreen.destInput.Blur()
				return t, t.AddScreen.sourceInput.Focus()
			}
			return t, nil
		case "down":
			if t.AddScreen.sourceInput.Focused() {
				t.AddScreen.sourceInput.Blur()
				return t, t.AddScreen.destInput.Focus()
			}
			return t, nil
		case "tab":
			if t.AddScreen.sourceInput.Focused() {
				t.AddScreen.sourceInput.Blur()
				return t, t.AddScreen.destInput.Focus()
			} else if t.AddScreen.destInput.Focused() {
				t.AddScreen.destInput.Blur()
				return t, t.AddScreen.sourceInput.Focus()
			}
		case "enter":
			if t.AddScreen.sourceInput.Focused() {
				t.AddScreen.sourceInput.Blur()
				return t, t.AddScreen.destInput.Focus()
			} else if t.AddScreen.destInput.Focused() {
				t.config.mu.Lock()
				t.config.Routes = append(t.config.Routes, Route{
					Source: t.AddScreen.sourceInput.Value(),
					Dest:   t.AddScreen.destInput.Value(),
					Active: false,
				})
				t.config.mu.Unlock()
				err := t.config.SaveToFile(t.config.File)
				if err != nil {
					// TODO: global error
					panic("cant save the configuration")
				}
				t.screen = ScreenList
				return t, nil
			}
			return t, nil
		case "esc":
			t.screen = ScreenList
			return t, nil
		}
	case tea.WindowSizeMsg:
		t.width = msg.Width
		t.height = msg.Height
	}

	var cmd tea.Cmd
	if t.AddScreen.sourceInput.Focused() {
		t.AddScreen.sourceInput, cmd = t.AddScreen.sourceInput.Update(msg)
	} else if t.AddScreen.destInput.Focused() {
		t.AddScreen.destInput, cmd = t.AddScreen.destInput.Update(msg)
	}
	return t, cmd
}

func (t Tui) ViewAdd() string {
	r := "Add:\n"
	r += "\n\n"
	r += t.AddScreen.sourceInput.View() + "\n"
	r += t.AddScreen.destInput.View() + "\n"
	return r
}
