// Package fsperm defines Unix permission bits for Mindpalace vault data on disk.
//
// The vault is a local, user-owned directory. These modes are defense-in-depth
// on multi-user hosts (gosec G301/G306); they do not replace encryption.
package fsperm

const (
	// DirMode is used for vault and entry directories (gosec: ≤0750).
	DirMode = 0o750

	// PrivateFileMode is used for config, entries, index metadata, and capture artifacts (gosec: ≤0600).
	PrivateFileMode = 0o600
)
