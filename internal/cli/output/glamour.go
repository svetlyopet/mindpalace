package output

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/glamour"
	"golang.org/x/term"
)

func RenderMarkdown(body string, width uint) (string, error) {
	opts := []glamour.TermRendererOption{
		glamour.WithWordWrap(int(width)),
	}
	if os.Getenv("NO_COLOR") != "" {
		opts = append(opts, glamour.WithStandardStyle("dark"))
	} else {
		opts = append(opts, glamour.WithAutoStyle())
	}
	r, err := glamour.NewTermRenderer(opts...)
	if err != nil {
		return "", fmt.Errorf("glamour renderer: %w", err)
	}
	out, err := r.Render(body)
	if err != nil {
		return "", fmt.Errorf("render markdown: %w", err)
	}
	return out, nil
}

func TerminalWidth(fallback uint) uint {
	if cols := os.Getenv("COLUMNS"); cols != "" {
		if n, err := strconv.Atoi(strings.TrimSpace(cols)); err == nil && n > 0 {
			return uint(n)
		}
	}
	if term.IsTerminal(int(os.Stdout.Fd())) {
		w, _, err := term.GetSize(int(os.Stdout.Fd()))
		if err == nil && w > 0 {
			return uint(w)
		}
	}
	return fallback
}
