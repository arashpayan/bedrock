package ui

import (
	"ara.sh/iabdaccounting/bedrock/persistence"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rs/zerolog/log"
)

type AccountsScreenModel struct {
	controller    *UI
	db            *persistence.Database
	menu          help.Model
	createAccount key.Binding
	w             int
	h             int
}

func NewAccountsScreen(controller *UI, db *persistence.Database, width, height int) AccountsScreenModel {
	return AccountsScreenModel{
		controller: controller,
		db:         db,
		menu:       help.New(),
		w:          width,
		h:          height,

		createAccount: key.NewBinding(
			key.WithKeys("ctrl+a"),
			key.WithHelp("ctrl+a", "Add Account"),
		),
	}
}

func (m AccountsScreenModel) isBedrockScreen() {}

func (m AccountsScreenModel) Init() tea.Cmd {
	return nil
}

func (m AccountsScreenModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	log.Info().Msgf("AccountsScreenModel.Update: %T", msg)
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w = msg.Width
		m.menu.Width = msg.Width
		m.h = msg.Height
		return m, nil
	case tea.KeyMsg:
		switch {
		case msg.Type == tea.KeyCtrlA:
			cas := NewCreateAccountScreen(m.controller, m.db, m.w, m.h)
			return m, switchScreenCmd(cas)
		}
	}

	return m, nil
}

func (m AccountsScreenModel) View() string {
	title := lipgloss.PlaceHorizontal(m.w, lipgloss.Center, "Accounts")
	menu := m.menu.ShortHelpView([]key.Binding{
		m.createAccount,
	})

	menu = lipgloss.PlaceVertical(m.h-1-lipgloss.Height(title)-2, lipgloss.Bottom, menu)

	return "\n" + title + "\n" + menu
}
