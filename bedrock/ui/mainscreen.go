package ui

import (
	"os"

	"ara.sh/iabdaccounting/bedrock/persistence"
	"ara.sh/iabdaccounting/bedrock/shell"
	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rs/zerolog/log"
)

type MainScreenModel struct {
	controller   *UI
	help         help.Model
	open         key.Binding
	enter        key.Binding
	new          key.Binding
	w            int
	h            int
	filePath     textinput.Model
	showFilePath bool
	openFile     bool
	newFile      bool
}

func NewMainScreen(controller *UI) MainScreenModel {
	m := MainScreenModel{
		controller: controller,
		help:       help.New(),
		open: key.NewBinding(
			key.WithKeys("ctrl+o"),
			key.WithHelp("ctrl+o", "Open File"),
		),
		enter: key.NewBinding(
			key.WithKeys("enter"),
		),
		new: key.NewBinding(
			key.WithKeys("ctrl+n"),
			key.WithHelp("ctrl+n", "New file"),
		),
		filePath: textinput.Model{
			Prompt:        "$ ",
			EchoMode:      textinput.EchoNormal,
			Width:         50,
			CharLimit:     2048,
			Cursor:        cursor.New(),
			EchoCharacter: '|',
			KeyMap:        textinput.DefaultKeyMap,
		},
	}
	m.filePath.Focus()
	return m
}

func (m MainScreenModel) isBedrockScreen() {}

func (m MainScreenModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m MainScreenModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	log.Info().Msgf("MainScreenModel.Update: %T, %v", msg, msg)
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.open):
			m.openFile = !m.openFile
			m.showFilePath = m.openFile
			if m.showFilePath {
				m.filePath.SetValue("")
				m.filePath.Focus()
				return m, m.filePath.Cursor.BlinkCmd()
			}
			m.filePath.Blur()
			return m, nil
		case key.Matches(msg, m.new):
			m.newFile = !m.newFile
			m.showFilePath = m.newFile
			if m.showFilePath {
				m.filePath.SetValue("")
				m.filePath.Focus()
				return m, m.filePath.Cursor.BlinkCmd()
			}
			m.filePath.Blur()
			return m, nil
		case key.Matches(msg, m.enter):
			if m.openFile {
				path, err := shell.ExpandPath(m.filePath.Value())
				if err != nil {
					m.filePath.SetValue(err.Error())
					return m, nil
				}
				if _, err := os.Stat(path); err != nil {
					log.Error().Err(err).Msg("os.Stat")
					m.filePath.SetValue(err.Error())
					return m, nil
				}

				db, err := persistence.Open(path)
				if err != nil {
					log.Error().Err(err).Msg("persistence.Open")
					m.filePath.Err = err
					return m, nil
				}
				m.showFilePath = false
				m.openFile = false
				acctScreen := NewAccountsScreen(m.controller, db, m.w, m.h)
				return m, switchScreenCmd(acctScreen)
			}
			if m.newFile {
				path, err := shell.ExpandPath(m.filePath.Value())
				if err != nil {
					m.filePath.Err = err
				}
				if err != nil {
					m.filePath.SetValue(err.Error())
					return m, nil
				}

				db, err := persistence.Open(path)
				if err != nil {
					log.Error().Err(err).Msg("persistence.Open")
					m.filePath.Err = err
					return m, nil
				}
				m.showFilePath = false
				m.newFile = false
				acctScreen := NewAccountsScreen(m.controller, db, m.w, m.h)
				return m, switchScreenCmd(acctScreen)
			}
		}
	case tea.WindowSizeMsg:
		m.w = msg.Width
		m.help.Width = msg.Width // so the help menu knows when to truncate
		m.filePath.Width = msg.Width - len(m.filePath.Prompt)
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

	bindings := []key.Binding{}
	if !m.showFilePath {
		bindings = append(bindings, m.open, m.new)
	}
	menu := lipgloss.PlaceVertical(m.h-1-lipgloss.Height(title)-lipgloss.Height(filePath), lipgloss.Bottom, m.help.ShortHelpView(bindings))

	view := "\n" + title + "\n" + menu
	if m.showFilePath {
		view += "\n"
		view += filePath
	}
	return view
}
