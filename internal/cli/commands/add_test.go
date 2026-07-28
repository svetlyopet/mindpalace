package commands

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRequireTitleAndTags(t *testing.T) {
	add := &cobra.Command{Use: "add"}
	add.PersistentFlags().StringVar(&addTitle, "title", "", "")
	add.PersistentFlags().StringSliceVar(&addTags, "tags", nil, "")
	child := &cobra.Command{Use: "note"}
	add.AddCommand(child)

	t.Run("missing both", func(t *testing.T) {
		addTitle = ""
		addTags = nil
		if err := requireTitleAndTags(child, "mp add note"); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("missing tags", func(t *testing.T) {
		addTitle = "Title"
		addTags = nil
		if err := requireTitleAndTags(child, "mp add note"); err == nil || !strings.Contains(err.Error(), "tags") {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("missing title", func(t *testing.T) {
		addTitle = "  "
		addTags = []string{"a"}
		if err := requireTitleAndTags(child, "mp add note"); err == nil || !strings.Contains(err.Error(), "title") {
			t.Fatalf("err = %v", err)
		}
	})
}

func TestAddNoteRequiresFlagsWithMessage(t *testing.T) {
	rt, _ := testRuntime(t)
	add := NewAdd(rt)
	addNote := NewAddNote(rt)
	addURL := NewAddURL(rt)
	ConfigureAddFlags(add, addNote, addURL)
	add.AddCommand(addNote)
	add.SilenceUsage = true
	add.SilenceErrors = true

	addMessage = "hello"
	addTitle = ""
	addTags = nil
	err := addNote.RunE(addNote, nil)
	if err == nil || !strings.Contains(err.Error(), "title") {
		t.Fatalf("RunE = %v, want title required", err)
	}
}

func TestAddURLRequiresFlags(t *testing.T) {
	rt, _ := testRuntime(t)
	add := NewAdd(rt)
	addNote := NewAddNote(rt)
	addURL := NewAddURL(rt)
	ConfigureAddFlags(add, addNote, addURL)
	add.AddCommand(addURL)

	addTitle = ""
	addTags = nil
	err := addURL.RunE(addURL, []string{"https://example.com"})
	if err == nil || !strings.Contains(err.Error(), "title") {
		t.Fatalf("RunE = %v", err)
	}
}

func TestAddFileRequiresFlags(t *testing.T) {
	rt, _ := testRuntime(t)
	add := NewAdd(rt)
	addNote := NewAddNote(rt)
	addURL := NewAddURL(rt)
	addFile := NewAddFile(rt)
	ConfigureAddFlags(add, addNote, addURL)
	add.AddCommand(addFile)

	addTitle = ""
	addTags = nil
	err := addFile.RunE(addFile, []string{"doc.txt"})
	if err == nil || !strings.Contains(err.Error(), "title") {
		t.Fatalf("RunE = %v", err)
	}
}
