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
	// Use SplitCommand to handle quoted arguments correctly
	argsSlice, err := sysstat.SplitCommand(args)
	if err != nil {
		log.Printf("Warning: Failed to parse journalctl args (len=%d) using SplitCommand: %v. Falling back to strings.Fields", len(args), err)
		argsSlice = strings.Fields(args)
	}
	return &JournalctlSource{
		CommandSource: NewCommandSource(name, "journalctl", argsSlice...),
	}
}
