package ui

import (
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
	w        int
	h        int
	filePath textinput.Model
}

func NewMainScreen() MainScreenModel {
	return MainScreenModel{
		help: help.New(),
		open: key.NewBinding(
			key.WithKeys("ctrl+o"),
			key.WithHelp("ctrl+o", "Open File"),
		),
		quit: key.NewBinding(
			key.WithKeys("esc", "ctrl+q"),
			key.WithHelp("esc/ctrl+q", "Quit"),
		),
		filePath: textinput.Model{
			Prompt:      "File path",
			Placeholder: "file.bedrock",
			EchoMode:    textinput.EchoNormal,
			Width:       20,
		},
	}
}

func (m MainScreenModel) Init() tea.Cmd {
	return nil
}

func (m MainScreenModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.quit):
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.w = msg.Width
		m.help.Width = msg.Width // so the help menu knows when to truncate
		m.h = msg.Height
		return m, nil
	}

	return m, nil
}

func (m MainScreenModel) View() string {
	title := lipgloss.PlaceHorizontal(m.w, lipgloss.Center, "Hello, world")
	help := m.help.ShortHelpView([]key.Binding{
		// quit
		m.quit, m.open,
	})
	ti := lipgloss.PlaceHorizontal(m.w, lipgloss.Center, m.filePath.View())
	menu := lipgloss.PlaceHorizontal(m.w, lipgloss.Center, lipgloss.PlaceVertical(m.h-lipgloss.Height(title)-lipgloss.Height(ti), lipgloss.Bottom, help))

	return lipgloss.JoinVertical(lipgloss.Center, title, menu)
}
