package httpjson

import (
	"encoding/json"
	"net/http"
)

// DefaultMaxJSONBytes is the default cap for public JSON request bodies.
const DefaultMaxJSONBytes int64 = 1 << 20 // 1 MiB

// LimitBody wraps r.Body with http.MaxBytesReader so oversized payloads return 400
// instead of unbounded buffering.
func LimitBody(w http.ResponseWriter, r *http.Request, n int64) {
	if n <= 0 {
		n = DefaultMaxJSONBytes
	}
	r.Body = http.MaxBytesReader(w, r.Body, n)
}

// Write encodes payload as JSON with the given status.
func Write(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
