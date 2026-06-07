package orquestaopesconnector

import (
	"bytes"
	"context"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	OutboundRetryReasonRateLimitedV0          = "rate_limited"
	OutboundRetryReasonRetryScheduledV0       = "retry_scheduled"
	OutboundRetryReasonRetryBudgetExhaustedV0 = "retry_budget_exhausted"
)

type RetryPolicyV0 struct {
	MaxAttempts       int
	BaseDelay         time.Duration
	MaxDelay          time.Duration
	RespectRetryAfter bool
	Jitter            func(int, time.Duration) time.Duration
	Sleep             func(context.Context, time.Duration) error
}

func normalizeRetryPolicyV0(policy RetryPolicyV0) RetryPolicyV0 {
	if policy.MaxAttempts < 1 {
		policy.MaxAttempts = 1
	}
	if policy.BaseDelay < 0 {
		policy.BaseDelay = 0
	}
	if policy.MaxDelay < 0 {
		policy.MaxDelay = 0
	}
	if policy.MaxDelay > 0 && policy.BaseDelay > policy.MaxDelay {
		policy.BaseDelay = policy.MaxDelay
	}
	return policy
}

func (client RESTClientV0) doJSONWithRetryV0(
	ctx context.Context,
	method string,
	requestURL string,
	body []byte,
	retryKey string,
	target any,
) error {
	for attempt := 1; ; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, requestURL, bytes.NewReader(body))
		if err != nil {
			return err
		}
		if method == http.MethodGet {
			req.Header.Set("Accept", "application/json")
		} else {
			req.Header.Set("Content-Type", "application/json")
		}
		res, err := client.httpClient.Do(req)
		if err != nil {
			code := opesHTTPErrorCodeV0(ctx, err)
			if decision, ok := opesRetryDecisionForTransportErrorV0(client.retryPolicy, method, attempt, retryKey, code, ctx); ok {
				if decision.Code != "" {
					return connectorErrorV0{code: decision.Code}
				}
				if err := sleepOPESRetryDecisionV0(ctx, client.retryPolicy, decision.Delay); err != nil {
					return connectorErrorV0{code: opesHTTPErrorCodeV0(ctx, err)}
				}
				continue
			}
			return connectorErrorV0{code: code}
		}
		if decision, ok := opesRetryDecisionForResponseV0(client.retryPolicy, method, attempt, retryKey, res, ctx); ok {
			discardOPESRetryResponseV0(res)
			if decision.Code != "" {
				return connectorErrorV0{code: decision.Code}
			}
			if err := sleepOPESRetryDecisionV0(ctx, client.retryPolicy, decision.Delay); err != nil {
				return connectorErrorV0{code: opesHTTPErrorCodeV0(ctx, err)}
			}
			continue
		}
		if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
			discardOPESResponseBodyV0(res)
			return connectorErrorV0{code: opesHTTPStatusErrorCodeV0(res.StatusCode)}
		}
		err = decodeOPESJSONResponseV0(res, target)
		res.Body.Close()
		return err
	}
}

type opesRetryDecisionV0 struct {
	Delay time.Duration
	Code  string
}

func opesRetryDecisionForTransportErrorV0(
	policy RetryPolicyV0,
	method string,
	attempt int,
	retryKey string,
	code string,
	ctx context.Context,
) (opesRetryDecisionV0, bool) {
	if code != ErrOPESHTTPRequestFailedV0 || policy.MaxAttempts <= 1 {
		return opesRetryDecisionV0{}, false
	}
	return opesRetryDecisionV0ForRetryableV0(policy, method, attempt, retryKey, "", ctx)
}

func opesRetryDecisionForResponseV0(
	policy RetryPolicyV0,
	method string,
	attempt int,
	retryKey string,
	response *http.Response,
	ctx context.Context,
) (opesRetryDecisionV0, bool) {
	if response == nil || !opesRetryableHTTPStatusV0(response.StatusCode) || policy.MaxAttempts <= 1 {
		return opesRetryDecisionV0{}, false
	}
	return opesRetryDecisionV0ForRetryableV0(policy, method, attempt, retryKey, response.Header.Get("Retry-After"), ctx)
}

