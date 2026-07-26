package capture

import (
	"fmt"
	"strings"
)

func validateTitle(title string) error {
	if strings.TrimSpace(title) == "" {
		return fmt.Errorf("title is required")
	}
	return nil
}
