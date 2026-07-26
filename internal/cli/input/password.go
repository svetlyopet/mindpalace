package input

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

func ReadPassword(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

func ReadPasswordConfirm(prompt string) (string, error) {
	pw, err := ReadPassword(prompt)
	if err != nil {
		return "", err
	}
	again, err := ReadPassword("Confirm password: ")
	if err != nil {
		return "", err
	}
	if pw != again {
		return "", fmt.Errorf("passwords do not match")
	}
	if pw == "" {
		return "", fmt.Errorf("password must not be empty")
	}
	return pw, nil
}
