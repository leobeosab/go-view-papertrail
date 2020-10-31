package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-runewidth"
	"github.com/muesli/termenv"
)

var term = termenv.ColorProfile()

const (
	headerHeight = 3
	footerHeight = 3
)

type log struct {
	env      string
	severity string
	label    string
	json     string
}

type model struct {
	options  []log
	cursor   int
	selected map[int]struct{}
	ready    bool
	spinner  spinner.Model
	viewport viewport.Model
}

func initialModel() model {
	m := model{
		options: []log{
			log{
				env:      "PROD",
				severity: "ERROR",
				label:    "YO SHITS FUCKED",
				json:     "",
			},
		},
		ready:    false,
		selected: make(map[int]struct{}),
		spinner:  spinner.NewModel(),
	}
	m.spinner.Frames = spinner.Dot
	return m
}

func (m model) Init() tea.Cmd {
	return spinner.Tick(m.spinner)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		verticalMargins := headerHeight + footerHeight

		if !m.ready {
			m.viewport = viewport.Model{Width: msg.Width, Height: msg.Height - verticalMargins}
			m.viewport.YPosition = headerHeight
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height
		}

	case tea.KeyMsg:
		switch msg.String() {

		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.options) {
				m.cursor++
			}

		case "enter", " ":
			_, ok := m.selected[m.cursor]
			if ok {
				delete(m.selected, m.cursor)
			} else {
				m.selected[m.cursor] = struct{}{}
			}
		}

	default:
		if !m.ready {
			var cmd tea.Cmd
			m.spinner, cmd = spinner.Update(msg, m.spinner)
			return m, cmd
		}
	}

	return m, nil
}

func (m model) View() string {

	if !m.ready {
		s := termenv.String(spinner.View(m.spinner)).
			Foreground(term.Color("205")).
			String()

		return fmt.Sprintf("\n\n %s Initializing... press q to quit \n\n", s)
	}

	gapSize := m.viewport.Width - runewidth.StringWidth("╭─────────────╮")

	headerTop := "╭─────────────╮" + strings.Repeat(" ", gapSize)
	headerMid := "│ Paper Trail ├" + strings.Repeat("─", gapSize)
	headerBot := "╰─────────────╯" + strings.Repeat(" ", gapSize)

	header := fmt.Sprintf("%s\n%s\n%s", headerTop, headerMid, headerBot)

	s := header

	for i, choice := range m.options {
		// Is cursor on this choice
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		// Is this choice selected
		checked := " "
		if _, ok := m.selected[i]; ok {
			checked = "x"
		}

		// Render the row
		s += fmt.Sprintf("%s [%s] %s\n", cursor, checked, choice.label)
	}

	// Footer
	s += "\nPress q to quit. \n"

	return s
}

func main() {
	p := tea.NewProgram(initialModel())

	p.EnterAltScreen()
	defer p.ExitAltScreen()

	p.EnableMouseCellMotion()
	defer p.DisableMouseCellMotion()

	if err := p.Start(); err != nil {
		fmt.Printf("Error %v", err)
		os.Exit(1)
	}
}
