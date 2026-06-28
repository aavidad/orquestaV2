# Implementación de mejoras Orquesta (cómo hacerlas)

Fecha: 2026-06-27.
Para: agente programador. Detalle de implementación de las mejoras de mayor
palanca del roadmap (`TAREA_OPES_ORQUESTA_ROADMAP_MEJORAS_2026-06-27.md`). Cada
bloque: ficheros, firmas, flujo y tests. Respeta el principio existente "el
trabajo útil debe llegar a review" (no convertir soft-rails en bloqueos de ACK).

---

## M1 (PRIORIDAD) — Robustez de railes/ACK: dejar de rechazar trabajo útil por tonterías

### Problema real (recurrente)
Railes y ACK han venido **rechazando entregas útiles por reglas mal construidas /
demasiado estrictas**, no por problemas reales. Hay que hacer la validación
**tolerante y correcta**: aceptar el trabajo bueno, bloquear solo lo genuinamente
inseguro/no identificable, y que los rechazos sean **informativos y recuperables**,
no fallos terminales por tecnicismos.

Fuentes de fragilidad concretas, todas en
`modulos/orquesta-runtime-codex/codex_ack_strict_validation_v0.go`:

1. **`schema_version` por igualdad exacta** (`:54`): cualquier versión distinta a
   `CodexAgentAckSchemaVersionV0` rechaza todo el ACK.
2. **Comando de test por match exacto de string** (`validateStrictTests`,
   `:151-152` con `codexAckStringInSetV0`): una diferencia de espacios/flags/ruta
   ⇒ `required_tests_mismatch` ⇒ rechazo.
3. **`ack.Tests == nil` ⇒ required** (`:75-76`) aunque haya evidencia de ficheros
   y notas.
4. **Cadena de igualdades de identidad** (`:82-88`): un solo desajuste de formato
   de ref (padre/subrol, token sobrante) ⇒ `correlation_mismatch`.
5. **Entrega útil sin `agent_ack.json`** tratada como fallo total (incidencia
   real: 5/6 subroles con ACK, trabajo válido, run marcada como fallo).

### Implementación (tolerancia + dos niveles + reconciliación)

**1) Normalización tolerante ANTES de comparar** (nuevo
`codex_ack_normalization_v0.go`):
- `schema_version`: aceptar **rango** `>= CodexAgentAckMinSchemaVersionV0` en vez
  de igualdad; solo rechazar si es más nuevo de lo soportado o irreconocible.
- Tests: comparar por **canonicalización** (trim, colapsar espacios, ordenar
  flags) y, mejor, por `test_ref`/fingerprint en vez de por el string crudo del
  comando. Función `canonicalTestCommandV0(string) string` y match sobre
  canónicos.
- Identidad: comparar refs con `equalRefTolerantV0(a, b)` que normaliza formatos
  conocidos equivalentes (padre/subrol) en vez de la cadena de `==`.

**2) Taxonomía de dos niveles** (refactor de `codexAckValidatorV0.add`):
```go
// distinguir bloqueo real vs rail informativo recuperable
type AckIssueSeverityV0 string
const (
    AckIssueHardV0 AckIssueSeverityV0 = "hard"   // inseguro/no identificable → no pasa
    AckIssueSoftV0 AckIssueSeverityV0 = "soft"   // recuperable → llega a review con aviso
)
func (v *codexAckValidatorV0) addSeverity(code, field, evidence string, sev AckIssueSeverityV0)
```
Reclasificar: `schema_version` desfasado-pero-legible, `tests` ausente con
evidencia de ficheros, `required_tests_mismatch` por formato, drift de write-set
⇒ **soft**. Mantener **hard** solo: status que no aporta trabajo revisable, paths
inseguros (`codexAckHasForbiddenArtifactPathV0`), identidad **irreconciliable**
tras normalizar. La entrega con issues solo-soft **debe llegar a review**.

