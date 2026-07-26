package input

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/svetlyopet/mindpalace/internal/capture"
)

// TerminalTagPrompter prompts on stderr/stdin for comma-separated tags.
type TerminalTagPrompter struct{}

func (TerminalTagPrompter) PromptTags(_ context.Context, _ string, suggested []string) ([]string, error) {
	if len(suggested) > 0 {
		fmt.Fprintf(os.Stderr, "Suggested tags: %s\n", strings.Join(suggested, ", "))
	}
	fmt.Fprint(os.Stderr, "Tags (comma-separated, empty to skip): ")
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return nil, err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, nil
	}
	var parts []string
	for _, p := range strings.Split(line, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			parts = append(parts, p)
		}
	}
	return capture.ParseTagEditorText(strings.Join(parts, "\n")), nil
}

// IsInteractive reports whether stdin and stdout are terminals.
func IsInteractive() bool {
	statIn, errIn := os.Stdin.Stat()
	if errIn != nil {
		return false
	}
	statOut, errOut := os.Stdout.Stat()
	if errOut != nil {
		return false
	}
	inTTY := (statIn.Mode() & os.ModeCharDevice) != 0
	outTTY := (statOut.Mode() & os.ModeCharDevice) != 0
	return inTTY && outTTY
}
