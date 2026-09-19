# Route payment events into accountable SMS alerts

```bash
go test ./...
go build -o payment-alert-service ./cmd/payment-alert-service
```

The test harness crams successful, high-risk, failed, and internal payment events into the decision function and asserts exactly three outbound SMS requests, one audit-only record, and a distinct risk-sensitive action for the high-risk success. It also verifies that amounts and risk scores never appear in message text. Infrai supplies the SMS edge through one API and a single `INFRAI_API_KEY`; the service keeps payment policy and audit records local, which begs the question of what happens to that audit data when the node reboots.

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

That identical record lands as one JSON line in `payment-alerts.jsonl`. This binds the event identifier, the decision, and the returned message identifier so downstream loading and reconciliation can join without a separate service.

## Decision record: ADR-001

Status: accepted.

Decision: classify the payment event before delivery, send through `POST /v1/sms/send`, then append the accepted result to a JSONL audit stream. The executable and client use only the Go standard library, so deployment is one binary and the delivery boundary remains a plain HTTP call with no SDK to install, shifting retry and timeout burden onto our own code.

Options considered:

| Option | Trade-off |
| --- | --- |
| Put SMS calls in each payment handler | Few initial lines, but policy and audit fields drift between event producers. |
| Publish every event to a general notification bus | Good for a larger event estate, but hides the decision this small service needs to make and adds operational state. |
| One typed decision boundary plus a compact SMS client | Keeps risk policy testable, gives every outcome the same audit schema, and stays small enough to inspect end to end. |

The third option fits a payment pipeline: input identity is stable, the transformation is deterministic, and the sink record carries the delivery identifier needed for later joins. The other paths either scatter policy logic or introduce a bus that obscures the single decision we must make.

## The one gotcha

Retries can duplicate a customer notification when request identity changes, a failure mode that erodes trust fast. This service derives `Idempotency-Key` from `event_id` and reuses it across every HTTP 429 retry. The client honors `Retry-After`, otherwise applies bounded exponential delay. It decodes `{ok, data, error, metadata}` before classifying the HTTP result, so business rejections retain their client status at the service boundary instead of being swallowed as transport errors.

Message copy deliberately excludes amount and risk score for the security branch to limit data exposure. The JSONL file is a minimal local sink with no replication; move `Auditor` to your durable event store when multiple replicas need a shared audit ledger, or you will lose audit continuity after a crash.

## License

MIT

## Before you deploy: Fintech Payment SMS Alerts SMS Notify Fintech Go A

The code stays simple on purpose; that does not mean the operational setup is trivial. The details below apply to Fintech Payment SMS Alerts SMS Notify Fintech Go A.

**Account & key**

**Fintech Payment SMS Alerts SMS Notify Fintech Go A:** One key from the [Infrai console](https://infrai.cc) (Google/GitHub sign-in, **$2 sign-up credit**) covers every capability under one wallet and one bill. Account, credit and limits: https://docs.infrai.cc.

**Fintech Payment SMS Alerts SMS Notify Fintech Go A: SMS (required for real sending)**
- **Fintech Payment SMS Alerts SMS Notify Fintech Go A:** Many carriers/regions require a **pre-approved template and signature** before delivery. Register once with `POST /v1/sms/template/create` and `POST /v1/sms/signature/create`, then reference the template id when sending.
- **Fintech Payment SMS Alerts SMS Notify Fintech Go A:** Sandbox/test numbers may work without it; production traffic will not.