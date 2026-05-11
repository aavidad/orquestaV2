package orquestarunqueue

type RunQueueIssueV0 struct {
	Code  string `json:"code"`
	Field string `json:"field,omitempty"`
}