**3) Reconciliación desde evidencia cuando el ACK falta o es parcial** (nuevo
`codex_ack_reconcile_v0.go`, consumido por el reconciliador de runs):
- Si el proceso terminó y existen artefactos + evidencia de tests pero **no hay
  `agent_ack.json`** (o le faltan campos reconstruibles), **sintetizar** un ACK
  reconciliado con status `completed_without_ack_file` y los refs derivados del
  packet, en vez de marcar fallo. Reutiliza
  `codexReceiptAckIssuesOnlyReviewableFailedTestEvidenceV0`
  (`orquesta-runtime-codex-delivery/worktree_ack_issue_v0.go:10`, hoy muerta).

**4) Deriva de write-set como rail informativo (no rechazo)**: cablear
`codexAckPathAllowedByWriteSetV0`/`codexAckPathMatchesWriteSetEntryV0`/
`codexAckPathGlobstarMatchV0` (hoy muertas, `codex_ack_path_policy_v0.go:216,237`)
en el review gate produciendo `evidence-ref-review-writeset-drift` **soft**,
respetando el principio ya documentado (`codex_ack_strict_validation_v0.go:128-130`).

### Guardia contra regresión: property tests de las propias reglas
Como el problema fue "reglas mal construidas", añadir **tests de propiedad** que
fijen el contrato de las reglas:
- Todo ACK canónico bien formado **siempre pasa** (sin issues hard).
- Variaciones cosméticas (espacios en tests, schema una menor, formato de ref
  equivalente) **no** producen issues hard.
- Solo las condiciones genuinamente inseguras producen hard.
Esto convierte "no rechazar por tonterías" en invariante verificable.

### Avance Orquesta 2026-06-28 - refs ACK tolerantes acotadas

Se cerró la parte de comparación de refs/correlación sin marcar M1 completo:

- `codexAckRefTolerantEqualV0` y `codexAckComparableRefV0` aceptan solo
  variaciones cosméticas recuperables: espacios y envoltorios simples
  `` `ref` ``, `"ref"` y `'ref'`;
- regular y strict reutilizan `codexAgentAckCorrelationMatchesSpecV0`;
- cuando la identidad cuadra, el ACK se canoniza contra el packet/spec;
- las colisiones padre/subrol y padre/`child_task_refs` siguen siendo hard
  incluso si vienen maquilladas con espacios o backticks;
- se añadió property test acotado para combinaciones de wrappers cosméticos.

Cobertura añadida:

- `TestCodexAgentAckReceiptV0ToleraRefsConEnvoltorioCosmeticoV0`
- `TestCodexAgentAckReceiptV0PropiedadRefsCosmeticasValidanV0`
- `TestCodexAgentAckReceiptV0DetectaColisionSubrolConEnvoltorioCosmeticoV0`
- `TestCodexAgentAckReceiptV0DetectaColisionChildConEnvoltorioCosmeticoV0`
- equivalentes strict en `codex_ack_strict_validation_v0_test.go`.

Validación ejecutada:

- `go test -count=1 ./modulos/orquesta-runtime-codex -run 'TestCodexAgentAckReceiptV0|TestStrictCompletedCodexAgentAckV0'`
- `go test -count=1 ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-app-codex-stack`

Sigue pendiente de M1: reconciliación completa cuando falta `agent_ack.json`
como contrato transversal. Hoy la recuperación existe sobre todo en
`orquesta-runtime-codex-delivery` y `orquesta-app-codex-stack`; no se ha movido
al conector puro.

### Resultado
El trabajo útil deja de caerse por tecnicismos; los rechazos pasan a ser avisos
recuperables que llegan a review; y la verificación sigue siendo **código puro**
(sin coste de tokens), cableando U1000 en vez de borrarlo.

---

## M2 — Observación evidence-first con guarda de "sin cambios" (hace barato T1)

### Estado real
El observador `CodexGoalObserverV0.ObserveGoalWorkV0`
(`modulos/orquesta-runtime-codex-goal/packet_v0.go:334`) delega en
`Observer.ObserveCodexGoalV0` y deriva estado del **receipt** (ya es
evidence-based). El coste a evitar es **re-observar en cada tick** una run que no
ha cambiado, cuando el ticker de T1 corra cada pocos segundos.