func opesRetryDecisionV0ForRetryableV0(
	policy RetryPolicyV0,
	method string,
	attempt int,
	retryKey string,
	retryAfter string,
	ctx context.Context,
) (opesRetryDecisionV0, bool) {
	if method != http.MethodGet && strings.TrimSpace(retryKey) == "" {
		return opesRetryDecisionV0{Code: ErrOPESRetryBlockedV0}, true
	}
	if attempt >= policy.MaxAttempts {
		return opesRetryDecisionV0{Code: ErrOPESRetryBudgetExhaustedV0}, true
	}
	decision, ok := opesRetryDelayDecisionV0(policy, attempt, retryAfter, ctx)
	if !ok {
		return opesRetryDecisionV0{Code: ErrOPESRetryBudgetExhaustedV0}, true
	}
	return decision, true
}

func opesRetryDelayDecisionV0(
	policy RetryPolicyV0,
	attempt int,
	retryAfter string,
	ctx context.Context,
) (opesRetryDecisionV0, bool) {
	delay := opesRetryBackoffDelayV0(policy, attempt)
	if policy.RespectRetryAfter {
		if parsed, ok := parseOPESRetryAfterDelayV0(retryAfter); ok {
			delay = parsed
		}
	}
	if policy.MaxDelay > 0 && delay > policy.MaxDelay {
		return opesRetryDecisionV0{}, false
	}
	if deadline, ok := ctx.Deadline(); ok && time.Now().Add(delay).After(deadline) {
		return opesRetryDecisionV0{}, false
	}
	if policy.Jitter != nil {
		delay = policy.Jitter(attempt, delay)
		if delay < 0 {
			delay = 0
		}
	}
	return opesRetryDecisionV0{Delay: delay}, true
}

func opesRetryBackoffDelayV0(policy RetryPolicyV0, attempt int) time.Duration {
	delay := policy.BaseDelay
	if delay <= 0 {
		delay = 100 * time.Millisecond
	}
	delay = time.Duration(float64(delay) * math.Pow(2, float64(attempt-1)))
	if policy.MaxDelay > 0 && delay > policy.MaxDelay {
		return policy.MaxDelay
	}
	return delay
}

func parseOPESRetryAfterDelayV0(value string) (time.Duration, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	if seconds, err := strconv.Atoi(value); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second, true
	}
	if when, err := http.ParseTime(value); err == nil {
		delay := time.Until(when)
		if delay < 0 {
			return 0, true
		}
		return delay, true
	}
	return 0, false
}

func sleepOPESRetryDecisionV0(ctx context.Context, policy RetryPolicyV0, delay time.Duration) error {
	if policy.Sleep != nil {
		return policy.Sleep(ctx, delay)
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func opesRetryableHTTPStatusV0(status int) bool {
	return status == http.StatusTooManyRequests || status == http.StatusServiceUnavailable || status == http.StatusBadGateway || status == http.StatusGatewayTimeout
}

func opesHTTPStatusErrorCodeV0(status int) string {
	if status <= 0 {
		return ErrOPESHTTPStatusV0
	}
	return ErrOPESHTTPStatusV0 + "_" + strconv.Itoa(status)
}

func opesHTTPStatusErrorCodeIs4xxV0(code string) bool {
	if !strings.HasPrefix(code, ErrOPESHTTPStatusV0+"_") {
		return false
	}
	status, err := strconv.Atoi(strings.TrimPrefix(code, ErrOPESHTTPStatusV0+"_"))
	return err == nil && status >= http.StatusBadRequest && status < http.StatusInternalServerError
}

func opesRetryKeyFromPayloadV0(payload any) string {
	switch value := payload.(type) {
	case opesCreateJobRequestV0:
		return strings.TrimSpace(value.IdempotencyKey)
	case opesArtifactRequestV0:
		return strings.TrimSpace(value.IdempotencyKey)
	default:
		return ""
	}
}

func discardOPESRetryResponseV0(response *http.Response) {
	if response == nil || response.Body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, defaultOPESResponseMaxBytesV0))
	response.Body.Close()
}
