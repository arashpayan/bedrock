package main

import (
	"fmt"
	"os"
	"time"

	"ara.sh/iabdaccounting/bedrock/shell"
	"ara.sh/iabdaccounting/bedrock/ui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	zerolog.DurationFieldUnit = time.Millisecond
	logPath, err := shell.ExpandPath("~/bedrock.log")
	if err != nil {
		panic(fmt.Sprintf("expanding log path: %v", err))
	}
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	cw := zerolog.ConsoleWriter{
		Out:        f,
		TimeFormat: "Jan 02 15:04:05",
		NoColor:    true,
	}
	log.Logger = zerolog.New(cw).With().Timestamp().Logger()
	prog := tea.NewProgram(ui.New(), tea.WithAltScreen())
	if _, err := prog.Run(); err != nil {
		fmt.Println("Failed to start program:", err)
		os.Exit(1)
	}
}
