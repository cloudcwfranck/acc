package ui

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
)

// Output symbols and colors (following AGENTS.md Section 4.1)
const (
	SymbolSuccess = "✔"
	SymbolWarning = "⚠"
	SymbolFailure = "✖"
	SymbolInfo    = "ℹ"
	SymbolTrust   = "🔐"
)

var (
	// Color styles
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))  // Green
	warningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))  // Yellow
	failureStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))  // Red
	infoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("4"))  // Blue
	trustStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))  // Cyan
	boldStyle    = lipgloss.NewStyle().Bold(true)

	// Global UI settings
	colorEnabled = true
	emojiEnabled = true
)

// SetColorMode sets the color output mode
func SetColorMode(mode string) {
	switch mode {
	case "always":
		colorEnabled = true
	case "never":
		colorEnabled = false
	case "auto":
		// Check if output is a terminal
		if fileInfo, _ := os.Stdout.Stat(); (fileInfo.Mode() & os.ModeCharDevice) != 0 {
			colorEnabled = true
		} else {
			colorEnabled = false
		}
	}
}

// SetEmojiEnabled sets whether emojis should be displayed
func SetEmojiEnabled(enabled bool) {
	emojiEnabled = enabled
}

// FormatSuccess formats a success message
func FormatSuccess(msg string) string {
	symbol := SymbolSuccess
	if !emojiEnabled {
		symbol = "[OK]"
	}
	if colorEnabled {
		return successStyle.Render(symbol) + " " + msg
	}
	return symbol + " " + msg
}

// FormatWarning formats a warning message
func FormatWarning(msg string) string {
	symbol := SymbolWarning
	if !emojiEnabled {
		symbol = "[WARN]"
	}
	if colorEnabled {
		return warningStyle.Render(symbol) + " " + msg
	}
	return symbol + " " + msg
}

// FormatError formats an error message
func FormatError(msg string) string {
	symbol := SymbolFailure
	if !emojiEnabled {
		symbol = "[ERROR]"
	}
	if colorEnabled {
		return failureStyle.Render(symbol) + " " + msg
	}
	return symbol + " " + msg
}

// FormatInfo formats an info message
func FormatInfo(msg string) string {
	symbol := SymbolInfo
	if !emojiEnabled {
		symbol = "[INFO]"
	}
	if colorEnabled {
		return infoStyle.Render(symbol) + " " + msg
	}
	return symbol + " " + msg
}

// FormatTrust formats a trust/security related message
func FormatTrust(msg string) string {
	symbol := SymbolTrust
	if !emojiEnabled {
		symbol = "[TRUST]"
	}
	if colorEnabled {
		return trustStyle.Render(symbol) + " " + msg
	}
	return symbol + " " + msg
}

// PrintSuccess prints a success message to stdout
func PrintSuccess(msg string) {
	fmt.Println(FormatSuccess(msg))
}

// PrintWarning prints a warning message to stdout
func PrintWarning(msg string) {
	fmt.Println(FormatWarning(msg))
}

// PrintError prints an error message to stderr
func PrintError(msg string) {
	fmt.Fprintln(os.Stderr, FormatError(msg))
}

// PrintInfo prints an info message to stdout
func PrintInfo(msg string) {
	fmt.Println(FormatInfo(msg))
}

// PrintTrust prints a trust message to stdout
func PrintTrust(msg string) {
	fmt.Println(FormatTrust(msg))
}
