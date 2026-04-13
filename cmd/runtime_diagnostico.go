/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"fmt"
	"net/url"
	"os/exec"
	"slices"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"orquesta/db"
	"orquesta/internal/controlruntime"
	"orquesta/runtimeagente"
)

type runtimeDiagnosticoData struct {
	Agente           string
	Proyecto         string
	Fuente           string
	Runtimes         []runtimeRow
	Handles          []*db.RuntimeHandle
	Workers          []runtimeDiagnosticoWorkerRow
	Issues           []runtimeDiagnosticoIssue
	Orders           []*db.RuntimeOrder
	MailboxPendiente []*db.RuntimeMailboxMessage
	Checkpoints      []*db.RuntimeCheckpoint
}

type runtimeDiagnosticoIssue struct {
	Severity string
	Code     string
	Message  string
}

type runtimeDiagnosticoWorkerRow struct {
	HandleID            int64
	Agent               string
	Driver              string
	Transport           string
	RuntimeRef          string
	ChildPID            int
	State               string
	Alive               bool
	HeartbeatAt         *time.Time
	UpdatedAt           *time.Time
	LastOutputAt        *time.Time
	LastProgressAt      *time.Time
	SessionRef          string
	ExternalSessionID   string
	MailboxDeliveryMode string
	ExitError           string
	ManifestPath        string
	StatusPath          string
	HeartbeatPath       string
	TmuxSession         string
	TmuxPaneID          string
}

const runtimeDiagnosticoWorkerHeartbeatThreshold = time.Minute
const runtimeDiagnosticoWorkerProgressThreshold = 20 * time.Minute
const runtimeDiagnosticoResumePendingThreshold = 10 * time.Minute
const runtimeDiagnosticoRestartLoopWindow = 6 * time.Hour
const runtimeDiagnosticoRestartLoopCount = 2

var runtimeDiagnosticoNow = func() time.Time {
	return time.Now().UTC()
}

var runtimeDiagnosticoCmd = &cobra.Command{
	Use:   "diagnostico",
	Short: "Resume el estado operativo de un agente en el runtime",
	RunE: func(cmd *cobra.Command, args []string) error {
		agente, _ := cmd.Flags().GetString("agente")
		proyecto, _ := cmd.Flags().GetString("proyecto")
		limit, _ := cmd.Flags().GetInt("limit")
		data, err := cargarRuntimeDiagnostico(strings.TrimSpace(agente), strings.TrimSpace(proyecto), limit)
		if err != nil {
			return err
		}
		renderRuntimeDiagnostico(data, limit)
		return nil
	},
}

func cargarRuntimeDiagnostico(agente, proyecto string, limit int) (*runtimeDiagnosticoData, error) {
	if strings.TrimSpace(agente) == "" {
		return nil, fmt.Errorf("--agente es obligatorio")
	}
	if limit <= 0 {
		limit = 5
	}
	if data, ok, err := cargarRuntimeDiagnosticoDesdeAPI(agente, proyecto, limit); err != nil {
		return nil, err
	} else if ok {
		return data, nil
	} else if runtimeModoRecuperacionLocalExplicito() {
		return cargarRuntimeDiagnosticoRecuperacionLocal(agente, proyecto, limit)
	}
	return nil, serverFirstCommandError("runtime diagnostico")
}

