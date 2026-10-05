package downloader

import (
	"os"
	"path/filepath"
	"strings"
	"time"
	"fmt"
)

func FixLocalFiles(root string, report func(string)) error {
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return fmt.Errorf("Directory not found: %s", root)
	}

	count := 0
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}

		base := info.Name()
		if len(base) >= 15 {
			dateStr := base[:15]
			if dateStr[8] == '_' {
				if t, err := time.Parse("20060102_150405", dateStr); err == nil {
					if strings.HasSuffix(strings.ToLower(base), ".mp4") {
						_ = patchMP4Times(path, t)
					}
					_ = setCreationAndModificationTime(path, t)
					count++
					if count%100 == 0 {
						if report != nil {
							report(fmt.Sprintf("Fixed %d files...", count))
						}
					}
				}
			}
		}
		return nil
	})

	if report != nil {
		report(fmt.Sprintf("Done! Successfully fixed %d files.", count))
	}

	return err
}
