package sources

type FileSource struct {
	*CommandSource
}

func NewFileSource(name string, path string) *FileSource {
	// tail -F -n 0 follows the file by name, retrying if it disappears (rotation),
	// and starts reading from the current end of file (no history).
	return &FileSource{
		CommandSource: NewCommandSource(name, "tail", "-F", "-n", "0", path),
	}
}