func cargarRuntimeDiagnosticoDesdeAPI(agente, proyecto string, limit int) (*runtimeDiagnosticoData, bool, error) {
	query := url.Values{"agente": []string{agente}}
	if proyecto != "" {
		query.Set("proyecto", proyecto)
	}
	tree, ok, err := cargarArbolRuntimesDesdeAPI(query)
	if !ok || err != nil {
		return nil, ok, err
	}

	handles, ok, err := cargarRuntimeHandlesDesdeAPI(url.Values{"agente": []string{agente}})
	if !ok || err != nil {
		return nil, ok, err
	}
	orders, ok, err := cargarRuntimeOrdersDesdeAPI(query)
	if !ok || err != nil {
		return nil, ok, err
	}
	mailQuery := url.Values{
		"to_agente": []string{agente},
		"estado":    []string{"pendiente"},
	}
	if proyecto != "" {
		mailQuery.Set("proyecto", proyecto)
	}
	mailbox, ok, err := cargarRuntimeMailboxDesdeAPI(mailQuery)
	if !ok || err != nil {
		return nil, ok, err
	}
	checkpoints, ok, err := cargarRuntimeCheckpointsDesdeAPI(agente, proyecto, "", "", limit)
	if !ok || err != nil {
		return nil, ok, err
	}
	workers := runtimeWorkerRows(handles, limit)

	return &runtimeDiagnosticoData{
		Agente:           agente,
		Proyecto:         proyecto,
		Fuente:           "api",
		Runtimes:         aplanarArbolAPI(tree),
		Handles:          handles,
		Workers:          workers,
		Orders:           orders,
		MailboxPendiente: mailbox,
		Checkpoints:      checkpoints,
		Issues:           runtimeDiagnosticoIssues(workers, orders, mailbox, checkpoints),
	}, true, nil
}

func cargarRuntimeDiagnosticoRecuperacionLocal(agente, proyecto string, limit int) (*runtimeDiagnosticoData, error) {
	query := url.Values{"agente": []string{agente}}
	if proyecto != "" {
		query.Set("proyecto", proyecto)
	}
	tree, err := cargarArbolRuntimesRecuperacionLocal(query)
	if err != nil {
		return nil, err
	}
	handles, err := cargarRuntimeHandlesRecuperacionLocal(url.Values{"agente": []string{agente}})
	if err != nil {
		return nil, err
	}
	orders, err := cargarRuntimeOrdersRecuperacionLocal(query)
	if err != nil {
		return nil, err
	}
	mailQuery := url.Values{
		"to_agente": []string{agente},
		"estado":    []string{"pendiente"},
	}
	if proyecto != "" {
		mailQuery.Set("proyecto", proyecto)
	}
	mailbox, err := cargarRuntimeMailboxRecuperacionLocal(mailQuery)
	if err != nil {
		return nil, err
	}
	checkpoints, err := cargarRuntimeCheckpointsRecuperacionLocal(agente, proyecto, "", "", limit)
	if err != nil {
		return nil, err
	}

	data := &runtimeDiagnosticoData{
		Agente:           agente,
		Proyecto:         proyecto,
		Fuente:           "local",
		Runtimes:         aplanarArbolLocal(tree),
		Handles:          handles,
		Workers:          runtimeWorkerRows(handles, limit),
		Orders:           orders,
		MailboxPendiente: mailbox,
		Checkpoints:      checkpoints,
	}
	data.Issues = runtimeDiagnosticoIssues(data.Workers, data.Orders, data.MailboxPendiente, data.Checkpoints)
	return data, nil
}

