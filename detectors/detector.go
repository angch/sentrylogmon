package detectors

// Detector is the interface for detecting issues in log lines.
type Detector interface {
	// Detect returns true if the line contains an issue.
	Detect(line []byte) bool
}

// ContextExtractor is an interface for extracting context from log lines.
type ContextExtractor interface {
	// GetContext returns a map of context data from the log line.
	GetContext(line []byte) map[string]interface{}
}

// MessageTransformer is an interface for transforming the log line before sending.
type MessageTransformer interface {
	// TransformMessage returns the transformed message.
	TransformMessage(line []byte) []byte
}

// TimestampExtractor is an interface for extracting timestamp from log lines.
type TimestampExtractor interface {
	// ExtractTimestamp returns the timestamp (unix float), string representation, and success boolean.
	ExtractTimestamp(line []byte) (float64, string, bool)
}
