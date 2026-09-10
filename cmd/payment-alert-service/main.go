package main

import (
	"log"
	"net/http"
	"os"

	"github.com/infrai-examples/fintech-payment-sms-alerts/internal/paymentalerts"
)

func main() {
	apiKey := os.Getenv("INFRAI_API_KEY")
	if apiKey == "" {
		log.Fatal("INFRAI_API_KEY is required")
	}
	address := os.Getenv("LISTEN_ADDR")
	if address == "" {
		address = ":8080"
	}
	auditPath := os.Getenv("AUDIT_LOG_PATH")
	if auditPath == "" {
		auditPath = "payment-alerts.jsonl"
	}

	handler := paymentalerts.Handler{Sender: paymentalerts.NewSMSClient(apiKey), Auditor: &paymentalerts.JSONLAuditor{Path: auditPath}}
	log.Printf("payment alert service listening on %s", address)
	log.Fatal(http.ListenAndServe(address, handler))
}