func renderRuntimeDiagnostico(data *runtimeDiagnosticoData, limit int) {
	if data == nil {
		return
	}
	if limit <= 0 {
		limit = 5
	}
	fmt.Printf("╔═══════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║              ORQUESTA — DIAGNÓSTICO RUNTIME             ║\n")
	fmt.Printf("╚═══════════════════════════════════════════════════════════╝\n\n")
	fmt.Printf("Agente:     %s\n", data.Agente)
	fmt.Printf("Proyecto:   %s\n", valorVacio(data.Proyecto))
	fmt.Printf("Fuente:     %s\n", data.Fuente)
	fmt.Printf("Runtimes:   %d\n", len(data.Runtimes))
	fmt.Printf("Handles:    %d\n", len(data.Handles))
	fmt.Printf("Órdenes:    %d total (%s)\n", len(data.Orders), resumirEstadosRuntimeOrders(data.Orders))
	fmt.Printf("Mailbox:    %d pendiente(s)\n", len(data.MailboxPendiente))
	fmt.Printf("Checkpoints:%d recientes\n", len(data.Checkpoints))

	runtimes := runtimeRowsRelevantes(data.Runtimes, limit)
	if len(runtimes) > 0 {
		fmt.Println("\nRuntimes")
		_ = imprimirArbolRuntimes(runtimes, false)
	}
	handles := runtimeHandlesRelevantes(data.Handles, limit)
	if len(handles) > 0 {
		fmt.Println("\nHandles")
		_ = imprimirRuntimeHandles(handles)
	}
	if len(data.Workers) > 0 {
		fmt.Println("\nWorker estructurado")
		_ = imprimirRuntimeDiagnosticoWorkers(data.Workers)
		if len(data.Issues) > 0 {
			fmt.Println("\nIncidencias")
			_ = imprimirRuntimeDiagnosticoIssues(data.Issues)
		}
	}
	orders := runtimeOrdersRelevantes(data.Orders, limit)
	if len(orders) > 0 {
		fmt.Println("\nÓrdenes relevantes")
		_ = imprimirRuntimeOrders(orders)
	}
	if len(data.MailboxPendiente) > 0 {
		fmt.Println("\nMailbox pendiente")
		_ = imprimirRuntimeMailbox(data.MailboxPendiente)
	}
	if len(data.Checkpoints) > 0 {
		fmt.Println("\nCheckpoints recientes")
		_ = imprimirRuntimeDiagnosticoCheckpoints(data.Checkpoints)
	}
}

func runtimeRowsRelevantes(rows []runtimeRow, limit int) []runtimeRow {
	if limit <= 0 {
		limit = 5
	}
	if len(rows) <= limit {
		return rows
	}
	pick := make([]runtimeRow, 0, limit)
	seen := map[int64]struct{}{}
	appendIf := func(row runtimeRow) bool {
		if _, ok := seen[row.ID]; ok {
			return false
		}
		seen[row.ID] = struct{}{}
		pick = append(pick, row)
		return len(pick) >= limit
	}
	for i := len(rows) - 1; i >= 0; i-- {
		row := rows[i]
		if runtimeEstadoTerminal(strings.TrimSpace(row.Estado)) {
			continue
		}
		if appendIf(row) {
			break
		}
	}
	if len(pick) < limit {
		for i := len(rows) - 1; i >= 0; i-- {
			row := rows[i]
			if !runtimeEstadoTerminal(strings.TrimSpace(row.Estado)) {
				continue
			}
			if appendIf(row) {
				break
			}
		}
	}
	slices.Reverse(pick)
	return pick
}

func runtimeHandlesRelevantes(handles []*db.RuntimeHandle, limit int) []*db.RuntimeHandle {
	if limit <= 0 {
		limit = 5
	}
	if len(handles) <= limit {
		return handles
	}
	pick := make([]*db.RuntimeHandle, 0, limit)
	seen := map[int64]struct{}{}
	appendIf := func(handle *db.RuntimeHandle) bool {
		if handle == nil {
			return false
		}
		if _, ok := seen[handle.ID]; ok {
			return false
		}
		seen[handle.ID] = struct{}{}
		pick = append(pick, handle)
		return len(pick) >= limit
	}
	for _, handle := range handles {
		if handle == nil || runtimeEstadoTerminal(strings.TrimSpace(handle.Estado)) {
			continue
		}
		if appendIf(handle) {
			return pick
		}
	}
	for _, handle := range handles {
		if handle == nil || !runtimeEstadoTerminal(strings.TrimSpace(handle.Estado)) {
			continue
		}
		if appendIf(handle) {
			return pick
		}
	}
	return pick
}

func runtimeEstadoTerminal(estado string) bool {
	switch strings.TrimSpace(strings.ToLower(estado)) {
	case "cerrado", "fallido", "cancelada", "cancelado", "completada", "completado":
		return true
	default:
		return false
	}
}

