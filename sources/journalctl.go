package sources

import (
	"github.com/angch/sentrylogmon/sysstat"
)

type JournalctlSource struct {
	*CommandSource
}

func NewJournalctlSource(name string, args string) (*JournalctlSource, error) {
	argsSlice, err := sysstat.SplitCommand(args)
	if err != nil {
		return nil, err
	}
	return &JournalctlSource{
		CommandSource: NewCommandSource(name, "journalctl", argsSlice...),
	}, nil
}
