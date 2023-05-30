package ui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/rs/zerolog/log"
)

type Screen int

type BedrockScreen interface {
	isBedrockScreen()
	tea.Model
}

const (
	MainScreen Screen = iota
	AccountsScreen
	CreateAccountScreen
	TransactionsScreen
)

type popScreenMsg bool

func popScreenCmd() tea.Msg {
	return popScreenMsg(true)
}

type screenDidAppearMsg bool

type switchScreenMsg BedrockScreen

func switchScreenCmd(m BedrockScreen) tea.Cmd {
	return func() tea.Msg {
		return switchScreenMsg(m)
	}
}

type UI struct {
	quit key.Binding

	stack []BedrockScreen
}

func New() *UI {
	controller := &UI{
		quit: key.NewBinding(
			key.WithKeys("ctrl+q", "ctrl+c"),
			key.WithHelp("ctrl+q/c", "Quit"),
		),
	}
	root := NewMainScreen(controller)
	controller.stack = []BedrockScreen{root}
	return controller
}

func (m *UI) Init() tea.Cmd {
	return nil
}

func (m *UI) popScreen() (tea.Model, tea.Cmd) {
	if len(m.stack) == 1 {
		return m, tea.Quit
	}

	m.stack = m.stack[:len(m.stack)-1]

	return m, nil
}

func (m *UI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	log.Info().Msgf("UI.Update: %T", msg)
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			return m.popScreen()
		case tea.KeyCtrlC:
			fallthrough
		case tea.KeyCtrlQ:
			return m, tea.Quit
		}
	case popScreenMsg:
		log.Info().Msg("received a pop screen message")
		return m.popScreen()
	case switchScreenMsg:
		m.stack = append(m.stack, BedrockScreen(msg))
		log.Info().Msgf("new screen %T", BedrockScreen(msg))
		return m, func() tea.Msg { return screenDidAppearMsg(true) }
	}

	curr := m.stack[len(m.stack)-1]
	newCurr, newCmd := curr.Update(msg)
	m.stack[len(m.stack)-1] = newCurr.(BedrockScreen)
	return m, newCmd
}

func (m *UI) View() string {
	screen := m.stack[len(m.stack)-1]
	return screen.View()
}