### Implementación
1. En `orquesta-goal`, añadir a `ObserveActiveGoalWorksV0`
   (`observe_active_v0.go`) una **guarda de fingerprint** por run antes de
   observar:
   ```go
   // observe_active_skip_v0.go (nuevo)
   type GoalObservationFingerprintV0 struct {
       RunRef        string
       AckFilesHash  string // hash de (paths + mtime) del agent_ack.json y artefactos
       ProcessAlive  bool
       LastStatus    string
   }
   func GoalObservationUnchangedV0(prev, current GoalObservationFingerprintV0) bool
   ```
2. El ticker del runtime (T1, `runGoalObservationTickV0`) mantiene un mapa
   `map[runRef]GoalObservationFingerprintV0` y **solo llama al observador** si el
   fingerprint cambió (nuevo ACK, proceso murió, o estado previo no terminal sin
   evidencia nueva). El fingerprint se calcula con E/S barata (stat de fichero +
   `ProcessRegistry.ResolveAgentProcessV0`), nunca con LLM.
3. Orden de evaluación dentro de la observación (barato→caro), cortando en el
   primer veredicto: (a) ¿existe `agent_ack.json`? (b) ¿proceso vivo? (c)
   ¿required tests con evidencia? (d) solo si ambiguo, observación completa.

### Tests
- `GoalObservationUnchangedV0`: mismo fingerprint → true (se salta); ACK nuevo o
  proceso caído → false (observa).
- Ticker: N ticks sobre una run estable → 1 sola observación real (contador de
  llamadas al observador). Sincronizar con `waitAsyncWorkV0`.

### Resultado
La autonomía (vigilar hasta cierre) deja de tener coste por tick; solo se trabaja
cuando hay evidencia nueva.

---

## M3 — Endpoint único de estado global accionable (Tier 1)

### Implementación
1. Reusar `ClassifyRunLivenessV0` (T2, `orquesta-run-coordinator`) y la lista de
   goals activos (`ListGoalWorkStatesV0`) para construir una proyección:
   ```go
   // modulos/orquesta-mcp/queue_global_status_projection_v0.go (nuevo)
   type QueueGlobalStatusItemV0 struct {
       RunRef           string
       Status           string // ready|running_live|running_stale|waiting_outbox|...
       NeedsAction      bool
       RecommendedAction string
       EvidenceRefs     []string
   }
   type QueueGlobalStatusV0 struct {
       Items        []QueueGlobalStatusItemV0
       WillFinishAlone bool   // true si ninguna item NeedsAction
       Summary       map[string]int
   }
   func BuildQueueGlobalStatusV0(...) QueueGlobalStatusV0
   ```
2. Nuevo handler HTTP `GET/POST /api/v0/queue/global-status` con el patrón de
   timeout acotado ya usado (`*_http_v0.go`, 2s → cuerpo siempre). Sin `run_ref`
   obligatorio.
3. `WillFinishAlone` responde la pregunta humana del operador; cada item con
   `NeedsAction=true` trae `RecommendedAction` (`retry|reencolar|cancel_stale|
   restart_observer|repair_runtime`).

### Tests
- Cola mixta (ready/running_live/running_stale/waiting_outbox) → summary correcto,
  `WillFinishAlone=false` si hay stale; respuesta acotada con cuerpo bajo presión.

---

## M4 — Auditoría como fuente de auto-mejora (O-1, capacidad estrella)

### Implementación
1. Nuevo adaptador "fuente de backlog" que el planner de idle self-improvement ya
   consume (`cmd/orquesta-server/idle_self_improvement_backlog_planner_v0.go`,
   `loadBacklogSectionsV0`):
   ```go
   // cmd/orquesta-server/self_audit_backlog_source_v0.go (nuevo)
   func selfAuditBacklogSectionsV0(ctx context.Context, projectDir string) []idleSelfImprovementBacklogSectionV0
   ```
   que ejecuta de forma acotada (timeout, sandbox) `go vet`, `staticcheck`,
   `govulncheck` y `go test -race` sobre el propio repo y **traduce cada hallazgo
   a una sección de backlog** con `Ref` estable (hash del hallazgo), criterio de
   aceptación ("staticcheck no reporta X") y write-set sugerido (el fichero del
   hallazgo).
