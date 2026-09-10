package paymentalerts

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestSMSClientRetriesRateResponseWithSameRequestIdentity(t *testing.T) {
	var calls int
	client := NewSMSClient("test-key")
	client.HTTP = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		calls++
		if request.Method != http.MethodPost || request.URL.Path != "/v1/sms/send" {
			t.Fatalf("request = %s %s", request.Method, request.URL.Path)
		}
		if request.Header.Get("Authorization") != "Bearer "+client.APIKey || request.Header.Get("Idempotency-Key") != "payment-event:evt-9" {
			t.Fatalf("request headers = %#v", request.Header)
		}
		if calls == 1 {
			return response(http.StatusTooManyRequests, `{"ok":false,"data":null,"error":{"message":"request rate reached"},"metadata":{}}`, "1"), nil
		}
		return response(http.StatusOK, `{"ok":true,"data":{"message_id":"msg_9"},"error":null,"metadata":{"vendor":"selected"}}`, ""), nil
	})}
	var delays []time.Duration
	client.Sleep = func(_ context.Context, delay time.Duration) error {
		delays = append(delays, delay)
		return nil
	}

	result, err := client.Send(context.Background(), "+15550109", "Review account activity.", "payment-event:evt-9")
	if err != nil {
		t.Fatal(err)
	}
	if result.MessageID != "msg_9" || calls != 2 {
		t.Fatalf("result = %#v, calls = %d", result, calls)
	}
	if len(delays) != 1 || delays[0] != time.Second {
		t.Fatalf("retry delays = %v", delays)
	}
}

func response(status int, body, retryAfter string) *http.Response {
	header := make(http.Header)
	if retryAfter != "" {
		header.Set("Retry-After", retryAfter)
	}
	return &http.Response{StatusCode: status, Header: header, Body: io.NopCloser(strings.NewReader(body))}
}