func resumirEstadosRuntimeOrders(orders []*db.RuntimeOrder) string {
	if len(orders) == 0 {
		return "sin órdenes"
	}
	counts := map[string]int{}
	for _, order := range orders {
		if order == nil {
			continue
		}
		counts[strings.TrimSpace(order.Estado)]++
	}
	prioridad := []string{"pendiente", "tomada", "ejecutando", "fallida", "completada", "cancelada", "expirada"}
	partes := make([]string, 0, len(prioridad))
	for _, estado := range prioridad {
		if counts[estado] > 0 {
			partes = append(partes, fmt.Sprintf("%s=%d", estado, counts[estado]))
			delete(counts, estado)
		}
	}
	for estado, n := range counts {
		if strings.TrimSpace(estado) == "" {
			continue
		}
		partes = append(partes, fmt.Sprintf("%s=%d", estado, n))
	}
	if len(partes) == 0 {
		return "sin órdenes"
	}
	return strings.Join(partes, ", ")
}

func runtimeOrdersRelevantes(orders []*db.RuntimeOrder, limit int) []*db.RuntimeOrder {
	if limit <= 0 {
		limit = 5
	}
	relevantes := make([]*db.RuntimeOrder, 0, limit)
	for _, order := range orders {
		if order == nil {
			continue
		}
		switch strings.TrimSpace(order.Estado) {
		case "pendiente", "tomada", "ejecutando", "fallida":
			relevantes = append(relevantes, order)
		}
		if len(relevantes) >= limit {
			return relevantes
		}
	}
	if len(relevantes) > 0 {
		return relevantes
	}
	if len(orders) <= limit {
		return orders
	}
	return orders[:limit]
}

func imprimirRuntimeDiagnosticoCheckpoints(checkpoints []*db.RuntimeCheckpoint) error {
	if len(checkpoints) == 0 {
		fmt.Println("No hay checkpoints para ese filtro.")
		return nil
	}
	fmt.Printf("%-5s %-12s %-16s %-18s %-16s %-18s %s\n", "ID", "AGENTE", "KIND", "SOURCE", "BRANCH", "CREADO", "RESUMEN")
	for _, cp := range checkpoints {
		if cp == nil {
			continue
		}
		fmt.Printf("%-5d %-12s %-16s %-18s %-16s %-18s %s\n",
			cp.ID, truncar(cp.Agente, 12), truncar(cp.CheckpointKind, 16), truncar(cp.Source, 18),
			truncar(cp.Branch, 16), cp.CreatedAt.Format("2006-01-02 15:04:05"), truncar(cp.Resumen, 48))
	}
	return nil
}

