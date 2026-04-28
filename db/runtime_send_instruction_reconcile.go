package db

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

func reconciliarRuntimeOrderSendInstructionConMailboxActual(order *RuntimeOrder, payload map[string]any, now time.Time) (bool, error) {
	if order == nil || !runtimeOrderSendInstructionProvieneMailbox(payload) {
		return false, nil
	}
	if reason, err := runtimeOrderSendInstructionObsoletaPorTarea(order, payload); err != nil {
		return false, err
	} else if reason != "" {
		if err := completarRuntimeOrderSendInstructionSupersedida(order, payload, reason); err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return false, err
			}
		}
		return true, nil
	}
	if newer, err := runtimeOrderSendInstructionDuplicadaMasReciente(order, payload); err != nil {
		return false, err
	} else if newer != nil {
		reason := fmt.Sprintf("covered_by_newer_mailbox_order:%d", newer.ID)
		if err := completarRuntimeOrderSendInstructionSupersedida(order, payload, reason); err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return false, err
			}
		}
		return true, nil
	}
	mailboxID := runtimeOrderSendInstructionMailboxID(payload)
	if mailboxID <= 0 {
		return false, nil
	}
	msg, err := GetRuntimeMailbox(mailboxID)
	if err != nil {
		return false, err
	}
	if msg == nil {
		if runtimeOrderSendInstructionDebeCerrarMailboxFaltante(order, payload) {
			if confirmed, receiptSource, receiptAt, err := runtimeOrderSendInstructionReceiptEvidence(order, payload, nil); err != nil {
				if !errors.Is(err, sql.ErrNoRows) {
					return false, err
				}
			} else if confirmed {
				if err := completarRuntimeOrderSendInstructionEntregadaPorReceipt(order, payload, receiptSource, receiptAt); err != nil {
					return false, err
				}
				return true, nil
			}
			if err := completarRuntimeOrderSendInstructionSupersedida(order, payload, "mailbox_missing_after_notify"); err != nil {
				if !errors.Is(err, sql.ErrNoRows) {
					return false, err
				}
			}
			return true, nil
		}
		if err := retenerRuntimeOrderSendInstructionDiferidaAMailbox(order, payload, "mailbox_missing_rearmed"); err != nil {
			return false, err
		}
		return true, nil
	}
	switch strings.ToLower(strings.TrimSpace(msg.Estado)) {
	case "pendiente":
		return false, nil
	case "entregado":
		if handled, err := reconciliarRuntimeOrderSendInstructionMailboxEntregado(order, payload, msg, now); err != nil {
			return false, err
		} else if handled {
			return true, nil
		}
		return false, nil
	default:
		if err := completarRuntimeOrderSendInstructionDiferidaAMailbox(order, payload, "mailbox ya "+strings.TrimSpace(msg.Estado)); err != nil {
			return false, err
		}
		return true, nil
	}
}

func completarRuntimeOrderSendInstructionSesionObsoleta(order *RuntimeOrder, payload map[string]any, handle *RuntimeHandle, externalSessionID string) (bool, error) {
	if order == nil || !runtimeOrderSendInstructionProvieneMailbox(payload) {
		return false, nil
	}
	payloadExternal := runtimeOrderSendInstructionExternalSessionID(payload)
	actualExternal := strings.TrimSpace(externalSessionID)
	if payloadExternal == "" || actualExternal == "" || payloadExternal == actualExternal {
		return false, nil
	}
	resultado := mergeRuntimeOrderResultJSON(order.ResultadoJSON, map[string]any{
		"ok":                  true,
		"mailbox_id":          runtimeOrderSendInstructionMailboxID(payload),
		"superseded":          true,
		"superseded_reason":   "external_session_id_changed",
		"external_session_id": actualExternal,
		"handle_id":           runtimeHandleID(handle),
	})
	return true, MarcarRuntimeOrderEstado(order.ID, "completada", resultado, "")
}

func completarRuntimeOrderSendInstructionCubiertaPorBootstrap(order *RuntimeOrder, payload map[string]any, handle *RuntimeHandle, runtime *RuntimeInstance) (bool, error) {
	if order == nil || !runtimeOrderSendInstructionProvieneMailbox(payload) {
		return false, nil
	}
	mailboxID := runtimeOrderSendInstructionMailboxID(payload)
	if mailboxID <= 0 {
		return false, nil
	}
	covered, bootstrapOrderID, startOrderID, err := RuntimeMailboxCubiertoPorBootstrapPendiente(mailboxID, handle, runtime)
	if err != nil || !covered {
		return false, err
	}
	resultado := mergeRuntimeOrderResultJSON(order.ResultadoJSON, map[string]any{
		"ok":                 true,
		"mailbox_id":         mailboxID,
		"superseded":         true,
		"superseded_reason":  "covered_by_bootstrap_lease",
		"bootstrap_order_id": bootstrapOrderID,
		"start_order_id":     startOrderID,
		"handle_id":          runtimeHandleID(handle),
		"runtime_id":         runtimeInstanceID(runtime),
	})
	return true, MarcarRuntimeOrderEstado(order.ID, "completada", resultado, "")
}
