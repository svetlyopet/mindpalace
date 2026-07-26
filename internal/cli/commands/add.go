package commands

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/svetlyopet/mindpalace/internal/capture"
	"github.com/svetlyopet/mindpalace/internal/clictx"
	"github.com/svetlyopet/mindpalace/internal/cli/input"
	"github.com/svetlyopet/mindpalace/internal/library"
	"github.com/svetlyopet/mindpalace/internal/vault"
)

var (
	addTags    []string
	addTitle   string
	addType    string
	addFull    bool
	addMessage string
)

func NewAdd(rt *clictx.Runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "add",
		Short: "Capture a new entry (note, url, file)",
	}
}

func ConfigureAddFlags(add, addNote, addURL *cobra.Command) {
	add.PersistentFlags().StringSliceVar(&addTags, "tags", nil, "tags (repeatable)")
	add.PersistentFlags().StringVar(&addTitle, "title", "", "entry title (required when stdin is not a TTY)")
	add.PersistentFlags().StringVar(&addType, "type", "", "entry type")
	addURL.Flags().BoolVar(&addFull, "full", false, "save full HTML bundle")
	addNote.Flags().StringVarP(&addMessage, "message", "m", "", "note text")
}

func NewAddNote(rt *clictx.Runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "note",
		Short: "Add a note (-m, stdin, or editor; prompts for tags unless --tags is set)",
		RunE: func(cmd *cobra.Command, args []string) error {
			body, usedEditor, err := noteBody(rt)
			if err != nil {
				return err
			}
			opts, err := addOptions(cmd)
			if err != nil {
				return err
			}
			title, err := input.RequireTitleOrPrompt(addTitle, "Title", capture.FirstLineTitle(strings.TrimSpace(body)))
			if err != nil {
				return err
			}
			opts.Title = title
			if usedEditor {
				ed, err := rt.Config.EditorCommand()
				if err != nil {
					return err
				}
				suggested, _ := rt.Capturer.SuggestTags(context.Background(), opts.Title, body)
				opts.Tags, err = noteTagsViaEditor(ed, opts.Tags, suggested)
				if err != nil {
					return err
				}
				opts.TagsExplicit = true
				opts.Prompter = nil
			}
			res, err := rt.Capturer.Note(context.Background(), body, opts)
			if err != nil {
				return err
			}
			return finishCapture(rt, res)
		},
	}
}

func NewAddURL(rt *clictx.Runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "url <link>",
		Short: "Capture a URL as an article (prompts for tags on a TTY unless --tags is set)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts, err := addOptions(cmd)
			if err != nil {
				return err
			}
			opts.FullHTML = addFull
			defaultHint := ""
			if strings.TrimSpace(addTitle) == "" && input.IsInteractive() {
				preview, err := rt.Capturer.PreviewURL(context.Background(), args[0], opts)
				if err != nil {
					return err
				}
				defaultHint = preview.Title
			}
			title, err := input.RequireTitleOrPrompt(addTitle, "Title", defaultHint)
			if err != nil {
				return err
			}
			opts.Title = title
			res, err := rt.Capturer.URL(context.Background(), args[0], opts)
			if err != nil {
				return err
			}
			return finishCapture(rt, res)
		},
	}
}

func NewAddFile(rt *clictx.Runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "file <path>",
		Short: "Import a file (image, text, PDF; text snippets prompt for tags on a TTY)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts, err := addOptions(cmd)
			if err != nil {
				return err
			}
			title, err := input.RequireTitleOrPrompt(addTitle, "Title", filepath.Base(args[0]))
			if err != nil {
				return err
			}
			opts.Title = title
			if isImageFile(args[0]) {
				opts.Prompter = nil
			}
			res, err := rt.Capturer.File(context.Background(), args[0], opts)
			if err != nil {
				return err
			}
			return finishCapture(rt, res)
		},
	}
}

func addOptions(cmd *cobra.Command) (capture.Options, error) {
	opts := capture.Options{
		Title: addTitle,
		Tags:  addTags,
	}
	if cmd != nil {
		opts.TagsExplicit = cmd.Flags().Changed("tags")
	}
	if addType != "" {
		t := vault.Type(addType)
		if !t.Valid() {
			return opts, fmt.Errorf("invalid type %q", addType)
		}
		opts = library.CaptureOptionsFromFields(addTitle, addTags, opts.TagsExplicit, t, addFull)
	} else {
		opts = library.CaptureOptionsFromFields(addTitle, addTags, opts.TagsExplicit, "", addFull)
	}
	if !opts.TagsExplicit && input.IsInteractive() {
		opts.Prompter = input.TerminalTagPrompter{}
	}
	return opts, nil
}

func isImageFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp":
		return true
	default:
		return false
	}
}

func finishCapture(rt *clictx.Runtime, res *capture.Result) error {
	for _, w := range res.Warnings {
		fmt.Fprintln(os.Stderr, "warning:", w)
	}
	if err := rt.Lib.CommitCapture(context.Background(), res); err != nil {
		return err
	}
	fmt.Printf("Created %s  %s\n", res.Entry.ID, res.Entry.Title)
	return nil
}

func noteBody(rt *clictx.Runtime) (string, bool, error) {
	if addMessage != "" {
		return addMessage, false, nil
	}
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", false, err
		}
		if strings.TrimSpace(string(b)) != "" {
			return string(b), false, nil
		}
	}
	ed, err := rt.Config.EditorCommand()
	if err != nil {
		return "", false, err
	}
	f, err := os.CreateTemp("", "mp-note-*.md")
	if err != nil {
		return "", false, err
	}
	path := f.Name()
	f.Close()
	defer os.Remove(path)
	if err := input.RunEditor(ed, path); err != nil {
		return "", false, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", false, err
	}
	return string(b), true, nil
}

const tagEditorHeader = `# Tags for this note (one per line). Lines starting with # are ignored.

`

func noteTagsViaEditor(editor string, initial, suggested []string) ([]string, error) {
	f, err := os.CreateTemp("", "mp-tags-*.txt")
	if err != nil {
		return nil, err
	}
	path := f.Name()
	var b strings.Builder
	b.WriteString(tagEditorHeader)
	if len(suggested) > 0 {
		b.WriteString("# Suggested: ")
		b.WriteString(strings.Join(suggested, ", "))
		b.WriteByte('\n')
	}
	for _, t := range initial {
		b.WriteString(t)
		b.WriteByte('\n')
	}
	if _, err := f.WriteString(b.String()); err != nil {
		f.Close()
		os.Remove(path)
		return nil, err
	}
	f.Close()
	defer os.Remove(path)
	if err := input.RunEditor(editor, path); err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return capture.ParseTagEditorText(string(raw)), nil
}
