package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"time"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func runMain(args []string, stdout io.Writer, stderr io.Writer) int {
	command := "run"
	if len(args) > 0 {
		command = args[0]
	}
	switch command {
	case "run":
		return runServerCommandV0(stdout, stderr)
	case "start":
		return startServerCommandV0(stdout, stderr)
	case "status":
		return statusServerCommandV0(stdout, stderr)
	case "stop":
		return stopServerCommandV0(stdout, stderr)
	case "opes-drain-once":
		return opesDrainOnceCommandV0(stdout, stderr)
	case "codex-launch-wave":
		return codexLaunchWaveCommandV0(args[1:], stdout, stderr)
	default:
		_, _ = fmt.Fprintf(stderr, "comando no soportado: %s\n", command)
		return 2
	}
}

func runServerCommandV0(_ io.Writer, stderr io.Writer) int {
	serverConfig, err := serverConfigFromEnvV0()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "orquesta-server: %v\n", err)
		return 1
	}
	runtime, err := buildRuntimeFromEnvV0()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "orquesta-server: %v\n", err)
		return 1
	}
	bridgeConfig, err := opesBridgeLoopConfigFromEnvV0("http://" + serverConfig.Addr)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "orquesta-server opes-bridge: %v\n", err)
		return 1
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	go runOPESBridgeLoopV0(ctx, bridgeConfig, stderr, runOPESDrainOnceV0)
	if err := runtime.RunV0(ctx); err != nil {
		_, _ = fmt.Fprintf(stderr, "orquesta-server: %v\n", err)
		return 1
	}
	return 0
}

func statusServerCommandV0(stdout io.Writer, stderr io.Writer) int {
	config, err := serverConfigFromEnvV0()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "orquesta-server status: %v\n", err)
		return 1
	}
	state, err := loadStateV0(config)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "orquesta-server status: %v\n", err)
		return 1
	}
	if body, err := getStatusBodyV0(state.Addr); err == nil {
		_, _ = stdout.Write(body)
		return 0
	}
	_ = json.NewEncoder(stdout).Encode(state)
	return 1
}

func stopServerCommandV0(stdout io.Writer, stderr io.Writer) int {
	config, err := serverConfigFromEnvV0()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "orquesta-server stop: %v\n", err)
		return 1
	}
	state, err := loadStateV0(config)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "orquesta-server stop: %v\n", err)
		return 1
	}
	if err := requestServerShutdownV0(state.Addr); err != nil {
		_, _ = fmt.Fprintf(stderr, "orquesta-server stop: %v\n", err)
		return 1
	}
	if err := signalProcessV0(state.PID); err != nil {
		_, _ = fmt.Fprintf(stderr, "orquesta-server stop: %v\n", err)
		return 1
	}
	waitUntilDownV0(state.Addr, 5*time.Second)
	_, _ = fmt.Fprintf(stdout, "stop solicitado pid=%d\n", state.PID)
	return 0
}

func loadStateV0(config orquestaserver.ConfigV0) (orquestaserver.StateV0, error) {
	store, err := orquestaserver.NewFileStateStoreV0(orquestaserver.StatePathV0(config))
	if err != nil {
		return orquestaserver.StateV0{}, err
	}
	return store.LoadServerStateV0(context.Background())
}

func getStatusBodyV0(addr string) ([]byte, error) {
	client := http.Client{Timeout: 2 * time.Second}
	response, err := client.Get("http://" + addr + "/api/status")
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status_http_%d", response.StatusCode)
	}
	return io.ReadAll(response.Body)
}

func waitUntilDownV0(addr string, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := getStatusBodyV0(addr); err != nil {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
}
