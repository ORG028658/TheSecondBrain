package ui

import (
	"math/rand"

	"github.com/charmbracelet/lipgloss"
)

const brainASCII = `
 ██████╗ ██████╗  █████╗ ██╗███╗   ██╗
 ██╔══██╗██╔══██╗██╔══██╗██║████╗  ██║
 ██████╔╝██████╔╝███████║██║██╔██╗ ██║
 ██╔══██╗██╔══██╗██╔══██║██║██║╚██╗██║
 ██████╔╝██║  ██║██║  ██║██║██║ ╚████║
 ╚═════╝ ╚═╝  ╚═╝╚═╝  ╚═╝╚═╝╚═╝  ╚═══╝`

const tagline = "your personal knowledge vault"

var thinkingPhrases = []string{
	"Thinking...",
	"Connecting the dots...",
	"Consulting the wiki...",
	"Weaving through knowledge...",
	"Searching memories...",
	"Synthesizing...",
	"Distilling insights...",
	"Navigating the knowledge graph...",
	"Assembling context...",
	"Cross-referencing...",
	"Reasoning through this...",
	"Pulling it together...",
	"Tracing the threads...",
	"Mapping the concepts...",
}

var analyzingPhrases = []string{
	"Reading carefully...",
	"Extracting knowledge...",
	"Building connections...",
	"Filing concepts...",
	"Mapping entities...",
	"Identifying patterns...",
	"Distilling the essence...",
	"Compiling insights...",
	"Cataloguing ideas...",
	"Weaving the graph...",
}

var greetings = []string{
	"What would you like to know?",
	"Your knowledge vault is ready.",
	"Ask me anything from the wiki.",
	"Ready to explore your knowledge base.",
	"Knowledge at your fingertips.",
}

func randomThinkingPhrase() string {
	return thinkingPhrases[rand.Intn(len(thinkingPhrases))]
}

func randomAnalyzingPhrase() string {
	return analyzingPhrases[rand.Intn(len(analyzingPhrases))]
}

func randomGreeting() string {
	return greetings[rand.Intn(len(greetings))]
}

func renderLogo(width int) string {
	art := lipgloss.NewStyle().
		Foreground(colorGreen).
		Bold(true).
		Render(brainASCII)

	tag := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Italic(true).
		Render("  " + tagline)

	return lipgloss.NewStyle().
		Width(width).
		Align(lipgloss.Center).
		Render(art + "\n" + tag)
}
