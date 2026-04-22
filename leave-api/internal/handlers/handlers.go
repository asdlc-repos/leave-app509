package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/asdlc/leave-api/internal/auth"
	"github.com/asdlc/leave-api/internal/store"
)

// Handlers bundles shared dependencies for all HTTP handler methods.
type Handlers struct {
	Store  *store.Store
	Signer *auth.Signer
}

func New(s *store.Store, signer *auth.Signer) *Handlers {
	return &Handlers{Store: s, Signer: signer}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func decodeJSON(r *http.Request, v interface{}) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

func parseDate(s string) (time.Time, error) {
	return time.ParseInLocation("2006-01-02", s, time.UTC)
}

// workingDays counts business days (Mon-Fri) inclusive between start and end.
func workingDays(start, end time.Time) float64 {
	if end.Before(start) {
		return 0
	}
	count := 0
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		if d.Weekday() != time.Saturday && d.Weekday() != time.Sunday {
			count++
		}
	}
	return float64(count)
}

// rangesOverlap returns true when [a1,a2] intersects [b1,b2] (inclusive).
func rangesOverlap(a1, a2, b1, b2 time.Time) bool {
	return !a2.Before(b1) && !b2.Before(a1)
}