func runtimeWorkerRows(handles []*db.RuntimeHandle, limit int) []runtimeDiagnosticoWorkerRow {
	if limit <= 0 {
		limit = 5
	}
	rows := make([]runtimeDiagnosticoWorkerRow, 0, limit)
	now := runtimeDiagnosticoNow()
	for _, handle := range runtimeHandlesRelevantes(handles, limit) {
		if handle == nil {
			continue
		}
		if repaired, restarted, err := controlruntime.EnsureTMUXMonitorFromMetadataJSON(handle.MetadataJSON); err == nil && strings.TrimSpace(repaired) != "" && strings.TrimSpace(repaired) != strings.TrimSpace(handle.MetadataJSON) {
			handle.MetadataJSON = repaired
			if restarted {
				_ = db.ActualizarMetadataRuntimeHandle(handle.ID, repaired)
			}
		}
		snap, err := runtimeagente.LoadWorkerSnapshotFromMetadataJSON(handle.MetadataJSON)
		if err != nil || snap == nil {
			continue
		}
		state := snap.EffectiveState()
		alive := snap.Alive()
		exitError := snap.ExitError()
		handleState := strings.TrimSpace(strings.ToLower(handle.Estado))
		if runtimeEstadoTerminal(handleState) {
			if alive {
				state = "stale"
				alive = false
			}
			if strings.TrimSpace(exitError) == "" {
				exitError = "handle=" + strings.TrimSpace(handle.Estado)
			}
		} else if handleState == "pausado" {
			state = "paused"
		} else if alive && snap.IsHeartbeatStale(now, runtimeDiagnosticoWorkerHeartbeatThreshold) {
			state = "stale"
			alive = false
			if strings.TrimSpace(exitError) == "" {
				exitError = "heartbeat_lag"
			}
		}
		mailboxDeliveryMode := snap.MailboxDeliveryMode()
		effectiveMailboxDeliveryMode := db.RuntimeHandleMailboxDeliveryMode(handle)
		if strings.TrimSpace(mailboxDeliveryMode) == "" {
			mailboxDeliveryMode = effectiveMailboxDeliveryMode
		} else if strings.TrimSpace(mailboxDeliveryMode) == runtimeagente.MailboxDeliveryBootstrapOnly &&
			strings.TrimSpace(effectiveMailboxDeliveryMode) == runtimeagente.MailboxDeliverySessionResume {
			mailboxDeliveryMode = effectiveMailboxDeliveryMode
		}
		rows = append(rows, runtimeDiagnosticoWorkerRow{
			HandleID:            handle.ID,
			Agent:               handle.Agente,
			Driver:              snap.Driver(),
			Transport:           snap.Transport(),
			RuntimeRef:          snap.RuntimeRef(),
			ChildPID:            snap.ChildPID(),
			State:               state,
			Alive:               alive,
			HeartbeatAt:         snap.HeartbeatTime(),
			UpdatedAt:           snap.UpdatedTime(),
			LastOutputAt:        snap.LastOutputTime(),
			LastProgressAt:      snap.LastProgressTime(),
			SessionRef:          snap.SessionRef(),
			ExternalSessionID:   valorVacioNoVacio(snap.ExternalSessionID(), sessionExternalIDForHandle(handle)),
			MailboxDeliveryMode: mailboxDeliveryMode,
			ExitError:           exitError,
			ManifestPath:        snap.ManifestPath,
			StatusPath:          snap.StatusPath,
			HeartbeatPath:       snap.HeartbeatPath,
			TmuxSession:         workerSnapshotManifestTMUXField(snap, true),
			TmuxPaneID:          workerSnapshotManifestTMUXField(snap, false),
		})
	}
	return rows
}

func sessionExternalIDForHandle(handle *db.RuntimeHandle) string {
	if handle == nil || handle.SesionID == nil || *handle.SesionID <= 0 {
		return ""
	}
	if db.DB == nil {
		return ""
	}
	sesion, err := db.GetSesionByID(*handle.SesionID)
	if err != nil || sesion == nil {
		return ""
	}
	return strings.TrimSpace(sesion.ExternalSessionID)
}

func valorVacioNoVacio(primary, fallback string) string {
	if strings.TrimSpace(primary) != "" {
		return strings.TrimSpace(primary)
	}
	return strings.TrimSpace(fallback)
}

func workerSnapshotManifestTMUXField(snap *runtimeagente.WorkerSnapshot, wantSession bool) string {
	if snap == nil || snap.Manifest == nil {
		return ""
	}
	if wantSession {
		return strings.TrimSpace(snap.Manifest.TmuxSession)
	}
	return strings.TrimSpace(snap.Manifest.TmuxPaneID)
}

var runtimeDiagnosticoTmuxHasSession = func(sessionName string) bool {
	sessionName = strings.TrimSpace(sessionName)
	if sessionName == "" {
		return false
	}
	tmuxPath, err := exec.LookPath("tmux")
	if err != nil {
		return false
	}
	cmd := exec.Command(tmuxPath, "has-session", "-t", sessionName)
	return cmd.Run() == nil
}

