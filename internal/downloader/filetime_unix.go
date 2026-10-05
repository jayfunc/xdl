//go:build !windows
// +build !windows

package downloader

import (
	"os"
	"time"
)

func setCreationAndModificationTime(p string, t time.Time) error {
	// Unix systems natively rely on Chtimes for modification and access time.
	return os.Chtimes(p, t, t)
}
