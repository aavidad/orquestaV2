package main

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestExternalBridgeLoopAplicaDeadlinePorTickV0(t *testing.T) {
	var sawDeadline bool
	runExternalBridgeLoopV0(
		context.Background(),
		externalBridgeLoopConfigV0{
			Enabled:       true,
			EffectTimeout: time.Second,
			MaxTicks:      1,
		},
		nil,
		func(ctx context.Context) (any, error) {
			_, sawDeadline = ctx.Deadline()
			return nil, nil
		},
	)
	if !sawDeadline {
		t.Fatalf("tick sin deadline")
	}
}

func TestSubmitOPESExternalWorkRunV0MapeaDeadlinePublico(t *testing.T) {
	client := &http.Client{Transport: roundTripFuncV0(func(r *http.Request) (*http.Response, error) {
		return nil, context.DeadlineExceeded
	})}

	_, err := submitOPESExternalWorkRunV0(
		context.Background(),
		client,
		"http://orquesta-test.invalid",
		map[string]string{"request_ref": "request-ref-001"},
	)
	if err == nil || err.Error() != "orquesta_unreachable_timeout" {
		t.Fatalf("err=%v", err)
	}
}

type roundTripFuncV0 func(*http.Request) (*http.Response, error)

func (fn roundTripFuncV0) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}