func runtimeDiagnosticoIssues(rows []runtimeDiagnosticoWorkerRow, orders []*db.RuntimeOrder, mailbox []*db.RuntimeMailboxMessage, checkpoints []*db.RuntimeCheckpoint) []runtimeDiagnosticoIssue {
	issues := make([]runtimeDiagnosticoIssue, 0)
	now := runtimeDiagnosticoNow()
	aliveSessions := map[string]bool{}
	aliveByAgent := map[string][]runtimeDiagnosticoWorkerRow{}
	for _, row := range rows {
		if row.Alive {
			aliveByAgent[strings.TrimSpace(row.Agent)] = append(aliveByAgent[strings.TrimSpace(row.Agent)], row)
		}
		if !strings.EqualFold(strings.TrimSpace(row.Transport), "tmux") {
			continue
		}
		if !row.Alive {
			continue
		}
		sessionName := strings.TrimSpace(row.TmuxSession)
		if sessionName == "" {
			continue
		}
		aliveSessions[sessionName] = true
	}
	for _, row := range rows {
		if row.State == "stale" && row.ExitError == "heartbeat_lag" {
			issues = append(issues, runtimeDiagnosticoIssue{
				Severity: "warn",
				Code:     "heartbeat_lag",
				Message:  fmt.Sprintf("%s handle=%d heartbeat atrasado (%s)", valorVacio(row.Agent), row.HandleID, valorVacio(row.RuntimeRef)),
			})
		}
		if row.Alive && runtimeDiagnosticoWorkerProgressStale(row, now) {
			issues = append(issues, runtimeDiagnosticoIssue{
				Severity: "warn",
				Code:     "worker_progress_stale",
				Message:  fmt.Sprintf("%s handle=%d sin progreso desde %s (%s)", valorVacio(row.Agent), row.HandleID, formatearRuntimeDiagTime(row.LastProgressAt), valorVacio(row.RuntimeRef)),
			})
		}
		if !strings.EqualFold(strings.TrimSpace(row.Transport), "tmux") {
			continue
		}
		sessionName := strings.TrimSpace(row.TmuxSession)
		if sessionName == "" {
			continue
		}
		exists := runtimeDiagnosticoTmuxHasSession(sessionName)
		switch {
		case row.Alive && !exists:
			issues = append(issues, runtimeDiagnosticoIssue{
				Severity: "fail",
				Code:     "tmux_session_missing",
				Message:  fmt.Sprintf("%s handle=%d sesión tmux ausente (%s)", valorVacio(row.Agent), row.HandleID, sessionName),
			})
		case !row.Alive && exists && !aliveSessions[sessionName]:
			issues = append(issues, runtimeDiagnosticoIssue{
				Severity: "warn",
				Code:     "orphan_tmux_session",
				Message:  fmt.Sprintf("%s handle=%d conserva sesión tmux huérfana (%s)", valorVacio(row.Agent), row.HandleID, sessionName),
			})
		}
	}
	issues = append(issues, runtimeDiagnosticoResumeIssues(now, aliveByAgent, orders, mailbox)...)
	issues = append(issues, runtimeDiagnosticoRestartLoopIssues(now, checkpoints)...)
	return issues
}

func runtimeDiagnosticoWorkerProgressStale(row runtimeDiagnosticoWorkerRow, now time.Time) bool {
	if !row.Alive {
		return false
	}
	if row.LastProgressAt == nil || row.LastProgressAt.IsZero() {
		return false
	}
	if now.Sub(row.LastProgressAt.UTC()) <= runtimeDiagnosticoWorkerProgressThreshold {
		return false
	}
	if row.LastOutputAt != nil && !row.LastOutputAt.IsZero() && now.Sub(row.LastOutputAt.UTC()) <= runtimeDiagnosticoWorkerProgressThreshold {
		return false
	}
	if row.UpdatedAt != nil && !row.UpdatedAt.IsZero() && now.Sub(row.UpdatedAt.UTC()) <= runtimeDiagnosticoWorkerProgressThreshold {
		return false
	}
	return true
}

