package main

import (
	"fmt"
	"os"

	"ara.sh/iabdaccounting/bedrock/ui"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	if _, err := tea.NewProgram(ui.NewMainScreen(), tea.WithAltScreen()).Run(); err != nil {
		fmt.Println("Failed to start program:", err)
		os.Exit(1)
	}
}
