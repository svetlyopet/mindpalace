package commands

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/svetlyopet/mindpalace/internal/capture"
	"github.com/svetlyopet/mindpalace/internal/cli/input"
	"github.com/svetlyopet/mindpalace/internal/clictx"
	"github.com/svetlyopet/mindpalace/internal/fsutil"
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
	add.PersistentFlags().StringSliceVar(&addTags, "tags", nil, "tags (required for url, file, and note with -m or stdin)")
	add.PersistentFlags().StringVar(&addTitle, "title", "", "entry title (required for url, file, and note with -m or stdin)")
	add.PersistentFlags().StringVar(&addType, "type", "", "entry type")
	addURL.Flags().BoolVar(&addFull, "full", false, "save full HTML bundle")
	addNote.Flags().StringVarP(&addMessage, "message", "m", "", "note text")
}

func NewAddNote(rt *clictx.Runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "note",
		Short: "Add a note (-m/stdin need --title and --tags; TTY editor collects body, tags, then title)",
		RunE: func(cmd *cobra.Command, args []string) error {
			body, usedEditor, err := noteBody(rt)
			if err != nil {
				return err
			}
			opts, err := addOptions(cmd)
			if err != nil {
				return err
			}
			trimmedBody := strings.TrimSpace(body)
			defaultTitle := capture.FirstLineTitle(trimmedBody)

			if usedEditor {
				ed, err := rt.Config.EditorCommand()
				if err != nil {
					return err
				}
				if !opts.TagsExplicit {
					suggested, _ := rt.Capturer.SuggestTags(context.Background(), defaultTitle, body)
					opts.Tags, err = noteTagsViaEditor(ed, opts.Tags, suggested)
					if err != nil {
						return err
					}
					opts.TagsExplicit = true
					opts.Prompter = nil
				}
				if strings.TrimSpace(addTitle) == "" {
					title, err := noteTitleViaEditor(ed, defaultTitle)
					if err != nil {
						return err
					}
					opts.Title = title
				} else {
					opts.Title = strings.TrimSpace(addTitle)
				}
			} else {
				if err := requireTitleAndTags(cmd, "mp add note"); err != nil {
					return err
				}
				applyRequiredCaptureFlags(&opts)
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
		Short: "Capture a URL as an article (--title and --tags required)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireTitleAndTags(cmd, "mp add url"); err != nil {
				return err
			}
			opts, err := addOptions(cmd)
			if err != nil {
				return err
			}
			applyRequiredCaptureFlags(&opts)
			opts.FullHTML = addFull
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
		Short: "Import a file (--title and --tags required)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireTitleAndTags(cmd, "mp add file"); err != nil {
				return err
			}
			opts, err := addOptions(cmd)
			if err != nil {
				return err
			}
			applyRequiredCaptureFlags(&opts)
			res, err := rt.Capturer.File(context.Background(), args[0], opts)
			if err != nil {
				return err
			}
			return finishCapture(rt, res)
		},
	}
}

func requireTitleAndTags(cmd *cobra.Command, subcommand string) error {
	if strings.TrimSpace(addTitle) == "" {
		return fmt.Errorf("title is required (use --title with %s)", subcommand)
	}
	if !tagsFlagSet(cmd) || len(addTags) == 0 {
		return fmt.Errorf("tags are required (use --tags with %s)", subcommand)
	}
	return nil
}

func tagsFlagSet(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		if c.Flags().Changed("tags") {
			return true
		}
		if c.PersistentFlags().Changed("tags") {
			return true
		}
	}
	return false
}

func applyRequiredCaptureFlags(opts *capture.Options) {
	opts.Title = strings.TrimSpace(addTitle)
	opts.Tags = addTags
	opts.TagsExplicit = true
	opts.Prompter = nil
}

func addOptions(cmd *cobra.Command) (capture.Options, error) {
	opts := capture.Options{
		Title: addTitle,
		Tags:  addTags,
	}
	opts.TagsExplicit = tagsFlagSet(cmd)
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
	_ = fsutil.CloseFile(f)
	defer fsutil.RemoveBestEffort(path)
	if err := input.RunEditor(ed, path); err != nil {
		return "", false, err
	}
	b, err := readEditorTemp(path)
	if err != nil {
		return "", false, err
	}
	return string(b), true, nil
}

const tagEditorHeader = `# Tags for this note (one per line). Lines starting with # are ignored.

`

const titleEditorHeader = `# Title for this note. Lines starting with # are ignored.

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
		_ = fsutil.CloseFile(f)
		fsutil.RemoveBestEffort(path)
		return nil, err
	}
	_ = fsutil.CloseFile(f)
	defer fsutil.RemoveBestEffort(path)
	if err := input.RunEditor(editor, path); err != nil {
		return nil, err
	}
	raw, err := readEditorTemp(path)
	if err != nil {
		return nil, err
	}
	return capture.ParseTagEditorText(string(raw)), nil
}

func noteTitleViaEditor(editor, defaultHint string) (string, error) {
	f, err := os.CreateTemp("", "mp-title-*.txt")
	if err != nil {
		return "", err
	}
	path := f.Name()
	var b strings.Builder
	b.WriteString(titleEditorHeader)
	if defaultHint != "" {
		b.WriteString(defaultHint)
		b.WriteByte('\n')
	}
	if _, err := f.WriteString(b.String()); err != nil {
		_ = fsutil.CloseFile(f)
		fsutil.RemoveBestEffort(path)
		return "", err
	}
	_ = fsutil.CloseFile(f)
	defer fsutil.RemoveBestEffort(path)
	if err := input.RunEditor(editor, path); err != nil {
		return "", err
	}
	raw, err := readEditorTemp(path)
	if err != nil {
		return "", err
	}
	title := capture.ParseTitleEditorText(string(raw))
	if title == "" {
		return "", fmt.Errorf("title is required")
	}
	return title, nil
}

func readEditorTemp(path string) ([]byte, error) {
	return os.ReadFile(path)
}
