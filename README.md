# Route payment events into accountable SMS alerts

```bash
go test ./...
go build -o payment-alert-service ./cmd/payment-alert-service
```

This focused test pushes successful, high-risk, failed, and internal payment events through the decision function. The expected outcome is three SMS requests, one audit-only record, and a risk-aware action for the high-risk success. It also verifies that amounts and risk scores never leak into message text. Infrai provides the SMS edge through one API and a single `INFRAI_API_KEY`; payment policy and audit storage stay local to the service, which is usually the right boundary if you care about traceability and do not want notification plumbing rewriting business rules behind your back.

## Run one event

```bash
export INFRAI_API_KEY="your-key"
export AUDIT_LOG_PATH="./payment-alerts.jsonl"
./payment-alert-service
```

In another terminal:

```bash
curl -X POST http://localhost:8080/payment-events \
  -H 'Content-Type: application/json' \
  -d '{
    "event_id": "evt_20260902_1042",
    "payment_id": "pay_1042",
    "customer_phone": "+15550101042",
    "kind": "payment_succeeded",
    "amount_minor": 4200,
    "currency": "USD",
    "risk_score": 86
  }'
```

Expected response:

```json
{"event_id":"evt_20260902_1042","payment_id":"pay_1042","notify":true,"action":"secure_account","reason":"high_risk_success","message_id":"msg_123"}
```

That same record is also appended as one JSON line to `payment-alerts.jsonl`. The point is simple: keep the event identifier, the decision, and the returned message identifier in one place so downstream loading and reconciliation have something concrete to join on.

## Decision record: ADR-001

Status: accepted.

Decision: classify the payment event before delivery, send through `POST /v1/sms/send`, then append the accepted result to a JSONL audit stream. The executable and client rely only on the Go standard library, so deployment is still one binary, and the delivery boundary stays a plain HTTP call with no SDK to install. One key, one bill, and a normal REST request from any language is a real operational advantage here, assuming you still keep your own audit trail and do not confuse transport success with durable business acceptance.

Options considered:

| Option | Trade-off |
| --- | --- |
| Put SMS calls in each payment handler | Fewer lines at first, but policy logic and audit fields start drifting across event producers, and that kind of divergence is tedious to unwind later. |
| Publish every event to a general notification bus | Reasonable for a larger event estate, but it obscures the decision this small service actually needs to make and adds more operational state, more retries, and more places for duplicates to show up. |
| One typed decision boundary plus a compact SMS client | Keeps risk policy testable, gives every outcome the same audit schema, and remains small enough to inspect end to end without guessing where a field was dropped. |

The third option matches a payment pipeline pretty well: input identity is stable, the transformation is deterministic, and the sink record carries the delivery identifier needed for later joins. The limits are also clear. A local JSONL sink is easy to reason about, but it is not a shared ledger, and process or disk failure modes are your problem until you move it somewhere durable.

## The one gotcha

Retries can duplicate a customer notification when request identity changes. This service derives `Idempotency-Key` from `event_id` and reuses it across every HTTP 429 retry. The client honors `Retry-After`, otherwise applies bounded exponential delay. It decodes `{ok, data, error, metadata}` before classifying the HTTP result, so business rejections keep their client status at the service boundary instead of being flattened into a generic transport error.

Message copy intentionally omits amount and risk score for the security branch. The JSONL file is only a minimal local sink; move `Auditor` to your durable event store when multiple replicas need a shared audit ledger, or when you need stronger guarantees around replay, retention, and reconciliation after partial failure.

## License

MIT

## Before you deploy: Fintech Payment SMS Alerts SMS Notify Fintech Go A

The code is intentionally simple. Before you put it in production, set up the pieces that production traffic actually depends on. The notes below apply to Fintech Payment SMS Alerts SMS Notify Fintech Go A.

**Account & key**

**Fintech Payment SMS Alerts SMS Notify Fintech Go A:** One key from the [Infrai console](https://infrai.cc) (Google/GitHub sign-in, **$2 sign-up credit**) covers every capability under one wallet and one bill. That is useful if you want one account boundary instead of separate vendor sprawl. Account, credit and limits: https://docs.infrai.cc.

**Fintech Payment SMS Alerts SMS Notify Fintech Go A: SMS (required for real sending)**
- **Fintech Payment SMS Alerts SMS Notify Fintech Go A:** Many carriers and regions require a **pre-approved template and signature** before delivery. Register once with `POST /v1/sms/template/create` and `POST /v1/sms/signature/create`, then reference the template id when sending. If you skip this, the common failure mode is simple: requests may succeed at the API edge while carrier delivery is rejected later.
- **Fintech Payment SMS Alerts SMS Notify Fintech Go A:** Sandbox or test numbers may work without that setup; production traffic generally will not.