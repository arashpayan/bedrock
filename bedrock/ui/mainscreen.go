package ui

import (
	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type MainScreenModel struct {
	help     help.Model
	quit     key.Binding
	open     key.Binding
	enter    key.Binding
	w        int
	h        int
	filePath textinput.Model
	showOpen bool
}

func NewMainScreen() MainScreenModel {
	m := MainScreenModel{
		help: help.New(),
		open: key.NewBinding(
			key.WithKeys("ctrl+o"),
			key.WithHelp("ctrl+o", "Open File"),
		),
		quit: key.NewBinding(
			key.WithKeys("esc", "ctrl+q"),
			key.WithHelp("esc/ctrl+q", "Quit"),
		),
		enter: key.NewBinding(
			key.WithKeys("enter"),
		),
		filePath: textinput.Model{
			Prompt:        "File path: ",
			EchoMode:      textinput.EchoNormal,
			Width:         50,
			CharLimit:     128,
			Cursor:        cursor.New(),
			EchoCharacter: '|',
			KeyMap:        textinput.DefaultKeyMap,
		},
	}
	m.filePath.Focus()
	return m
}

func (m MainScreenModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m MainScreenModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.quit):
			return m, tea.Quit
		case key.Matches(msg, m.open):
			m.showOpen = !m.showOpen
			return m, nil
		case key.Matches(msg, m.enter):
			if m.showOpen {

			}
		}
	case tea.WindowSizeMsg:
		m.w = msg.Width
		m.help.Width = msg.Width // so the help menu knows when to truncate
		m.h = msg.Height
		return m, nil
	}

	fp, cmd := m.filePath.Update(msg)
	m.filePath = fp
	return m, cmd
}

func (m MainScreenModel) View() string {
	title := lipgloss.PlaceHorizontal(m.w, lipgloss.Center, "Bedrock")
	filePath := lipgloss.PlaceHorizontal(m.w, lipgloss.Center, m.filePath.View())
	filePathHeight := lipgloss.Height(filePath)

	nlCount := filePathHeight + 3
	bindings := []key.Binding{
		m.quit,
	}
	if !m.showOpen {
		bindings = append(bindings, m.open)
	}
	menu := m.help.ShortHelpView(bindings)

	menu = lipgloss.PlaceHorizontal(m.w, lipgloss.Center, lipgloss.PlaceVertical(m.h-nlCount-lipgloss.Height(title)-lipgloss.Height(filePath), lipgloss.Bottom, menu))

	view := "\n" + title + "\n\n"
	if m.showOpen {
		view += filePath
		view += "\n"
	} else {
		for i := 0; i < filePathHeight; i++ {
			view += "\n"
		}
	}
	view += menu
	return view
}
