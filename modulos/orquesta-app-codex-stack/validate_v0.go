package orquestaappcodexstack

import (
	"fmt"
	"strings"

	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

func validateConfigV0(config ConfigV0) error {
	switch {
	case !config.Enabled:
		return fmt.Errorf("orquesta_app_codex_stack: opt_in_requerido")
	case config.Stores.RunStore == nil:
		return fmt.Errorf("orquesta_app_codex_stack: run_store requerido")
	case config.Stores.EventSink == nil:
		return fmt.Errorf("orquesta_app_codex_stack: event_sink requerido")
	case config.Stores.OutboxLedger == nil:
		return fmt.Errorf("orquesta_app_codex_stack: outbox_ledger requerido")
	case config.Stores.TaskStore == nil:
		return fmt.Errorf("orquesta_app_codex_stack: task_store requerido")
	case config.Stores.AppChangeStore == nil:
		return fmt.Errorf("orquesta_app_codex_stack: app_change_store requerido")
	case config.Stores.ReceiptStore == nil:
		return fmt.Errorf("orquesta_app_codex_stack: receipt_store requerido")
	case config.Stores.ProgressState == nil:
		return fmt.Errorf("orquesta_app_codex_stack: progress_state requerido")
	case config.Stores.ProcessRegistry == nil:
		return fmt.Errorf("orquesta_app_codex_stack: process_registry requerido")
	case config.Stores.RunControl == nil:
		return fmt.Errorf("orquesta_app_codex_stack: run_control requerido")
	case config.Stores.RunQueue == nil:
		return fmt.Errorf("orquesta_app_codex_stack: run_queue requerido")
	case config.ReviewGate.FileEvidence == nil:
		return fmt.Errorf("orquesta_app_codex_stack: review_gate.file_evidence requerido")
	case config.Codex.Runtime == nil:
		return fmt.Errorf("orquesta_app_codex_stack: runtime requerido")
	case config.Codex.ProcessStopper == nil:
		return fmt.Errorf("orquesta_app_codex_stack: process_stopper requerido")
	case config.Codex.MaxBatchReady <= 0:
		return fmt.Errorf("orquesta_app_codex_stack: max_batch_ready requerido")
	case config.Codex.MaxConcurrency <= 0:
		return fmt.Errorf("orquesta_app_codex_stack: max_concurrency requerido")
	case config.Codex.WaitInterval <= 0:
		return fmt.Errorf("orquesta_app_codex_stack: wait_interval requerido")
	case strings.TrimSpace(config.Capacity.OccurredAt) == "":
		return fmt.Errorf("orquesta_app_codex_stack: capacity.occurred_at requerido")
	case strings.TrimSpace(config.Capacity.RequestedBy) == "":
		return fmt.Errorf("orquesta_app_codex_stack: capacity.requested_by requerido")
	case config.AutoprogrammingPromotion.Enabled && config.AutoprogrammingPromotion.Port == nil:
		return fmt.Errorf("orquesta_app_codex_stack: autoprogramming_promotion.port requerido")
	}
	if issues := orquestaruntimecodex.ValidateCodexConnectorProfileV0(
		codexProfileV0(config.Codex, config.Codex.RuntimeWorkDir),
	); len(issues) > 0 {
		return fmt.Errorf("orquesta_app_codex_stack: codex_profile.%s", issues[0].Field)
	}
	if issues := orquestaruntimecodex.ValidateCodexConnectorProfileV0(
		codexProfileForAreaV0(config.Codex, config.Codex.RuntimeWorkDir, "director"),
	); len(issues) > 0 {
		return fmt.Errorf("orquesta_app_codex_stack: codex_director_profile.%s", issues[0].Field)
	}
	return nil
}
