package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"orquesta/runtimeagente"
)

const (
	agenteScoreNeutral              = 5.0
	agenteScoreObservedAlpha        = 0.35
	agenteScoreConfidenceStep       = 10.0
	agenteScoreMaxConfidence        = 0.90
	agenteScoreRecencyHalfLifeDays  = 90.0
	agenteScoreMinOperationalWeight = 0.0001
)

var agenteScoreMateriasCanonicas = []string{
	"codigo",
	"documentacion",
	"orquestacion",
	"brainstorming",
	"analisis",
	"revision",
	"frontend",
	"testing",
	"arquitectura",
	"infraestructura",
	"seguridad",
	"integraciones",
	"depuracion",
	"datos",
	"producto",
}

type AgenteScoreLocal struct {
	ID              int64
	Agente          string
	ConectorSlug    string
	Materia         string
	ScoreBase       float64
	ScoreObservado  float64
	ScoreTotal      float64
	Confianza       float64
	Muestras        int
	Exitos          int
	Benchmarks      int
	MetadataJSON    string
	LastBenchmarkAt *time.Time
	LastObservedAt  *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func MateriasScoreAgenteCanonicas() []string {
	out := make([]string, len(agenteScoreMateriasCanonicas))
	copy(out, agenteScoreMateriasCanonicas)
	return out
}

func NormalizarMateriaScoreAgente(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	switch raw {
	case "", "general":
		return "codigo"
	case "codigo", "code", "implementacion", "correccion", "microprogramacion":
		return "codigo"
	case "doc", "docs", "documentacion", "documentation":
		return "documentacion"
	case "orquestacion", "orchestration", "orquestar":
		return "orquestacion"
	case "brainstorm", "brainstorming", "ideacion", "ideación":
		return "brainstorming"
	case "analisis", "analysis", "especificacion", "especificación":
		return "analisis"
	case "revision", "review":
		return "revision"
	case "frontend", "ui", "ux":
		return "frontend"
	case "testing", "tests", "qa":
		return "testing"
	case "arquitectura", "architecture":
		return "arquitectura"
	case "infra", "infraestructura", "infrastructure", "devops":
		return "infraestructura"
	case "seguridad", "security", "hardening", "secure":
		return "seguridad"
	case "integracion", "integraciones", "integration", "integrations":
		return "integraciones"
	case "debug", "debugging", "depuracion", "depurar":
		return "depuracion"
	case "datos", "data", "bd", "database", "persistencia":
		return "datos"
	case "producto", "product", "ux_producto":
		return "producto"
	default:
		return raw
	}
}

func MateriaScoreDesdePipeline(fase, perfilTarea, carril string) string {
	perfil := NormalizarMateriaScoreAgente(perfilTarea)
	if perfil != "" {
		switch perfil {
		case "codigo", "revision", "analisis", "documentacion", "orquestacion", "brainstorming", "frontend", "testing", "arquitectura", "infraestructura", "seguridad", "integraciones", "depuracion", "datos", "producto":
			return perfil
		}
	}
	fase = strings.ToLower(strings.TrimSpace(fase))
	carril = strings.ToLower(strings.TrimSpace(carril))
	switch {
	case carril == "microprogramacion_local":
		return "codigo"
	case carril == "revision_diff":
		return "revision"
	case carril == "determinista_app":
		return "orquestacion"
	case fase == "revision":
		return "revision"
	case fase == "especificacion":
		return "analisis"
	case fase == "integracion":
		return "orquestacion"
	default:
		return "codigo"
	}
}

func PesosScoreMateriaDesdePipeline(fase, perfilTarea, carril string) map[string]float64 {
	switch MateriaScoreDesdePipeline(fase, perfilTarea, carril) {
	case "revision":
		return map[string]float64{
			"revision":  0.50,
			"analisis":  0.20,
			"codigo":    0.15,
			"testing":   0.10,
			"seguridad": 0.05,
		}
	case "analisis":
		return map[string]float64{
			"analisis":      0.35,
			"brainstorming": 0.25,
			"arquitectura":  0.20,
			"producto":      0.10,
			"documentacion": 0.10,
		}
	case "orquestacion":
		return map[string]float64{
			"orquestacion":    0.30,
			"infraestructura": 0.30,
			"arquitectura":    0.15,
			"integraciones":   0.10,
			"analisis":        0.10,
			"producto":        0.05,
		}
	case "documentacion":
		return map[string]float64{
			"documentacion": 0.55,
			"analisis":      0.20,
			"producto":      0.15,
			"codigo":        0.10,
		}
	case "frontend":
		return map[string]float64{
			"frontend":      0.45,
			"producto":      0.20,
			"codigo":        0.15,
			"testing":       0.10,
			"documentacion": 0.10,
		}
	case "testing":
		return map[string]float64{
			"testing":    0.45,
			"codigo":     0.25,
			"revision":   0.15,
			"depuracion": 0.15,
		}
	case "arquitectura":
		return map[string]float64{
			"arquitectura":    0.45,
			"analisis":        0.20,
			"orquestacion":    0.15,
			"infraestructura": 0.10,
			"producto":        0.10,
		}
	case "infraestructura":
		return map[string]float64{
			"infraestructura": 0.45,
			"orquestacion":    0.20,
			"seguridad":       0.15,
			"integraciones":   0.10,
			"datos":           0.10,
		}
	case "seguridad":
		return map[string]float64{
			"seguridad":       0.45,
			"revision":        0.20,
			"infraestructura": 0.15,
			"codigo":          0.10,
			"testing":         0.10,
		}
	case "integraciones":
		return map[string]float64{
			"integraciones": 0.40,
			"codigo":        0.20,
			"datos":         0.15,
			"documentacion": 0.15,
			"testing":       0.10,
		}
	case "depuracion":
		return map[string]float64{
			"depuracion": 0.45,
			"codigo":     0.20,
			"testing":    0.20,
			"revision":   0.15,
		}
	case "datos":
		return map[string]float64{
			"datos":           0.40,
			"codigo":          0.20,
			"infraestructura": 0.15,
			"integraciones":   0.15,
			"testing":         0.10,
		}
	case "producto":
		return map[string]float64{
			"producto":      0.40,
			"brainstorming": 0.20,
			"analisis":      0.20,
			"documentacion": 0.10,
			"frontend":      0.05,
			"integraciones": 0.05,
		}
	default:
		return map[string]float64{
			"codigo":        0.45,
			"testing":       0.20,
			"arquitectura":  0.10,
			"depuracion":    0.10,
			"seguridad":     0.05,
			"documentacion": 0.05,
			"analisis":      0.05,
		}
	}
}

func AgenteEsLocalPuntuable(agente, conector string) bool {
	conector = strings.TrimSpace(conector)
	if conector == "" {
		conector = runtimeagente.ConectorPorDefectoAgente(strings.TrimSpace(agente))
	}
	return runtimeagente.EsConectorFamiliaOllama(conector, "")
}

func GarantizarScoresLocalesAgente(agente, conector string) error {
	agente = strings.TrimSpace(agente)
	conector = strings.TrimSpace(conector)
	if agente == "" || !AgenteEsLocalPuntuable(agente, conector) {
		return nil
	}
	if conector == "" {
		conector = runtimeagente.ConectorPorDefectoAgente(agente)
	}
	for _, materia := range MateriasScoreAgenteCanonicas() {
		if _, err := garantizarScoreLocalAgenteMateria(agente, conector, materia); err != nil {
			return err
		}
	}
	return nil
}

func RegistrarBenchmarkScoreAgenteLocal(agente, conector, materia string, score float64, detalle string) (*AgenteScoreLocal, error) {
	item, err := garantizarScoreLocalAgenteMateria(agente, conector, materia)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, fmt.Errorf("score local no disponible")
	}
	item.ScoreBase = clampAgenteScore(score)
	item.ScoreTotal = recomputarScoreTotalAgenteLocal(item)
	item.Benchmarks++
	now := time.Now().UTC()
	item.LastBenchmarkAt = &now
	item.MetadataJSON = mergeAgenteScoreMetadata(item.MetadataJSON, map[string]any{
		"ultimo_benchmark_detalle": strings.TrimSpace(detalle),
		"ultimo_benchmark_score":   item.ScoreBase,
		"ultimo_benchmark_at":      now.Format(time.RFC3339),
	})
	if err := guardarAgenteScoreLocal(item); err != nil {
		return nil, err
	}
	return GetAgenteScoreLocal(agente, conector, materia)
}

