package ui

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"ara.sh/iabdaccounting/bedrock/datetime"
	"ara.sh/iabdaccounting/bedrock/model"
	"ara.sh/iabdaccounting/bedrock/persistence"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rs/zerolog/log"
)

type saveAccountMsg struct {
	err error
}

type CreateAccountScreenModel struct {
	controller  *UI
	db          *persistence.Database
	status      string
	statusStyle lipgloss.Style
	menu        help.Model
	w           int
	h           int
	save        key.Binding

	focusedIdx  int
	focusables  []textinput.Model
	nameIdx     int
	descIdx     int
	balIdx      int
	openDateIdx int
}

func NewCreateAccountScreen(controller *UI, db *persistence.Database, width, height int) CreateAccountScreenModel {
	name := textinput.New()
	name.Prompt = " Name: "
	name.Width = 32
	name.CharLimit = 32
	name.Placeholder = "e.g. Local Fund"
	name.PromptStyle = name.PromptStyle.Bold(true).Foreground(lipgloss.Color("#0000FF"))
	name.Focus()

	description := textinput.New()
	description.Prompt = " Description: "
	description.Width = 64
	description.CharLimit = 64
	description.Placeholder = "e.g. Checking account at Acme Bank"
	description.PromptStyle = description.PromptStyle.Bold(true).Foreground(lipgloss.Color("#0000FF"))

	balance := textinput.New()
	balance.Prompt = " Balance: "
	balance.Width = 15
	balance.CharLimit = 15
	balance.Placeholder = "$1000"
	balance.PromptStyle = balance.PromptStyle.Bold(true).Foreground(lipgloss.Color("#0000FF"))
	balance.Validate = DollarValidator

	openDate := textinput.New()
	openDate.Prompt = " Open date: "
	openDate.Width = 10
	openDate.CharLimit = 10
	openDate.Placeholder = "YYYY-MM-DD (e.g. 2023-05-01)"
	openDate.PromptStyle = openDate.PromptStyle.Bold(true).Foreground(lipgloss.Color("#0000FF"))

	return CreateAccountScreenModel{
		controller:  controller,
		db:          db,
		menu:        help.New(),
		statusStyle: lipgloss.NewStyle().Background(lipgloss.Color("#FF0000")).Foreground(lipgloss.Color("#FFFFFF")),
		w:           width,
		h:           height,

		focusedIdx: 0,
		focusables: []textinput.Model{
			name, description, balance, openDate,
		},
		nameIdx:     0,
		descIdx:     1,
		balIdx:      2,
		openDateIdx: 3,

		save: key.NewBinding(
			key.WithKeys("ctrl+s"),
			key.WithHelp("ctrl+s", "Save"),
		),
	}
}

func (m CreateAccountScreenModel) isBedrockScreen() {}

func (m CreateAccountScreenModel) Init() tea.Cmd {
	return nil
}

func (m *CreateAccountScreenModel) nextResponder(forward bool) tea.Cmd {
	m.focusables[m.focusedIdx].Blur()
	if forward {
		m.focusedIdx += 1
		m.focusedIdx = m.focusedIdx % len(m.focusables)
	} else {
		if m.focusedIdx == 0 {
			m.focusedIdx = len(m.focusables) - 1
		} else {
			m.focusedIdx -= 1
		}
	}

	return m.focusables[m.focusedIdx].Focus()
}

func (m *CreateAccountScreenModel) saveAccount() tea.Cmd {
	return func() tea.Msg {
		// validate everything
		name := strings.TrimSpace(m.focusables[m.nameIdx].Value())
		if name == "" {
			return saveAccountMsg{err: errors.New("name is required")}
		}

		desc := strings.TrimSpace(m.focusables[m.descIdx].Value())
		if desc == "" {
			return saveAccountMsg{err: errors.New("description is required")}
		}
		balStr := strings.TrimSpace(m.focusables[m.balIdx].Value())
		balStr = strings.TrimPrefix(balStr, "$")
		bal, err := model.NewMoney(balStr)
		if err != nil {
			return saveAccountMsg{err: fmt.Errorf("invalid opening balance: %v", err)}
		}

		dateStr := strings.TrimSpace(m.focusables[m.openDateIdx].Value())
		openDate, err := ParseDayDate(dateStr)
		if err != nil {
			return saveAccountMsg{err: fmt.Errorf("invalid open date: %v", err)}
		}

		_, err = m.db.CreateAccount(context.Background(), model.CreateAccountInput{
			Type:            model.AccountBank,
			Name:            name,
			Description:     desc,
			Denomination:    model.USD,
			StartingBalance: bal,
			StartingDate:    datetime.FromTime(openDate),
		})

		return saveAccountMsg{err: err}
	}
}

func (m CreateAccountScreenModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	log.Info().Msgf("CAS.Update: %T, %v", msg, msg)
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w = msg.Width
		m.h = msg.Height
		m.menu.Width = msg.Width
		return m, nil
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyTab, tea.KeyDown:
			return m, m.nextResponder(true)
		case tea.KeyShiftTab, tea.KeyUp:
			return m, m.nextResponder(false)
		case tea.KeyCtrlS:
			return m, m.saveAccount()
		}
	case saveAccountMsg:
		log.Info().Msg("saveAccountMsg")
		if msg.err != nil {
			m.status = msg.err.Error()
		} else {
			return m, popScreenCmd
		}
		return m, nil
	}

	var cmd tea.Cmd
	f := &m.focusables[m.focusedIdx]
	m.focusables[m.focusedIdx], cmd = f.Update(msg)
	m.status = ""

	return m, cmd
}

func (m CreateAccountScreenModel) View() string {
	title := lipgloss.PlaceHorizontal(m.w, lipgloss.Center, "Create Account")
	view := "\n" + title + "\n\n"

	for _, f := range m.focusables {
		view += f.View() + "\n"
	}
	view += "\n " + m.statusStyle.Render(m.status) + "\n"

	bindings := []key.Binding{m.save}

	menu := lipgloss.PlaceVertical(m.h-lipgloss.Height(view), lipgloss.Bottom, m.menu.ShortHelpView(bindings))
	view += menu

	return view
}
