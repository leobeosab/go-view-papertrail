package main

import (
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-runewidth"
	"github.com/muesli/termenv"
	"github.com/tidwall/pretty"
)

var (
	term           = termenv.ColorProfile()
	cursorStyle    = termenv.String("==>").Foreground(term.Color("13")).String()
	logViewHeight  = 0
	jsonViewHeight = 0
	screenHeight   = 0
	screenWidth    = 0
)

const (
	headerHeight = 6
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
				json:     "{\"name\":{\"first\":\"Tom\",\"last\":\"Anderson\"},\"age\":37,\"children\":[\"Sara\",\"Alex\",\"Jack\"],\"fav.movie\":\"Deer Hunter\",\"friends\":[{\"first\":\"Janet\",\"last\":\"Murphy\",\"age\":44}]}",
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

	var (
		cmd           tea.Cmd
		cmds          []tea.Cmd
		updateContent bool
	)

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:

		if !m.ready {
			screenHeight = msg.Height
			screenWidth = msg.Width
			logViewHeight = int(math.Floor(float64(screenHeight-headerHeight) * 0.6))
			m.viewport = viewport.Model{Width: screenWidth, Height: screenHeight - (headerHeight + logViewHeight)}
			m.viewport.YPosition = (headerHeight + logViewHeight + 2)
			m.viewport.HighPerformanceRendering = true
			updateContent = true
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
				updateContent = true
			}

		case "down", "j":
			if m.cursor < len(m.options)-1 {
				m.cursor++
				updateContent = true
			}

		case "-":
			if m.viewport.Height > 15 {
				logViewHeight += 5
				m.viewport.Height -= 5
				m.viewport.YPosition += 5
				updateContent = true
			}

		case "+":
			if m.viewport.Height <= 50 && logViewHeight >= 15 {
				m.viewport.Height += 5
				m.viewport.YPosition -= 5
				logViewHeight -= 5
				updateContent = true
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
			m.spinner, cmd = spinner.Update(msg, m.spinner)
			cmds = append(cmds, cmd)
		}
	}

	if updateContent {
		formattedJSON := pretty.Pretty([]byte(m.options[m.cursor].json))
		m.viewport.SetContent(string(pretty.Color(formattedJSON, nil)))
		cmds = append(cmds, viewport.Sync(m.viewport))
	}

	m.viewport, cmd = viewport.Update(msg, m.viewport)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m model) View() string {

	if !m.ready {
		s := termenv.String(spinner.View(m.spinner)).
			Foreground(term.Color("205")).
			String()

		return fmt.Sprintf("\n\n %s Initializing... press q to quit \n\n", s)
	}

	gapSize := screenWidth - runewidth.StringWidth("╭─────────────╮")

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

	s += strings.Repeat("\n", logViewHeight-lineCount)

	s += jsonHeader(selected)

	s += fmt.Sprintf("\n%s\n", viewport.View(m.viewport))

	return s
}

func jsonHeader(l log) string {

	logGapSize := runewidth.StringWidth(l.display(false)) + 1
	stringGapSize := screenWidth - (runewidth.StringWidth("│ JSON ├─") + logGapSize + 4)

	headerTop := "╭──────╮  ╭─" + strings.Repeat("─", logGapSize) + "╮" + strings.Repeat(" ", stringGapSize)
	headerMid := "│ JSON ├──┤ " + l.display(true) + " ├" + strings.Repeat("─", stringGapSize)
	headerBot := "╰──────╯  ╰─" + strings.Repeat("─", logGapSize) + "╯" + strings.Repeat(" ", stringGapSize)

	jHeader := fmt.Sprintf("%s\n%s\n%s", headerTop, headerMid, headerBot)

	return fmt.Sprintf("%s", jHeader)
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
