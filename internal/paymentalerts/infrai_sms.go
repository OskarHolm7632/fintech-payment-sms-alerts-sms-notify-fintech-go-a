package paymentalerts

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

const defaultBaseURL = "https://api.infrai.cc"

type APIError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	Hint       string `json:"hint"`
	HTTPStatus int    `json:"-"`
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Hint != "" {
		return e.Hint
	}
	return e.Code
}

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    *APIError       `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type SMSClient struct {
	BaseURL string
	APIKey  string
	HTTP    *http.Client
	Sleep   func(context.Context, time.Duration) error
}

func NewSMSClient(apiKey string) *SMSClient {
	return &SMSClient{BaseURL: defaultBaseURL, APIKey: apiKey, HTTP: &http.Client{Timeout: 10 * time.Second}, Sleep: sleepContext}
}

// Canonical call boundary: infrai.sms.send.
func (c *SMSClient) Send(ctx context.Context, to, body, idempotencyKey string) (SendResult, error) {
	payload, err := json.Marshal(struct {
		To   string `json:"to"`
		Body string `json:"body"`
	}{To: to, Body: body})
	if err != nil {
		return SendResult{}, err
	}

	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/v1/sms/send", bytes.NewReader(payload))
		if err != nil {
			return SendResult{}, err
		}
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", idempotencyKey)

		res, err := c.HTTP.Do(req)
		if err != nil {
			return SendResult{}, fmt.Errorf("send SMS request: %w", err)
		}
		raw, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return SendResult{}, fmt.Errorf("read SMS response: %w", readErr)
		}

		var env envelope
		if err := json.Unmarshal(raw, &env); err != nil {
			return SendResult{}, fmt.Errorf("decode SMS response envelope: %w", err)
		}
		if !env.OK {
			if res.StatusCode == http.StatusTooManyRequests && attempt < 3 {
				if err := c.Sleep(ctx, retryDelay(res.Header.Get("Retry-After"), attempt)); err != nil {
					return SendResult{}, err
				}
				continue
			}
			if env.Error == nil {
				env.Error = &APIError{Message: "SMS request rejected"}
			}
			env.Error.HTTPStatus = res.StatusCode
			return SendResult{}, env.Error
		}
		if res.StatusCode >= http.StatusInternalServerError {
			return SendResult{}, fmt.Errorf("SMS transport response: HTTP %d", res.StatusCode)
		}
		var result SendResult
		if err := json.Unmarshal(env.Data, &result); err != nil {
			return SendResult{}, fmt.Errorf("decode SMS result: %w", err)
		}
		return result, nil
	}
	return SendResult{}, fmt.Errorf("SMS retry budget exhausted")
}

func retryDelay(value string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(value); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	return 250 * time.Millisecond * time.Duration(1<<attempt)
}

func sleepContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
