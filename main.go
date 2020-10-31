package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
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
}

var initialModel = model{
	options: []log{
		log{
			env:      "PROD",
			severity: "ERROR",
			label:    "YO SHITS FUCKED",
			json:     "",
		},
	},

	selected: make(map[int]struct{}),
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
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
	}

	return m, nil
}

func (m model) View() string {
	s := "Logs from Papertrail\n\n"

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
	p := tea.NewProgram(initialModel)
	if err := p.Start(); err != nil {
		fmt.Printf("Error %v", err)
		os.Exit(1)
	}
}
