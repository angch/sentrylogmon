package sources

import (
	"strings"

	"github.com/kballard/go-shellquote"
)

type JournalctlSource struct {
	*CommandSource
}

func NewJournalctlSource(name string, args string) *JournalctlSource {
	// Proper shell-like splitting of args.
	argsSlice, err := shellquote.Split(args)
	if err != nil {
		// Fallback to simple fields if unbalanced quotes
		argsSlice = strings.Fields(args)
	}
	return &JournalctlSource{
		CommandSource: NewCommandSource(name, "journalctl", argsSlice...),
	}
}
