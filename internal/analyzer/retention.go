package analyzer

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type retainedFile struct {
	path    string
	size    int64
	modTime time.Time
}

func PruneAuthorizedPCAP(dir string, maxAge time.Duration, maxBytes int64, now time.Time) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	files := make([]retainedFile, 0, len(entries))
	var total int64
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".authorized.pcap") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if now.Sub(info.ModTime()) > maxAge {
			if err := os.Remove(path); err != nil {
				return err
			}
			continue
		}
		files = append(files, retainedFile{path: path, size: info.Size(), modTime: info.ModTime()})
		total += info.Size()
	}
	sort.Slice(files, func(i, j int) bool { return files[i].modTime.Before(files[j].modTime) })
	for _, file := range files {
		if total <= maxBytes {
			break
		}
		if err := os.Remove(file.path); err != nil {
			return err
		}
		total -= file.size
	}
	return nil
}
