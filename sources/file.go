package sources

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type FileSource struct {
	name      string
	path      string
	watcher   *fsnotify.Watcher
	reader    *io.PipeReader
	writer    *io.PipeWriter
	closeChan chan struct{}
	wg        sync.WaitGroup
}

func NewFileSource(name string, path string) *FileSource {
	absPath, err := filepath.Abs(path)
	if err != nil {
		// Fallback to original if Abs fails, though unlikely
		absPath = path
	}
	return &FileSource{
		name:      name,
		path:      absPath,
		closeChan: make(chan struct{}),
	}
}

func (s *FileSource) Name() string {
	return s.name
}

func (s *FileSource) Close() error {
	select {
	case <-s.closeChan:
		return nil
	default:
		close(s.closeChan)
	}

	if s.writer != nil {
		s.writer.Close()
	}

	s.wg.Wait()

	if s.watcher != nil {
		return s.watcher.Close()
	}
	return nil
}

func (s *FileSource) Stream() (io.Reader, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create watcher: %v", err)
	}
	s.watcher = watcher

	pr, pw := io.Pipe()
	s.reader = pr
	s.writer = pw

	s.wg.Add(1)
	go s.run(watcher, pw)

	return pr, nil
}

func (s *FileSource) run(watcher *fsnotify.Watcher, pw *io.PipeWriter) {
	defer s.wg.Done()
	defer pw.Close()

	var file *os.File
	buf := make([]byte, 4096)

	// Helper to safely read from file
	readUntilEOF := func() {
		if file == nil {
			return
		}
		for {
			n, err := file.Read(buf)
			if n > 0 {
				if _, wErr := pw.Write(buf[:n]); wErr != nil {
					return // Pipe closed
				}
			}
			if err == io.EOF {
				return
			}
			if err != nil {
				log.Printf("Error reading file %s: %v", s.path, err)
				return
			}
		}
	}

	openFile := func(seekEnd bool) {
		if file != nil {
			file.Close()
			file = nil
		}
		f, err := os.Open(s.path)
		if err == nil {
			file = f
			if seekEnd {
				file.Seek(0, io.SeekEnd)
			}
			watcher.Add(s.path)
		}
	}

	// Initial setup
	openFile(true)

	parent := filepath.Dir(s.path)
	if err := watcher.Add(parent); err != nil {
		log.Printf("Failed to watch parent directory %s: %v", parent, err)
	}

	// Ticker for retries (e.g. if file didn't exist initially or was deleted and not recreated yet)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.closeChan:
			if file != nil {
				file.Close()
			}
			return

		case <-ticker.C:
			// If file is missing, try to open it
			if file == nil {
				openFile(false) // Start from beginning if it reappeared
				if file != nil {
					readUntilEOF()
				}
			}
			// Ensure parent watch is active (idempotent)
			watcher.Add(parent)

		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			if event.Name == s.path {
				if event.Has(fsnotify.Write) {
					readUntilEOF()
				}
				if event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
					// File rotated. Read remaining content if any.
					readUntilEOF()
					if file != nil {
						file.Close()
						file = nil
					}
					// Wait for creation
				}
				if event.Has(fsnotify.Create) {
					// File created (should come from parent watch, but if we somehow watched s.path before??)
					// Actually, Create event on s.path only happens if we are watching parent.
					openFile(false)
					readUntilEOF()
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Watcher error: %v", err)
		}
	}
}
