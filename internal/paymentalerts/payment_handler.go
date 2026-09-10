package paymentalerts

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

type Handler struct {
	Sender  Sender
	Auditor Auditor
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || r.URL.Path != "/payment-events" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	var event PaymentEvent
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&event); err != nil || !valid(event) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid payment event"})
		return
	}
	record, err := Process(r.Context(), event, h.Sender, h.Auditor)
	if err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.HTTPStatus >= 400 && apiErr.HTTPStatus < 500 {
			writeJSON(w, apiErr.HTTPStatus, map[string]string{"error": apiErr.Code})
			return
		}
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "notification could not be processed"})
		return
	}
	writeJSON(w, http.StatusAccepted, record)
}

func valid(event PaymentEvent) bool {
	return event.EventID != "" && event.PaymentID != "" && strings.HasPrefix(event.CustomerPhone, "+") && event.AmountMinor >= 0 && event.Currency != "" && event.RiskScore >= 0 && event.RiskScore <= 100
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
