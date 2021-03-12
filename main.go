package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/leobeosab/go-view-papertrail/pkg/papertrail"
	"github.com/mattn/go-runewidth"
	"github.com/muesli/reflow/wordwrap"
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
	headerHeight = 8
)

type model struct {
	options    []papertrail.Log
	cursor     int
	selected   map[int]struct{}
	ready      bool
	shouldSpin bool
	spinner    spinner.Model
	viewport   viewport.Model
	search     textinput.Model
	searching  bool
	logOffset  int
	err        error
}

func initialModel() model {
	m := model{
		options:   papertrail.GetLogs(""),
		logOffset: 0,
		ready:     false,
		selected:  make(map[int]struct{}),
		spinner:   spinner.NewModel(),
		search:    textinput.NewModel(),
		searching: false,
	}
	m.spinner.Spinner = spinner.Dot
	return m
}

func (m model) Init() tea.Cmd {
	return spinner.Tick
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
			m.viewport.YPosition = (headerHeight + logViewHeight)
			m.viewport.HighPerformanceRendering = false
			updateContent = true
			m.ready = true
			m.viewport, _ = m.viewport.Update(msg)
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height
		}

	case tea.KeyMsg:
		switch msg.String() {

		case "ctrl+c":
			return m, tea.Quit

		case "up":
			if m.cursor > 0 {
				if m.cursor-m.logOffset == 0 {
					m.logOffset--
				}

				m.cursor--
				updateContent = true
			}

		case "down":
			if m.cursor < len(m.options)-1 {
				m.cursor++

				if m.cursor-m.logOffset >= logViewHeight-1 {
					m.logOffset++
				}

				updateContent = true
			}

		case "j":
			// Yeah these are gross
			if m.searching {
				m.search, cmd = m.search.Update(msg)
				cmds = append(cmds, cmd)
			} else {
				m.viewport.LineDown(1)
			}

		case "k":
			// Yeah these are gross
			if m.searching {
				m.search, cmd = m.search.Update(msg)
				cmds = append(cmds, cmd)
			} else {
				m.viewport.LineUp(1)
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

		case "/":
			m.searching = true
			m.search.Focus()

		case "enter", " ":
			if m.searching {
				m.options = papertrail.GetLogs(m.search.Value())
				m.search.SetValue("")
				m.searching = false
			} else {
				_, ok := m.selected[m.cursor]
				if ok {
					delete(m.selected, m.cursor)
				} else {
					m.selected[m.cursor] = struct{}{}
				}
			}

		default:
			if m.searching {
				m.search, cmd = m.search.Update(msg)
				cmds = append(cmds, cmd)
			}

		}

	default:

		if !m.ready || m.shouldSpin {
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	if updateContent && len(m.options) > 0 && m.options[m.cursor].JSON != "" {
		m.viewport.GotoTop()

		if json.Valid([]byte(m.options[m.cursor].JSON)) {
			formattedJSON := pretty.Pretty([]byte(m.options[m.cursor].JSON))
			coloredJSON := string(pretty.Color(formattedJSON, nil))
			m.viewport.SetContent(wordwrap.String(coloredJSON, screenWidth))
		} else {
			m.viewport.SetContent("Error Loading JSON: ")
		}
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {

	if !m.ready {
		s := termenv.String(m.spinner.View()).
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

	var selected papertrail.Log

	lineCount := 0

	for i, choice := range m.options[m.logOffset : logViewHeight+m.logOffset-1] {

		logIndex := i + m.logOffset

		// Is cursor on this choice
		cursor := " "
		if m.cursor == logIndex {
			selected = m.options[logIndex]
			cursor = cursorStyle
		}

		// Render the row
		s += fmt.Sprintf("%s %s\n", cursor, choice.Display(true, term))

		if i == logViewHeight-1 {
			break
		}

		lineCount++
	}

	s += strings.Repeat("\n", logViewHeight-lineCount)

	s += jsonHeader(selected)

	s += fmt.Sprintf("\n%s\n", m.viewport.View())

	if m.searching {
		s += fmt.Sprintf("\n/%s", m.search.Value())
	}

	return s
}

func jsonHeader(l papertrail.Log) string {

	logGapSize := runewidth.StringWidth(l.Display(false, term)) + 1
	stringGapSize := screenWidth - (runewidth.StringWidth("│ JSON ├─") + logGapSize + 4)

	headerTop := "╭──────╮  ╭─" + strings.Repeat("─", logGapSize) + "╮" + strings.Repeat(" ", stringGapSize)
	headerMid := "│ JSON ├──┤ " + l.Display(true, term) + " ├" + strings.Repeat("─", stringGapSize)
	headerBot := "╰──────╯  ╰─" + strings.Repeat("─", logGapSize) + "╯" + strings.Repeat(" ", stringGapSize)

	jHeader := fmt.Sprintf("%s\n%s\n%s", headerTop, headerMid, headerBot)

	return fmt.Sprintf("%s", jHeader)
}

func main() {
	papertrail.Init()
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
