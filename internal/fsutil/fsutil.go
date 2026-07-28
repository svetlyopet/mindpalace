package fsutil

import "os"

// CloseFile closes f and returns its error (for explicit handling on error paths).
func CloseFile(f *os.File) error {
	if f == nil {
		return nil
	}
	return f.Close()
}

// RemoveBestEffort removes path; errors are ignored during cleanup after a primary failure.
func RemoveBestEffort(path string) {
	_ = os.Remove(path)
}
