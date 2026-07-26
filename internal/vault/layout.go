package vault

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	legacyMetaDir  = ".mindpalace"
	configFileName = "config.yaml"
	indexDirName   = "index"
	metaFileName   = "meta.json"
)

// MetaDir returns the directory holding config and derived index files.
// Flat layout uses vault root; legacy uses vaultRoot/.mindpalace.
func MetaDir(root string) string {
	root = filepath.Clean(root)
	if fileExists(filepath.Join(root, configFileName)) {
		return root
	}
	legacyCfg := filepath.Join(root, legacyMetaDir, configFileName)
	if fileExists(legacyCfg) {
		return filepath.Join(root, legacyMetaDir)
	}
	return root
}

func ConfigPath(root string) string {
	return filepath.Join(MetaDir(root), configFileName)
}

func IndexDir(root string) string {
	return filepath.Join(MetaDir(root), indexDirName)
}

func MetaPath(root string) string {
	return filepath.Join(MetaDir(root), metaFileName)
}

// SkipWalkDir reports date-walk directories that are not entry trees.
func SkipWalkDir(name string) bool {
	return name == legacyMetaDir || name == indexDirName || name == sessionDirName
}

// IsDerivedPath reports paths under derived metadata (index, config, legacy .mindpalace).
func IsDerivedPath(vaultRoot, path string) bool {
	root := filepath.Clean(vaultRoot)
	path = filepath.Clean(path)
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	meta := MetaDir(root)
	if meta != root {
		metaRel, err := filepath.Rel(meta, path)
		if err == nil && metaRel != ".." && !strings.HasPrefix(metaRel, ".."+string(filepath.Separator)) {
			return true
		}
	}
	switch rel {
	case configFileName, metaFileName, "mapping_version":
		return true
	}
	if rel == indexDirName || strings.HasPrefix(rel, indexDirName+string(filepath.Separator)) {
		return true
	}
	if rel == legacyMetaDir || strings.HasPrefix(rel, legacyMetaDir+string(filepath.Separator)) {
		return true
	}
	return false
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
