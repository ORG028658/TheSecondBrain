package ui

import "github.com/charmbracelet/lipgloss"

var (
	colorGreen  = lipgloss.Color("#25A065")
	colorPurple = lipgloss.Color("#7D56F4")
	colorGray   = lipgloss.Color("#626262")
	colorYellow = lipgloss.Color("#E8B84B")
	colorRed    = lipgloss.Color("#E06C75")
	colorWhite  = lipgloss.Color("#FFFDF5")
	colorDim    = lipgloss.Color("#4A4A4A")

	headerStyle = lipgloss.NewStyle().
			Background(colorGreen).
			Foreground(colorWhite).
			Bold(true).
			Padding(0, 2)

	headerStatStyle = lipgloss.NewStyle().
			Background(colorGreen).
			Foreground(lipgloss.Color("#D4F0E4")).
			Padding(0, 2)

	userLabelStyle = lipgloss.NewStyle().
			Foreground(colorPurple).
			Bold(true)

	brainLabelStyle = lipgloss.NewStyle().
			Foreground(colorGreen).
			Bold(true)

	systemMsgStyle = lipgloss.NewStyle().
			Foreground(colorGray).
			Italic(true)

	errorMsgStyle = lipgloss.NewStyle().
			Foreground(colorRed)

	refStyle = lipgloss.NewStyle().
			Foreground(colorGray)

	refPathStyle = lipgloss.NewStyle().
			Foreground(colorGreen).
			Underline(true)

	dividerStyle = lipgloss.NewStyle().
			Foreground(colorDim)

	inputPromptStyle = lipgloss.NewStyle().
				Foreground(colorPurple).
				Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(colorGray).
			Italic(true)

	spinnerStyle = lipgloss.NewStyle().
			Foreground(colorYellow)
)
