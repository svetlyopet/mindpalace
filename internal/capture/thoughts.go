package capture

import "strings"

const ThoughtsHeading = "## Thoughts"

// AppendThoughts appends a Thoughts section when thoughts is non-empty.
func AppendThoughts(body, thoughts string) string {
	thoughts = strings.TrimSpace(thoughts)
	if thoughts == "" {
		return body
	}
	section := ThoughtsHeading + "\n\n" + thoughts
	body = strings.TrimSpace(body)
	if body == "" {
		return section
	}
	return body + "\n\n" + section
}

// ThoughtsPlainText returns trimmed thoughts for indexing.
func ThoughtsPlainText(thoughts string) string {
	return strings.TrimSpace(thoughts)
}

// MergeIndexText appends thoughts to index plain text when present.
func MergeIndexText(indexText, thoughts string) string {
	thoughts = ThoughtsPlainText(thoughts)
	if thoughts == "" {
		return indexText
	}
	indexText = strings.TrimSpace(indexText)
	if indexText == "" {
		return thoughts
	}
	return indexText + "\n\n" + thoughts
}
