package ui

import (
	"ara.sh/iabdaccounting/bedrock/persistence"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type TransactionsScreenModel struct {
	db   *persistence.Database
	help help.Model
	w    int
	h    int
}

func NewTransactionsScreen(db *persistence.Database, width, height int) TransactionsScreenModel {
	return TransactionsScreenModel{
		db:   db,
		help: help.New(),
		w:    width,
		h:    height,
	}
}

func (m TransactionsScreenModel) isBedrockScreen() {}

func (m TransactionsScreenModel) Init() tea.Cmd {
	return nil
}

func (m TransactionsScreenModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w = msg.Width
		m.help.Width = msg.Width // so the help menu knows when to truncate
		m.h = msg.Height
		return m, nil
	}

	return m, nil
}

func (m TransactionsScreenModel) View() string {
	title := lipgloss.PlaceHorizontal(m.w, lipgloss.Center, "Bahá'ís of Thousand Oaks")
	menu := m.help.ShortHelpView([]key.Binding{})
	menu = lipgloss.PlaceVertical(m.h-4, lipgloss.Bottom, menu)

	view := "\n" + title + "\n" + menu
	return view
}
