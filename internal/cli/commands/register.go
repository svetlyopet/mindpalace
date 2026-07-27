package commands

import (
	"github.com/spf13/cobra"
	"github.com/svetlyopet/mindpalace/internal/clictx"
	"github.com/svetlyopet/mindpalace/internal/cliformat"
)

// Register attaches the full mp command tree to root.
func Register(root *cobra.Command, rt *clictx.Runtime) {
	search := NewSearch(rt)
	list := NewList(rt)
	cliformat.RegisterSearchFlags(search, list)

	add := NewAdd(rt)
	addNote := NewAddNote(rt)
	addURL := NewAddURL(rt)
	addFile := NewAddFile(rt)
	ConfigureAddFlags(add, addNote, addURL)
	add.AddCommand(addNote, addURL, addFile)

	vault := NewVault(rt)
	vault.AddCommand(
		NewInit(rt),
		NewVaultEncrypt(rt),
		NewVaultDecrypt(rt),
		NewVaultPassword(rt),
		NewVaultUnlock(rt),
		NewVaultLock(rt),
	)

	root.AddCommand(
		add,
		search,
		list,
		NewShow(rt),
		NewOpen(rt),
		NewEdit(rt),
		NewTag(rt),
		NewTags(rt),
		NewDelete(rt),
		NewReindex(rt),
		NewServe(rt),
		vault,
	)
}
