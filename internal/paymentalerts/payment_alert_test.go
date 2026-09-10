package paymentalerts

import (
	"context"
	"strings"
	"testing"
)

type recordingSender struct {
	calls []sendCall
}

type sendCall struct {
	to, body, key string
}

func (s *recordingSender) Send(_ context.Context, to, body, key string) (SendResult, error) {
	s.calls = append(s.calls, sendCall{to, body, key})
	return SendResult{MessageID: "msg_42"}, nil
}

type memoryAuditor struct {
	records []AuditRecord
}

func (a *memoryAuditor) Append(record AuditRecord) error {
	a.records = append(a.records, record)
	return nil
}

func TestProcessPaymentEvent(t *testing.T) {
	tests := []struct {
		name       string
		event      PaymentEvent
		wantNotify bool
		wantAction string
	}{
		{"ordinary success emits receipt", PaymentEvent{EventID: "evt-1", PaymentID: "pay-1", CustomerPhone: "+15550101", Kind: "payment_succeeded", AmountMinor: 4200, Currency: "USD", RiskScore: 12}, true, "record_receipt"},
		{"high risk success emits security action", PaymentEvent{EventID: "evt-2", PaymentID: "pay-2", CustomerPhone: "+15550102", Kind: "payment_succeeded", AmountMinor: 9900, Currency: "USD", RiskScore: 91}, true, "secure_account"},
		{"failed payment asks for review", PaymentEvent{EventID: "evt-3", PaymentID: "pay-3", CustomerPhone: "+15550103", Kind: "payment_failed", AmountMinor: 8100, Currency: "USD", RiskScore: 25}, true, "review_payment"},
		{"internal event is audit only", PaymentEvent{EventID: "evt-4", PaymentID: "pay-4", CustomerPhone: "+15550104", Kind: "risk_assessed", AmountMinor: 8100, Currency: "USD", RiskScore: 45}, false, "record_only"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender := &recordingSender{}
			auditor := &memoryAuditor{}
			record, err := Process(context.Background(), tt.event, sender, auditor)
			if err != nil {
				t.Fatal(err)
			}
			if record.Notify != tt.wantNotify || record.Action != tt.wantAction {
				t.Fatalf("decision = notify:%v action:%s", record.Notify, record.Action)
			}
			if len(sender.calls) != btoi(tt.wantNotify) {
				t.Fatalf("send calls = %d", len(sender.calls))
			}
			if len(auditor.records) != 1 || auditor.records[0] != record {
				t.Fatalf("audit records = %#v", auditor.records)
			}
			if tt.wantNotify {
				call := sender.calls[0]
				if call.key != "payment-event:"+tt.event.EventID {
					t.Fatalf("idempotency key = %q", call.key)
				}
				if strings.Contains(call.body, "4200") || strings.Contains(call.body, "9900") || strings.Contains(call.body, "8100") || strings.Contains(call.body, "91") {
					t.Fatalf("SMS contains financial or risk detail: %q", call.body)
				}
			}
		})
	}
}

func btoi(value bool) int {
	if value {
		return 1
	}
	return 0
}
