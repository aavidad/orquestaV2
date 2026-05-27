package orquestadomainworkhttp

import (
	"bytes"
	"context"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
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

func (client ClientV0) postJSONEncodedV0(
	ctx context.Context,
	path string,
	data []byte,
	retryKey string,
	target any,
) error {
	ctx, cancel := client.effectContextV0(ctx)
	defer cancel()
	for attempt := 1; ; attempt++ {
		request, err := http.NewRequestWithContext(ctx, http.MethodPost, client.baseURL+path, bytes.NewReader(data))
		if err != nil {
			return ErrorV0{Code: ErrDomainWorkHTTPRequestBuildFailedV0}
		}
		request.Header.Set("Content-Type", "application/json")
		response, err := client.httpClient.Do(request)
		if err != nil {
			code := domainWorkHTTPRequestErrorCodeV0(ctx, err)
			if decision, ok := retryDecisionForTransportErrorV0(client.retryPolicy, attempt, retryKey, code, ctx); ok {
				if decision.Code != "" {
					return ErrorV0{Code: decision.Code}
				}
				if err := sleepRetryDecisionV0(ctx, client.retryPolicy, decision.Delay); err != nil {
					return ErrorV0{Code: domainWorkHTTPRequestErrorCodeV0(ctx, err)}
				}
				continue
			}
			return ErrorV0{Code: code}
		}
		if decision, ok := retryDecisionForResponseV0(client.retryPolicy, attempt, retryKey, response, ctx); ok {
			discardAndCloseResponseV0(response)
			if decision.Code != "" {
				return ErrorV0{Code: decision.Code}
			}
			if err := sleepRetryDecisionV0(ctx, client.retryPolicy, decision.Delay); err != nil {
				return ErrorV0{Code: domainWorkHTTPRequestErrorCodeV0(ctx, err)}
			}
			continue
		}
		err = decodeDomainWorkHTTPJSONResponseV0(response, target)
		response.Body.Close()
		return err
	}
}

type retryDecisionV0 struct {
	Delay time.Duration
	Code  string
}

func retryDecisionForTransportErrorV0(
	policy RetryPolicyV0,
	attempt int,
	retryKey string,
	code string,
	ctx context.Context,
) (retryDecisionV0, bool) {
	if code != ErrDomainWorkHTTPRequestFailedV0 || policy.MaxAttempts <= 1 {
		return retryDecisionV0{}, false
	}
	if strings.TrimSpace(retryKey) == "" {
		return retryDecisionV0{Code: ErrDomainWorkHTTPRetryBlockedV0}, true
	}
	if attempt >= policy.MaxAttempts {
		return retryDecisionV0{Code: ErrDomainWorkHTTPRetryBudgetExhaustedV0}, true
	}
	decision, ok := retryDelayDecisionV0(policy, attempt, "", ctx)
	if !ok {
		return retryDecisionV0{Code: ErrDomainWorkHTTPRetryBudgetExhaustedV0}, true
	}
	return decision, true
}

func retryDecisionForResponseV0(
	policy RetryPolicyV0,
	attempt int,
	retryKey string,
	response *http.Response,
	ctx context.Context,
) (retryDecisionV0, bool) {
	if response == nil || !retryableHTTPStatusV0(response.StatusCode) {
		return retryDecisionV0{}, false
	}
	if policy.MaxAttempts <= 1 {
		return retryDecisionV0{}, false
	}
	if strings.TrimSpace(retryKey) == "" {
		return retryDecisionV0{Code: ErrDomainWorkHTTPRetryBlockedV0}, true
	}
	if attempt >= policy.MaxAttempts {
		return retryDecisionV0{Code: ErrDomainWorkHTTPRetryBudgetExhaustedV0}, true
	}
	decision, ok := retryDelayDecisionV0(policy, attempt, response.Header.Get("Retry-After"), ctx)
	if !ok {
		return retryDecisionV0{Code: ErrDomainWorkHTTPRetryBudgetExhaustedV0}, true
	}
	return decision, true
}

func retryDelayDecisionV0(
	policy RetryPolicyV0,
	attempt int,
	retryAfter string,
	ctx context.Context,
) (retryDecisionV0, bool) {
	delay := retryBackoffDelayV0(policy, attempt)
	if policy.RespectRetryAfter {
		if parsed, ok := parseRetryAfterDelayV0(retryAfter); ok {
			delay = parsed
		}
	}
	if policy.MaxDelay > 0 && delay > policy.MaxDelay {
		return retryDecisionV0{}, false
	}
	if deadline, ok := ctx.Deadline(); ok && time.Now().Add(delay).After(deadline) {
		return retryDecisionV0{}, false
	}
	if policy.Jitter != nil {
		delay = policy.Jitter(attempt, delay)
		if delay < 0 {
			delay = 0
		}
	}
	return retryDecisionV0{Delay: delay}, true
}

func retryBackoffDelayV0(policy RetryPolicyV0, attempt int) time.Duration {
	delay := policy.BaseDelay
	if delay <= 0 {
		delay = 100 * time.Millisecond
	}
	factor := math.Pow(2, float64(attempt-1))
	delay = time.Duration(float64(delay) * factor)
	if policy.MaxDelay > 0 && delay > policy.MaxDelay {
		return policy.MaxDelay
	}
	return delay
}

func parseRetryAfterDelayV0(value string) (time.Duration, bool) {
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

func sleepRetryDecisionV0(ctx context.Context, policy RetryPolicyV0, delay time.Duration) error {
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

func retryableHTTPStatusV0(status int) bool {
	return status == http.StatusTooManyRequests || status == http.StatusServiceUnavailable || status == http.StatusBadGateway || status == http.StatusGatewayTimeout
}

func retryKeyFromPayloadV0(payload any) string {
	switch value := payload.(type) {
	case orquestadomainwork.DomainWorkJobRequestV0:
		return strings.TrimSpace(value.IdempotencyKey)
	case orquestadomainwork.DomainWorkArtifactSubmissionV0:
		return strings.TrimSpace(value.IdempotencyKey)
	default:
		return ""
	}
}

func discardAndCloseResponseV0(response *http.Response) {
	if response == nil || response.Body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, defaultDomainWorkHTTPResponseMaxBytesV0))
	response.Body.Close()
}
