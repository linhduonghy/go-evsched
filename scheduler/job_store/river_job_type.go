package job_store

import (
	"encoding/json"

	"github.com/google/uuid"
)

const DefaultJobKind = "default"

// GenericJobArgs Generic args type that satisfies riverjob.JobArgs
type GenericJobArgs struct {
	ManagedJobID uuid.UUID `json:"managed_job_id"`
}

func (a GenericJobArgs) Kind() string {
	return DefaultJobKind
}

type RawJSONJobArgs struct {
	JobKind string          `json:"-"`
	RawJson json.RawMessage `json:"-"`
}

func (a RawJSONJobArgs) Kind() string {
	return a.JobKind
}

// MarshalJSON customize ensures only the inner raw JSON is written to DB
func (a RawJSONJobArgs) MarshalJSON() ([]byte, error) {
	if len(a.RawJson) == 0 {
		return []byte("{}"), nil
	}
	return a.RawJson, nil
}