func runtimeDiagnosticoResumeIssues(now time.Time, aliveByAgent map[string][]runtimeDiagnosticoWorkerRow, orders []*db.RuntimeOrder, mailbox []*db.RuntimeMailboxMessage) []runtimeDiagnosticoIssue {
	issues := make([]runtimeDiagnosticoIssue, 0)
	seenMailbox := map[string]bool{}
	for _, msg := range mailbox {
		if msg == nil || strings.TrimSpace(msg.Estado) != "pendiente" {
			continue
		}
		agente := strings.TrimSpace(msg.ToAgente)
		if agente == "" || seenMailbox[agente] {
			continue
		}
		if now.Sub(msg.CreatedAt.UTC()) < runtimeDiagnosticoResumePendingThreshold {
			continue
		}
		rows := aliveByAgent[agente]
		if !runtimeDiagnosticoHasSessionResumeWorker(rows) {
			continue
		}
		if runtimeDiagnosticoWorkerProgressedSince(rows, msg.CreatedAt.UTC()) {
			continue
		}
		seenMailbox[agente] = true
		issues = append(issues, runtimeDiagnosticoIssue{
			Severity: "warn",
			Code:     "session_resume_blocked",
			Message:  fmt.Sprintf("%s mantiene mailbox pendiente sin progreso tras session_resume desde %s", valorVacio(agente), msg.CreatedAt.Local().Format("2006-01-02 15:04:05")),
		})
	}

	seenOrders := map[string]bool{}
	for _, order := range orders {
		if order == nil {
			continue
		}
		if seenOrders[strings.TrimSpace(order.Agente)] {
			continue
		}
		if !runtimeDiagnosticoIsResumeOrder(order) {
			continue
		}
		if now.Sub(order.CreatedAt.UTC()) < runtimeDiagnosticoResumePendingThreshold {
			continue
		}
		rows := aliveByAgent[strings.TrimSpace(order.Agente)]
		if runtimeDiagnosticoWorkerProgressedSince(rows, order.CreatedAt.UTC()) {
			continue
		}
		seenOrders[strings.TrimSpace(order.Agente)] = true
		issues = append(issues, runtimeDiagnosticoIssue{
			Severity: "warn",
			Code:     "resume_order_pending",
			Message:  fmt.Sprintf("%s conserva orden %s pendiente sin progreso desde %s", valorVacio(order.Agente), valorVacio(order.Tipo), order.CreatedAt.Local().Format("2006-01-02 15:04:05")),
		})
	}
	return issues
}

func runtimeDiagnosticoRestartLoopIssues(now time.Time, checkpoints []*db.RuntimeCheckpoint) []runtimeDiagnosticoIssue {
	issues := make([]runtimeDiagnosticoIssue, 0)
	counts := map[string]int{}
	for _, cp := range checkpoints {
		if cp == nil {
			continue
		}
		if now.Sub(cp.CreatedAt.UTC()) > runtimeDiagnosticoRestartLoopWindow {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(cp.CheckpointKind), "stop") {
			continue
		}
		text := strings.ToLower(strings.TrimSpace(cp.Resumen + " " + cp.Source))
		if !strings.Contains(text, "worker_atascado") {
			continue
		}
		counts[strings.TrimSpace(cp.Agente)]++
	}
	for agente, count := range counts {
		if count < runtimeDiagnosticoRestartLoopCount {
			continue
		}
		issues = append(issues, runtimeDiagnosticoIssue{
			Severity: "warn",
			Code:     "restart_loop",
			Message:  fmt.Sprintf("%s acumula %d reinicios por worker_atascado en la última ventana de %s", valorVacio(agente), count, runtimeDiagnosticoRestartLoopWindow),
		})
	}
	return issues
}

