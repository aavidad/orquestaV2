package main

import (
	"context"
	"io"
)

type opesRegistryFinalPkgProducerV0 func(
	context.Context,
	opesRegistryFinalPkgConfigV0,
	opesRegistryFinalPkgSubmitterV0,
) (opesRegistryFinalPkgSummaryV0, error)

func runOPESRegistryFinalPkgLoopV0(
	ctx context.Context,
	config opesRegistryFinalPkgLoopConfigV0,
	stderr io.Writer,
	producer opesRegistryFinalPkgProducerV0,
) {
	if producer == nil {
		return
	}
	runExternalBridgeLoopV0(ctx, config.Loop, stderr, func(ctx context.Context) (any, error) {
		return producer(ctx, config.Producer, submitOPESRegistryFinalPkgRunV0)
	})
}

func runOPESRegistryFinalPkgLoopAsyncV0(
	ctx context.Context,
	config opesRegistryFinalPkgLoopConfigV0,
	stderr io.Writer,
	producer opesRegistryFinalPkgProducerV0,
) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		runOPESRegistryFinalPkgLoopV0(ctx, config, stderr, producer)
	}()
	return done
}

func waitOPESRegistryFinalPkgLoopDoneV0(
	_ context.Context,
	config opesRegistryFinalPkgLoopConfigV0,
	done <-chan struct{},
) bool {
	if done == nil {
		return true
	}
	timeout := externalBridgeLoopShutdownTimeoutV0(config.Loop)
	waitCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	select {
	case <-done:
		return true
	case <-waitCtx.Done():
		if config.Loop.Observer != nil {
			config.Loop.Observer(context.Background(), externalBridgeLoopEventV0{
				Component:  config.Loop.Component,
				Status:     "timeout",
				ErrorCode:  "external_bridge_shutdown_timeout",
				StopReason: "shutdown_timeout",
			})
		}
		return false
	}
}