func RegistrarObservacionScoreAgenteLocal(agente, conector, materia string, score float64, exito bool, detalle string) (*AgenteScoreLocal, error) {
	item, err := garantizarScoreLocalAgenteMateria(agente, conector, materia)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, fmt.Errorf("score local no disponible")
	}
	score = clampAgenteScore(score)
	if item.Muestras <= 0 {
		item.ScoreObservado = score
	} else {
		item.ScoreObservado = roundAgenteScore(item.ScoreObservado*(1.0-agenteScoreObservedAlpha) + score*agenteScoreObservedAlpha)
	}
	item.Muestras++
	if exito {
		item.Exitos++
	}
	item.Confianza = roundAgenteConfidence(math.Min(agenteScoreMaxConfidence, float64(item.Muestras)/agenteScoreConfidenceStep))
	now := time.Now().UTC()
	item.LastObservedAt = &now
	item.ScoreTotal = recomputarScoreTotalAgenteLocal(item)
	item.MetadataJSON = mergeAgenteScoreMetadata(item.MetadataJSON, map[string]any{
		"ultima_observacion_detalle": strings.TrimSpace(detalle),
		"ultima_observacion_score":   score,
		"ultima_observacion_exito":   exito,
		"ultima_observacion_at":      now.Format(time.RFC3339),
	})
	if err := guardarAgenteScoreLocal(item); err != nil {
		return nil, err
	}
	return GetAgenteScoreLocal(agente, conector, materia)
}

