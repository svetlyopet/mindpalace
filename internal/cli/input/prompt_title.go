package input

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// PromptLine reads one line from stdin after writing prompt to stderr.
// When defaultHint is non-empty it is shown in the prompt; Enter accepts it.
func PromptLine(prompt, defaultHint string) (string, error) {
	if defaultHint != "" {
		fmt.Fprintf(os.Stderr, "%s [%s]: ", prompt, defaultHint)
	} else {
		fmt.Fprintf(os.Stderr, "%s: ", prompt)
	}
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return "", err
	}
	line = strings.TrimSpace(line)
	if line == "" && defaultHint != "" {
		line = strings.TrimSpace(defaultHint)
	}
	if line == "" {
		return "", fmt.Errorf("title is required")
	}
	return line, nil
}

// RequireTitleOrPrompt returns flagTitle when set; otherwise prompts on an interactive terminal.
func RequireTitleOrPrompt(flagTitle, prompt, defaultHint string) (string, error) {
	if strings.TrimSpace(flagTitle) != "" {
		return strings.TrimSpace(flagTitle), nil
	}
	if !IsInteractive() {
		return "", fmt.Errorf("title is required (use --title)")
	}
	return PromptLine(prompt, defaultHint)
}
