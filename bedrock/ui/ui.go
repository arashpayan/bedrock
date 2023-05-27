package ui

import tea "github.com/charmbracelet/bubbletea"

type Screen int

const (
	MainScreen Screen = iota
	CommunityScreen
)

type UI struct {
	Current Screen
	ms      MainScreenModel
}

func New() UI {
	return UI{
		ms: NewMainScreen(),
	}
}

func (m UI) Init() tea.Cmd {
	return nil
}

func (m UI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.Current {
	case MainScreen:
		return m.ms.Update(msg)
	}

	return m, nil
}

func (m UI) View() string {
	switch m.Current {
	case MainScreen:
		return m.ms.View()
	}

	return "Broken world!"
}
