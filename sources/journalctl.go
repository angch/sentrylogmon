package sources

import (
	"log"
	"strings"

	"github.com/angch/sentrylogmon/sysstat"
)

type JournalctlSource struct {
	*CommandSource
}

func NewJournalctlSource(name string, args string) *JournalctlSource {
	// Use secure splitter to handle quotes properly
	argsSlice, err := sysstat.SplitCommand(args)
	if err != nil {
		// Fallback to simple split on error, but log it
		log.Printf("Warning: Failed to parse journalctl args securely (error: %v), falling back to simple split: %s", err, args)
		argsSlice = strings.Fields(args)
	}

	return &JournalctlSource{
		CommandSource: NewCommandSource(name, "journalctl", argsSlice...),
	}
}
