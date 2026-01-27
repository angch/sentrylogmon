package sources

import "io"

type LogSource interface {
	// Stream returns a reader that streams the log output.
	// It should handle starting the underlying process if necessary.
	Stream() (io.Reader, error)

	// Close stops the log source and releases resources.
	Close() error

	// Name returns the name of the source (e.g. for logging).
	Name() string
}