func GetAgenteScoreLocal(agente, conector, materia string) (*AgenteScoreLocal, error) {
	agente = strings.TrimSpace(agente)
	conector = normalizarConectorScoreAgente(agente, conector)
	materia = NormalizarMateriaScoreAgente(materia)
	if agente == "" || conector == "" || materia == "" {
		return nil, nil
	}
	row := DB.QueryRow(`
		SELECT id, agente, conector_slug, materia, score_base, score_observado, score_total, confianza,
		       muestras, exitos, benchmarks, metadata_json, last_benchmark_at, last_observed_at, created_at, updated_at
		FROM agente_scores_locales
		WHERE agente = ? AND conector_slug = ? AND materia = ?`,
		agente, conector, materia,
	)
	return scanAgenteScoreLocal(row)
}

func ListarAgenteScoresLocales(agente *string, conector *string) ([]*AgenteScoreLocal, error) {
	if items, ok, err := listarAgenteScoresLocalesPrepareLiteReadOnly(agente, conector); ok {
		return items, err
	}
	var (
		rows *sql.Rows
		err  error
	)
	switch {
	case agente != nil && strings.TrimSpace(*agente) != "" && conector != nil && strings.TrimSpace(*conector) != "":
		rows, err = DB.Query(`
			SELECT id, agente, conector_slug, materia, score_base, score_observado, score_total, confianza,
			       muestras, exitos, benchmarks, metadata_json, last_benchmark_at, last_observed_at, created_at, updated_at
			FROM agente_scores_locales
			WHERE agente = ? AND conector_slug = ?
			ORDER BY agente, conector_slug, materia`,
			strings.TrimSpace(*agente), strings.TrimSpace(*conector),
		)
	case agente != nil && strings.TrimSpace(*agente) != "":
		rows, err = DB.Query(`
			SELECT id, agente, conector_slug, materia, score_base, score_observado, score_total, confianza,
			       muestras, exitos, benchmarks, metadata_json, last_benchmark_at, last_observed_at, created_at, updated_at
			FROM agente_scores_locales
			WHERE agente = ?
			ORDER BY agente, conector_slug, materia`,
			strings.TrimSpace(*agente),
		)
	default:
		rows, err = DB.Query(`
			SELECT id, agente, conector_slug, materia, score_base, score_observado, score_total, confianza,
			       muestras, exitos, benchmarks, metadata_json, last_benchmark_at, last_observed_at, created_at, updated_at
			FROM agente_scores_locales
			ORDER BY agente, conector_slug, materia`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*AgenteScoreLocal
	for rows.Next() {
		item, err := scanAgenteScoreLocal(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func listarAgenteScoresLocalesPrepareLiteReadOnly(agente *string, conector *string) ([]*AgenteScoreLocal, bool, error) {
	raw, ok, err := openPrepareLiteReadOnlyLocal()
	if !ok || err != nil {
		return nil, ok, err
	}
	defer raw.Close()

	var (
		rows *sql.Rows
		query = `
			SELECT id, agente, conector_slug, materia, score_base, score_observado, score_total, confianza,
			       muestras, exitos, benchmarks, metadata_json, last_benchmark_at, last_observed_at, created_at, updated_at
			FROM agente_scores_locales`
		args []any
	)
	switch {
	case agente != nil && strings.TrimSpace(*agente) != "" && conector != nil && strings.TrimSpace(*conector) != "":
		query += ` WHERE agente = ? AND conector_slug = ? ORDER BY agente, conector_slug, materia`
		args = append(args, strings.TrimSpace(*agente), strings.TrimSpace(*conector))
	case agente != nil && strings.TrimSpace(*agente) != "":
		query += ` WHERE agente = ? ORDER BY agente, conector_slug, materia`
		args = append(args, strings.TrimSpace(*agente))
	default:
		query += ` ORDER BY agente, conector_slug, materia`
	}
	rows, err = raw.Query(query, args...)
	if err != nil {
		return nil, true, err
	}
	defer rows.Close()
	var out []*AgenteScoreLocal
	for rows.Next() {
		item, err := scanAgenteScoreLocal(rows)
		if err != nil {
			return nil, true, err
		}
		out = append(out, item)
	}
	return out, true, rows.Err()
}

func FitnessAgenteLocal(agente, conector string, pesos map[string]float64) (float64, error) {
	agente = strings.TrimSpace(agente)
	conector = normalizarConectorScoreAgente(agente, conector)
	if agente == "" || conector == "" || !AgenteEsLocalPuntuable(agente, conector) {
		return 0, nil
	}
	if err := GarantizarScoresLocalesAgente(agente, conector); err != nil {
		return 0, err
	}
	agenteRef := agente
	conectorRef := conector
	items, err := ListarAgenteScoresLocales(&agenteRef, &conectorRef)
	if err != nil {
		return 0, err
	}
	if len(items) == 0 {
		return agenteScoreNeutral, nil
	}
	if len(pesos) == 0 {
		pesos = map[string]float64{"codigo": 1.0}
	}
	byMateria := make(map[string]*AgenteScoreLocal, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		byMateria[NormalizarMateriaScoreAgente(item.Materia)] = item
	}
	var totalPeso, totalScore float64
	for materia, peso := range pesos {
		peso = math.Max(0, peso)
		if peso <= agenteScoreMinOperationalWeight {
			continue
		}
		materia = NormalizarMateriaScoreAgente(materia)
		totalPeso += peso
		if item := byMateria[materia]; item != nil {
			totalScore += peso * recomputarScoreTotalAgenteLocal(item)
		} else {
			totalScore += peso * agenteScoreNeutral
		}
	}
	if totalPeso <= agenteScoreMinOperationalWeight {
		return agenteScoreNeutral, nil
	}
	return roundAgenteScore(totalScore / totalPeso), nil
}

func scanAgenteScoreLocal(s scanner) (*AgenteScoreLocal, error) {
	var (
		item      AgenteScoreLocal
		lastBench sql.NullTime
		lastObs   sql.NullTime
	)
	if err := s.Scan(
		&item.ID, &item.Agente, &item.ConectorSlug, &item.Materia, &item.ScoreBase, &item.ScoreObservado, &item.ScoreTotal, &item.Confianza,
		&item.Muestras, &item.Exitos, &item.Benchmarks, &item.MetadataJSON, &lastBench, &lastObs, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if lastBench.Valid {
		item.LastBenchmarkAt = &lastBench.Time
	}
	if lastObs.Valid {
		item.LastObservedAt = &lastObs.Time
	}
	item.ScoreTotal = recomputarScoreTotalAgenteLocal(&item)
	return &item, nil
}

func garantizarScoreLocalAgenteMateria(agente, conector, materia string) (*AgenteScoreLocal, error) {
	agente = strings.TrimSpace(agente)
	conector = normalizarConectorScoreAgente(agente, conector)
	materia = NormalizarMateriaScoreAgente(materia)
	if agente == "" || conector == "" || materia == "" {
		return nil, nil
	}
	if !AgenteEsLocalPuntuable(agente, conector) {
		return nil, nil
	}
	sqlText := buildUpsertValuesSQL(
		CurrentStorageDriver(),
		"agente_scores_locales",
		[]string{"agente", "conector_slug", "materia", "score_base", "score_observado", "score_total", "confianza", "muestras", "exitos", "benchmarks", "metadata_json"},
		[]string{"agente", "conector_slug", "materia"},
		[]upsertAssignment{
			{Column: "updated_at", Expr: "CURRENT_TIMESTAMP"},
		},
	)
	if _, err := DB.Exec(sqlText,
		agente, conector, materia,
		agenteScoreNeutral, agenteScoreNeutral, agenteScoreNeutral, 0.0, 0, 0, 0, "{}",
	); err != nil {
		return nil, err
	}
	return GetAgenteScoreLocal(agente, conector, materia)
}

func guardarAgenteScoreLocal(item *AgenteScoreLocal) error {
	if item == nil {
		return fmt.Errorf("score local nil")
	}
	if item.ID <= 0 {
		return fmt.Errorf("score local sin id")
	}
	var lastBench any
	if item.LastBenchmarkAt != nil && !item.LastBenchmarkAt.IsZero() {
		lastBench = item.LastBenchmarkAt.UTC()
	}
	var lastObs any
	if item.LastObservedAt != nil && !item.LastObservedAt.IsZero() {
		lastObs = item.LastObservedAt.UTC()
	}
	_, err := DB.Exec(`
		UPDATE agente_scores_locales
		SET score_base=?, score_observado=?, score_total=?, confianza=?,
		    muestras=?, exitos=?, benchmarks=?, metadata_json=?, last_benchmark_at=?, last_observed_at=?
		WHERE id=?`,
		item.ScoreBase, item.ScoreObservado, item.ScoreTotal, item.Confianza,
		item.Muestras, item.Exitos, item.Benchmarks, item.MetadataJSON, lastBench, lastObs, item.ID,
	)
	return err
}

func normalizarConectorScoreAgente(agente, conector string) string {
	conector = strings.TrimSpace(conector)
	if conector != "" {
		return conector
	}
	return strings.TrimSpace(runtimeagente.ConectorPorDefectoAgente(strings.TrimSpace(agente)))
}

func clampAgenteScore(v float64) float64 {
	switch {
	case v < 0:
		return 0
	case v > 10:
		return 10
	default:
		return roundAgenteScore(v)
	}
}

func roundAgenteScore(v float64) float64 {
	return math.Round(v*10) / 10
}

func roundAgenteConfidence(v float64) float64 {
	return math.Round(v*100) / 100
}

func recomputarScoreTotalAgenteLocal(item *AgenteScoreLocal) float64 {
	if item == nil {
		return agenteScoreNeutral
	}
	base := clampAgenteScore(item.ScoreBase)
	observado := clampAgenteScore(item.ScoreObservado)
	confianza := math.Max(0, math.Min(agenteScoreMaxConfidence, item.Confianza))
	if item.LastObservedAt != nil && !item.LastObservedAt.IsZero() {
		days := time.Since(item.LastObservedAt.UTC()).Hours() / 24.0
		if days > 0 {
			factor := math.Exp(-math.Ln2 * days / agenteScoreRecencyHalfLifeDays)
			confianza = roundAgenteConfidence(confianza * factor)
		}
	}
	if item.Muestras <= 0 && item.Benchmarks <= 0 {
		return base
	}
	return roundAgenteScore(base*(1-confianza) + observado*confianza)
}

func mergeAgenteScoreMetadata(raw string, overlay map[string]any) string {
	payload := map[string]any{}
	raw = strings.TrimSpace(raw)
	if raw != "" && raw != "{}" {
		_ = json.Unmarshal([]byte(raw), &payload)
	}
	for k, v := range overlay {
		if strings.TrimSpace(k) == "" || v == nil {
			continue
		}
		payload[k] = v
	}
	if len(payload) == 0 {
		return "{}"
	}
	keys := make([]string, 0, len(payload))
	for k := range payload {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	ordered := make(map[string]any, len(keys))
	for _, k := range keys {
		ordered[k] = payload[k]
	}
	out, err := json.Marshal(ordered)
	if err != nil {
		return "{}"
	}
	return string(out)
}
