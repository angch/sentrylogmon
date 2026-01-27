package sources

import "strings"

type JournalctlSource struct {
	*CommandSource
}

func NewJournalctlSource(name string, args string) *JournalctlSource {
	// Simple splitting of args.
	argsSlice := strings.Fields(args)
	return &JournalctlSource{
		CommandSource: NewCommandSource(name, "journalctl", argsSlice...),
	}
}
