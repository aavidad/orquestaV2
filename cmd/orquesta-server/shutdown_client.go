package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type serverShutdownClientResultV0 struct {
	Estado                  string `json:"estado"`
	Status                  string `json:"status"`
	ShutdownReady           bool   `json:"shutdown_ready"`
	RunsRequested           int    `json:"runs_requested"`
	RunsStopped             int    `json:"runs_stopped"`
	AgentsInFlight          int    `json:"agents_in_flight"`
	CheckpointsPending      int    `json:"checkpoints_pending"`
	CheckpointAgentsPending int    `json:"checkpoint_agents_pending"`
}

func requestServerShutdownV0(addr string) error {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return fmt.Errorf("server_addr_vacio")
	}
	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(map[string]any{
		"forced":          true,
		"requested_by":    "orquesta-director",
		"reason":          "apagado controlado solicitado por CLI al Director",
		"idempotency_key": "idem-orquesta-server-stop",
	}); err != nil {
		return fmt.Errorf("shutdown_request_encode")
	}
	client := http.Client{Timeout: 120 * time.Second}
	response, err := client.Post(
		"http://"+addr+"/api/v0/server/shutdown",
		"application/json",
		body,
	)
	if err != nil {
		return fmt.Errorf("shutdown_request_failed")
	}
	defer response.Body.Close()
	var result serverShutdownClientResultV0
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return fmt.Errorf("shutdown_response_invalid")
	}
	if response.StatusCode != http.StatusOK || result.Estado != "ok" {
		return fmt.Errorf("shutdown_http_%d_%s", response.StatusCode, result.Status)
	}
	if !result.ShutdownReady {
		return fmt.Errorf(
			"shutdown_not_ready status=%s runs=%d/%d agents_in_flight=%d checkpoints=%d checkpoint_agents=%d",
			result.Status,
			result.RunsStopped,
			result.RunsRequested,
			result.AgentsInFlight,
			result.CheckpointsPending,
			result.CheckpointAgentsPending,
		)
	}
	return nil
}