2. Deduplicar con la maquinaria existente
   (`idleSelfImprovementDedupeBacklogProposalsV0`) para no reproponer lo mismo.
3. Gating: detrás de flag `ORQUESTA_SELF_AUDIT_BACKLOG_ENABLED` (default false al
   principio; subir cuando esté probado). El resto del lazo (lanzar goal de
   auto-mejora + observador T1) ya existe.

### Tests
- Con un fichero que dispara un hallazgo conocido de staticcheck, la fuente emite
  una sección con `Ref` estable y criterio verificable; segunda pasada no la
  duplica.
- Avance 2026-06-28: ademas del planner unitario, queda cubierto el camino
  operacional goal-first del runtime. `TestRuntimeV0SelfAuditBacklogGoalFirstLanzaSpecOperacionalV0`
  arranca `RuntimeV0` aislado con `IdleSelfImprovementGoalFirst=true`, planner
  self-audit opt-in y launcher Goal fake; verifica que el hallazgo de
  `staticcheck` se transforma en `GoalWorkSpecV0` con write-set, required test,
  contexto `self_audit://staticcheck`, cierre por tests requeridos y estado goal
  persistido, sin proveedor Codex real ni fallback legacy.

### Resultado
Orquesta detecta sus propios fallos y los mete en su backlog goal-first: se audita
y se arregla solo, usando el cerebro de auto-mejora ya cableado.

---

## M5 — Dry-run del wizard con coste estimado (W-1)

### Implementación
1. Reusar `BuildExternalWorkGoalWorkSpecV0`
   (`modulos/orquesta-external-work-run`) para, **sin lanzar**, producir el
   `GoalWorkSpec` y exponerlo en un endpoint de preview:
   ```go
   // modulos/orquesta-mcp/external_work_dry_run_v0.go (nuevo)
   type ExternalWorkDryRunResultV0 struct {
       Spec         orquestagoal.GoalWorkSpecV0
       WriteSet     []string
       RequiredTests []string
       Model        string // de la matriz modelo/razonamiento
       EstTokens    int
       EstCostUSD   float64
       EstWallClock string
       Issues       []MCPExternalWorkRunIssueV0
   }
   func BuildExternalWorkDryRunV0(...) ExternalWorkDryRunResultV0
   ```
2. La estimación de modelo sale de la matriz existente (docs/matriz; adaptador de
   capacidad `capacity_policy_adapter_v0.go`). `EstTokens` con una heurística por
   tamaño de spec/subroles (parametrizable), no exacta.
3. En el wizard web (`modulos/orquesta-web`, flujo /nueva-app) añadir un paso
   "Revisar antes de lanzar" que llama al preview y muestra spec + write-set +
   coste estimado, con botón de confirmación. Mantener i18n ES/EN y accesibilidad
   ya presentes.

### Tests
- `BuildExternalWorkDryRunV0` devuelve el mismo spec que el lanzamiento real para
  la misma entrada (paridad), más estimaciones > 0.
- Web: el paso de preview no lanza agentes (verifica que no se crea run/runtime).

---

## Orden sugerido e interdependencias
- **M1 primero**: es el dolor recurrente (railes/ACK rechazando trabajo útil).
  Robustez + reconciliación desde evidencia + property tests de las reglas.
  Desbloquea que las olas cierren sin intervención.
- **M2** es prerrequisito de coste del lazo (va con T1/T2).
- **M3** depende de `ClassifyRunLivenessV0` (T2).
- **M4** depende de que T1 (observador) cierre los goals que lanza.
- **M5** es independiente; puede ir en paralelo.

Nota: M1 se apoya en código hoy muerto que hay que **cablear, no borrar**
(`worktree_ack_issue_v0.go:10`, `codex_ack_path_policy_v0.go:216,237`); coordinar
con T7 del encargo (decisión cablear/borrar del Grupo B) para no eliminarlo antes.

## Comprobación final
```bash
go build ./... && go vet ./...
go test ./... && go test -race ./...
staticcheck ./... && govulncheck ./...
```
