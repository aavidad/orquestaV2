package orquestaopesconnector

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

const defaultOPESHTTPTimeoutV0 = 30 * time.Second

func (client RESTClientV0) getJSONV0(
	ctx context.Context,
	path string,
	target any,
) error {
	ctx, cancel := client.effectContextV0(ctx)
	defer cancel()
	requestURL, err := client.controlledURLV0(path)
	if err != nil {
		return err
	}
	return client.doJSONWithRetryV0(ctx, "GET", requestURL, nil, "", target)
}

func (client RESTClientV0) postJSONV0(
	ctx context.Context,
	path string,
	payload any,
	target any,
) error {
	ctx, cancel := client.effectContextV0(ctx)
	defer cancel()
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	requestURL, err := client.controlledURLV0(path)
	if err != nil {
		return err
	}
	return client.doJSONWithRetryV0(ctx, "POST", requestURL, body, opesRetryKeyFromPayloadV0(payload), target)
}

func trimTrailingSlashV0(value string) string {
	return strings.TrimRight(strings.TrimSpace(value), "/")
}

func (client RESTClientV0) effectContextV0(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, client.timeout)
}

func opesRESTTimeoutV0(timeout time.Duration) time.Duration {
	if timeout <= 0 {
		return defaultOPESHTTPTimeoutV0
	}
	return timeout
}

func opesHTTPErrorCodeV0(ctx context.Context, err error) string {
	if code, ok := opesHTTPRedirectErrorCodeV0(err); ok {
		return code
	}
	switch {
	case errors.Is(err, context.DeadlineExceeded), errors.Is(ctx.Err(), context.DeadlineExceeded):
		return ErrOPESHTTPTimeoutV0
	case errors.Is(err, context.Canceled), errors.Is(ctx.Err(), context.Canceled):
		return ErrOPESHTTPCancelledV0
	default:
		return ErrOPESHTTPRequestFailedV0
	}
}
