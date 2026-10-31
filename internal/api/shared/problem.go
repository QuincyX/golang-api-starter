package shared

import (
	"encoding/json"
	"net/http"

	"github.com/example/homebrew-api-starter/internal/platform/reqid"
)

// Problem is the RFC 9457 error body. Message is kept as an alias of Detail so
// the existing Vue client can keep reading `message` without a breaking change.
type Problem struct {
	Type      string `json:"type"`
	Title     string `json:"title"`
	Status    int    `json:"status"`
	Detail    string `json:"detail,omitempty"`
	Message   string `json:"message,omitempty"`
	Code      string `json:"code,omitempty"`
	RequestID string `json:"requestId,omitempty"`
}

func WriteProblem(w http.ResponseWriter, r *http.Request, status int, code, detail string) {
	if status == 0 {
		status = http.StatusInternalServerError
	}
	problem := Problem{
		Type:    "about:blank",
		Title:   http.StatusText(status),
		Status:  status,
		Detail:  detail,
		Message: detail,
		Code:    code,
	}
	if r != nil {
		problem.RequestID = reqid.From(r.Context())
	}

	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(problem)
}
