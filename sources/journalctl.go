package sources

import (
	"log"

	"github.com/angch/sentrylogmon/sysstat"
)

type JournalctlSource struct {
	*CommandSource
}

func NewJournalctlSource(name string, args string) *JournalctlSource {
	argsSlice, err := sysstat.SplitCommand(args)
	if err != nil {
		log.Printf("Error parsing journalctl args '%s' for monitor '%s': %v", args, name, err)
		// Fallback to empty args prevents execution with potentially dangerous/wrong args
		argsSlice = []string{}
	}
	return &JournalctlSource{
		CommandSource: NewCommandSource(name, "journalctl", argsSlice...),
	}
}
