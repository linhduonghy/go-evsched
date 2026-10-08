package domain

// JobData represents the arguments for a job of type T.
type JobData struct {
	JobType  string
	SourceID string
	RawData  map[string]any
}
