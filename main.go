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

var (
	term        = termenv.ColorProfile()
	cursorStyle = termenv.String("==>").Foreground(term.Color("13")).String()
)

const (
	headerHeight   = 3
	jsonViewHeight = 25 // TODO: make this 40% of the height or adjustable
)

type log struct {
	env      string
	severity string
	label    string
	json     string
}

func (l log) display(color bool) string {
	s := ""

	var env string
	var severity string

	if color {
		env = termenv.String(l.env).Foreground(term.Color("14")).String()
		severity = displaySeverity(l.severity)
	} else {
		severity = " " + l.severity + " "
		env = l.env
	}

	s += "[" + env + "]"
	s += " - "
	s += severity
	s += " "
	s += l.label

	return s
}

func displaySeverity(s string) string {
	var background string
	switch s {
	case "error":
		background = "1"
	case "warning":
		background = "11"
	case "info":
		background = "10"
	default:
		background = "15"
	}

	s = " " + s + " "

	return termenv.String(s).Foreground(term.Color("0")).Background(term.Color(background)).String()
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
				env:      "production",
				severity: "error",
				label:    "We had an error in production, I blame Ryan",
				json:     "The JSON",
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
		verticalMargins := headerHeight + jsonViewHeight

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

	var selected log

	lineCount := 0

	for i, choice := range m.options {
		// Is cursor on this choice
		cursor := " "
		if m.cursor == i {
			selected = m.options[i]
			cursor = cursorStyle
		}

		// Is this choice selected
		checked := " "
		if _, ok := m.selected[i]; ok {
			checked = "x"
		}

		// Render the row
		s += fmt.Sprintf("%s [%s] %s\n", cursor, checked, choice.display(true))
		lineCount++
	}

	// Footer
	s += "\nPress q to quit. \n"
	lineCount++

	s += strings.Repeat("\n", m.viewport.Height-lineCount)

	s += viewJSON(m, selected)

	return s
}

func viewJSON(m model, l log) string {

	logGapSize := runewidth.StringWidth(l.display(false)) + 1
	stringGapSize := m.viewport.Width - (runewidth.StringWidth("│ JSON ├─") + logGapSize + 4)

	headerTop := "╭──────╮  ╭─" + strings.Repeat("─", logGapSize) + "╮" + strings.Repeat(" ", stringGapSize)
	headerMid := "│ JSON ├──┤ " + l.display(true) + " ├" + strings.Repeat("─", stringGapSize)
	headerBot := "╰──────╯  ╰─" + strings.Repeat("─", logGapSize) + "╯" + strings.Repeat(" ", stringGapSize)

	jHeader := fmt.Sprintf("%s\n%s\n%s", headerTop, headerMid, headerBot)

	jContent := l.json

	jEnd := strings.Repeat("\n", jsonViewHeight-6)

	return fmt.Sprintf("%s\n%s\n%s", jHeader, jContent, jEnd)
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
