package sources

type JournalctlSource struct {
	*CommandSource
}

func NewJournalctlSource(name string, args ...string) *JournalctlSource {
	return &JournalctlSource{
		CommandSource: NewCommandSource(name, "journalctl", args...),
	}
}
