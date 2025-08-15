package handler

type Response struct {
	Message    string      `json:"message"`
	TraceID    string      `json:"traceId,omitempty"`
	Duration   string      `json:"duration,omitempty"`
	Result     interface{} `json:"result,omitempty"`
	Iterations int         `json:"iterations,omitempty"`
}