func runtimeDiagnosticoHasSessionResumeWorker(rows []runtimeDiagnosticoWorkerRow) bool {
	for _, row := range rows {
		if !row.Alive {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(row.MailboxDeliveryMode), "session_resume") {
			return true
		}
	}
	return false
}

func runtimeDiagnosticoWorkerProgressedSince(rows []runtimeDiagnosticoWorkerRow, since time.Time) bool {
	for _, row := range rows {
		if !row.Alive {
			continue
		}
		if row.LastProgressAt != nil && row.LastProgressAt.UTC().After(since) {
			return true
		}
		if row.LastOutputAt != nil && row.LastOutputAt.UTC().After(since) {
			return true
		}
		if row.UpdatedAt != nil && row.UpdatedAt.UTC().After(since) {
			return true
		}
	}
	return false
}

func runtimeDiagnosticoIsResumeOrder(order *db.RuntimeOrder) bool {
	if order == nil {
		return false
	}
	switch strings.TrimSpace(order.Estado) {
	case "pendiente", "tomada", "ejecutando":
	default:
		return false
	}
	switch strings.TrimSpace(order.Tipo) {
	case "resume", "start":
		return true
	default:
		return false
	}
}

func imprimirRuntimeDiagnosticoWorkers(rows []runtimeDiagnosticoWorkerRow) error {
	if len(rows) == 0 {
		return nil
	}
	fmt.Printf("%-7s %-12s %-12s %-22s %-10s %-6s %-18s %-18s %-18s %-18s %s\n", "HANDLE", "AGENTE", "DRIVER", "RUNTIME", "STATE", "ALIVE", "HEARTBEAT", "PROGRESS", "SESSION", "DELIVERY", "ERROR")
	for _, row := range rows {
		driver := valorVacio(row.Driver)
		if transport := strings.TrimSpace(row.Transport); transport != "" {
			driver = driver + "/" + transport
		}
		runtimeRef := valorVacio(row.RuntimeRef)
		if row.ChildPID > 0 {
			runtimeRef = truncar(runtimeRef+" pid="+fmt.Sprintf("%d", row.ChildPID), 22)
		}
		fmt.Printf("%-7d %-12s %-12s %-22s %-10s %-6t %-18s %-18s %-18s %-18s %s\n",
			row.HandleID,
			truncar(row.Agent, 12),
			truncar(driver, 12),
			truncar(runtimeRef, 22),
			truncar(valorVacio(row.State), 10),
			row.Alive,
			formatearRuntimeDiagTime(row.HeartbeatAt),
			formatearRuntimeDiagTime(row.LastProgressAt),
			truncar(valorVacio(row.SessionRef), 18),
			truncar(valorVacio(row.MailboxDeliveryMode), 18),
			truncar(valorVacio(row.ExitError), 36),
		)
	}
	return nil
}

func imprimirRuntimeDiagnosticoIssues(issues []runtimeDiagnosticoIssue) error {
	for _, issue := range issues {
		if strings.TrimSpace(issue.Code) == "" {
			continue
		}
		icon := "[!!]"
		if strings.EqualFold(strings.TrimSpace(issue.Severity), "fail") {
			icon = "[XX]"
		}
		fmt.Printf("%s %-22s %s\n", icon, truncar(issue.Code, 22), issue.Message)
	}
	return nil
}

func formatearRuntimeDiagTime(ts *time.Time) string {
	if ts == nil || ts.IsZero() {
		return "—"
	}
	return ts.Local().Format("2006-01-02 15:04:05")
}

func init() {
	runtimeDiagnosticoCmd.Flags().String("agente", "", "Agente a diagnosticar")
	runtimeDiagnosticoCmd.Flags().String("proyecto", "", "Proyecto asociado al runtime")
	runtimeDiagnosticoCmd.Flags().Int("limit", 5, "Numero maximo de órdenes y checkpoints a mostrar")
	_ = runtimeDiagnosticoCmd.MarkFlagRequired("agente")
	runtimeCmd.AddCommand(runtimeDiagnosticoCmd)
}
