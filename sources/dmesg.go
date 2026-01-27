package sources

type DmesgSource struct {
	*CommandSource
}

func NewDmesgSource(name string) *DmesgSource {
	return &DmesgSource{
		CommandSource: NewCommandSource(name, "dmesg", "-w"),
	}
}
