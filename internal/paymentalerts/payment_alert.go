package paymentalerts

import (
	"context"
	"fmt"
)

type PaymentEvent struct {
	EventID       string `json:"event_id"`
	PaymentID     string `json:"payment_id"`
	CustomerPhone string `json:"customer_phone"`
	Kind          string `json:"kind"`
	AmountMinor   int64  `json:"amount_minor"`
	Currency      string `json:"currency"`
	RiskScore     int    `json:"risk_score"`
}

type Decision struct {
	Notify bool   `json:"notify"`
	Action string `json:"action"`
	Reason string `json:"reason"`
	Body   string `json:"-"`
}

type SendResult struct {
	MessageID string `json:"message_id"`
}

type Sender interface {
	Send(context.Context, string, string, string) (SendResult, error)
}

type AuditRecord struct {
	EventID   string `json:"event_id"`
	PaymentID string `json:"payment_id"`
	Notify    bool   `json:"notify"`
	Action    string `json:"action"`
	Reason    string `json:"reason"`
	MessageID string `json:"message_id,omitempty"`
}

type Auditor interface {
	Append(AuditRecord) error
}

func Decide(event PaymentEvent) Decision {
	switch {
	case event.Kind == "payment_failed":
		return Decision{true, "review_payment", "payment_failed", "A payment attempt failed. Review it in your account before trying again."}
	case event.Kind == "payment_succeeded" && event.RiskScore >= 80:
		return Decision{true, "secure_account", "high_risk_success", "A payment was approved and needs your attention. Sign in to review account activity."}
	case event.Kind == "payment_succeeded":
		return Decision{true, "record_receipt", "payment_succeeded", fmt.Sprintf("Payment %s was approved. Your receipt is available in your account.", event.PaymentID)}
	default:
		return Decision{false, "record_only", "event_not_customer_visible", ""}
	}
}

func Process(ctx context.Context, event PaymentEvent, sender Sender, auditor Auditor) (AuditRecord, error) {
	decision := Decide(event)
	record := AuditRecord{EventID: event.EventID, PaymentID: event.PaymentID, Notify: decision.Notify, Action: decision.Action, Reason: decision.Reason}
	if decision.Notify {
		result, err := sender.Send(ctx, event.CustomerPhone, decision.Body, "payment-event:"+event.EventID)
		if err != nil {
			return record, err
		}
		record.MessageID = result.MessageID
	}
	if err := auditor.Append(record); err != nil {
		return record, fmt.Errorf("append audit record: %w", err)
	}
	return record, nil
}
