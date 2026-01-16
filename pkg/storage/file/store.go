package file

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/predatorx7/logtopus/pkg/model"
	"github.com/predatorx7/logtopus/pkg/storage"
)

type FileStore struct {
	dir string
}

func NewFileStore(dir string) (*FileStore, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, fmt.Errorf("directory does not exist: %s", dir)
	}
	return &FileStore{dir: dir}, nil
}

func (s *FileStore) Query(ctx context.Context, params storage.QueryParams) ([]model.LogEntry, error) {
	files, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read log directory: %w", err)
	}

	// Sort files by modification time (or name) descending to read newest first
	// Assuming log filenames don't strictly sort chronologically across rotations unless numbered.
	// `ls -t` equivalent.
	fileInfos := make([]os.FileInfo, 0, len(files))
	for _, entry := range files {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".log") {
			info, err := entry.Info()
			if err == nil {
				fileInfos = append(fileInfos, info)
			}
		}
	}

	sort.Slice(fileInfos, func(i, j int) bool {
		return fileInfos[i].ModTime().After(fileInfos[j].ModTime())
	})

	var results []model.LogEntry
	limit := params.Limit
	if limit <= 0 {
		limit = 100
	}

	for _, info := range fileInfos {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		path := filepath.Join(s.dir, info.Name())
		entries, err := s.scanFile(ctx, path, params)
		if err != nil {
			// Log error but continue? or fail? Fails fast for now.
			return nil, fmt.Errorf("failed to scan file %s: %w", info.Name(), err)
		}

		// Since we scan file lines in efficient order (ideally reverse, but simple scan is forward),
		// if we scan forward, we get oldest logs in file first.
		// BUT we iterate files Newest -> Oldest.
		// If we want "Limit 5" most recent, we want the LAST 5 lines of the NEWEST file.

		// To be efficient for "tail", we should ideally read file backward or read all and reverse.
		// Reading full file for large logs is bad.
		// For simplicity/v1: Read lines, parse, filter.
		// Optimization: Read file forward, append to temp buffer, sort/trim?
		// Better: Scan backward. But implementing reliable backward scanner is complex.
		// Compromise for now: Read forward, filter, append.
		// NOTE: This puts oldest logs of the newest file first in 'entries'.
		// We need to handle this order.

		// Let's assume we collect ALL matches from this file (respecting global time limits)
		// and then see if we have enough.

		// Wait, if we iterate files Newest->Oldest, but read lines Oldest->Newest (forward scan),
		// we get: [FileNew: Old->New], [FileOld: Old->New].
		// This is mixed order.

		// Let's reverse 'entries' from this file so they are New->Old.
		reverseEntries(entries)

		results = append(results, entries...)
		if len(results) >= limit {
			results = results[:limit]
			break
		}
	}

	return results, nil
}

func (s *FileStore) scanFile(ctx context.Context, path string, params storage.QueryParams) ([]model.LogEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var matches []model.LogEntry
	scanner := bufio.NewScanner(f)
	// handling large lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	// Context buffers
	ringBuffer := make([]model.LogEntry, 0, params.Before+1)
	afterCount := 0

	for scanner.Scan() {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		line := scanner.Bytes()
		var entry model.LogEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue // skip malformed
		}

		isMatch := match(entry, params)

		if isMatch {
			// Found a match!
			// 1. Flush ring buffer (before context)
			matches = append(matches, ringBuffer...)
			ringBuffer = ringBuffer[:0] // clear buffer

			// 2. Add the clean match
			matches = append(matches, entry)

			// 3. Reset after counter
			afterCount = params.After
		} else {
			// Not a match
			if afterCount > 0 {
				// We are in the "after" context of a previous match
				matches = append(matches, entry)
				afterCount--
			}

			// We also buffer for potential future "before" context
			// Even if it was used as "after" context for a previous match
			if params.Before > 0 {
				if len(ringBuffer) >= params.Before {
					// Slide buffer: drop oldest (index 0)
					ringBuffer = ringBuffer[1:]
				}
				ringBuffer = append(ringBuffer, entry)
			}
		}
	}

	return matches, scanner.Err()
}

func match(entry model.LogEntry, params storage.QueryParams) bool {

	if !params.StartTime.IsZero() && entry.Time.Before(params.StartTime) {
		return false
	}
	if !params.EndTime.IsZero() && entry.Time.After(params.EndTime) {
		return false
	}
	if params.Level != "" && !strings.EqualFold(string(entry.Level), params.Level) {
		return false
	}
	if params.Search != "" && !strings.Contains(strings.ToLower(entry.Message), strings.ToLower(params.Search)) {
		return false
	}
	if params.SessionID != "" && !strings.EqualFold(entry.SessionID, params.SessionID) {
		return false
	}
	if params.ClientID != "" && !strings.EqualFold(entry.ClientID, params.ClientID) {
		return false
	}
	if params.Source != "" && !strings.Contains(strings.ToLower(entry.Source), strings.ToLower(params.Source)) {
		return false
	}
	if params.Error != "" && !strings.Contains(strings.ToLower(entry.Error), strings.ToLower(params.Error)) {
		return false
	}
	return true
}

func reverseEntries(s []model.LogEntry) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}
