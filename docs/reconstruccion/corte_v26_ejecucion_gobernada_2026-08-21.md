# Corte V26: admisión gobernada de tools de lectura

Fecha: 2026-08-21. Estado: incremento WIP de `application` implementado y
ejercitado, sin wiring productivo ni acreditación. `TLS-12` continúa
`declared`; `AC-V26-TOOLS-SKILLS-SDK` continúa `planned`. Este corte no cambia
roadmap, capabilities ni evidencia.

## Autoridad y alcance

- capability principal: `TLS-12`, sobre contratos base locales de `TLS-01` y
  `TLS-05`, todavía no acreditados como V26;
- invariante: solo una selección curada exacta, autorizada y ligada a un
  WorkItem y ejecución vigentes puede alcanzar el connector;
- autoridad: `internal/application`; el catálogo filtra y el connector solo
  observa el request admitido;
- write-set de la corrección: `internal/application/{tool_execution.go,
  tool_execution_test.go,ports.go,orchestrator.go}` (solo hunks ToolExecution),
  `acceptance/v26_tool_execution_test.go` y este corte;
- fuera de alcance: adapters, bootstrap, configuración/SDK/schema, efectos,
  reserva durable, ledger de observación y `execution_service`.

Antes de editar se ejecutó:

```text
scripts/preflight_reutilizacion_legacy.sh --capability TLS-12 \
  --path internal/application/tool_execution.go --operation corregir \
  --task 'compactar y endurecer invocacion manual humana de tool curada: input raw acotado antes de hash/decode, coste observado conocido y binding exacto de WorkItem running a ejecucion vigente; servicio cerrado sin claim/fence verificable' \
  --function InvokeTool
```

El resultado advisory fue `reimplement`, sin función exacta reutilizable ni
permiso para copiar código legacy.

## Admisión corregida

El input crudo se rechaza si supera 64 KiB inmediatamente después de validar
disponibilidad/contexto/request ref. Esa comprobación precede a cualquier
digest, autorización, decode, canonización o JSON y no abre configuración ad
hoc.

La única ruta admitida es una invocación manual autenticada con
`PrincipalKindHuman`. El request aporta la `ToolRef` exacta
`tool:<ToolID>`, `ExecutionRef`, attempt, plan generation, AppSpec generation y
spec hash. `application` exige Goal, WorkItem y ejecución `running`, cruza el
proyecto, capability, ToolRef, ejecución actualmente ligada, generaciones,
attempt y hash contra `GoalRecord`, y rechaza refs de ejecución duplicadas.
Tras autorizar todos los permisos, relee el mismo sujeto durable justo antes
del connector. Pending, terminal, interrupted, superseded, reemplazada,
cruzada o cambiada durante la admisión fallan cerradas.

La autorización `goals.get` sigue siendo el gate inicial que permite releer el
Goal sin bypass RBAC. Por ello una tool registrada pero no curada puede dejar
exactamente ese receipt inicial; no deja autorizaciones de permisos de la tool,
llamada al connector ni artefacto. Catálogo incorrecto, input raw excesivo y
principal no humano fallan antes incluso de esa autorización.

El connector y la observación transportan el binding causal completo. La
validación de envelope y el receipt lo comparan exactamente, y los digests de
selección, invocación, idempotencia y observación lo incluyen. El receipt solo
puede construirse cuando todas las dimensiones de `ResourceUsage` son conocidas
y la calidad es `measured` o `exact`; `Known=0`, `unknown`, `estimated`, una
dimensión ausente o un bit desconocido no se contabilizan como coste cero.
Después se exige que el vector observado quepa en la cota declarada.

Se conservan validación/canonización de output, receipt exacto, inline bajo el
umbral y spill CAS. SQLite hace durable solo la admisión RBAC: un replay exacto
recupera sus receipts, vuelve a cruzar la ejecución vigente y reejecuta por
`read_reexecute`. Si la ejecución terminó o fue reemplazada, falla cerrado. El
receipt/output del connector no tiene todavía ledger durable.

Los errores públicos conservan únicamente el código máquina: una causa externa
del connector, schema o CAS no queda accesible mediante `Unwrap` y no puede
reintroducir payloads, rutas o secretos en la cadena de error.

## Diferimiento explícito de agentes

La invocación no crea, reclama, renueva ni transfiere `ActionClaim`, fence o
lease. Todo principal no humano, incluido `execution_service`, se rechaza antes
del connector. `NewExecutionAccess` y campos del payload no prueban claim,
fence o lease vigentes. La ejecución de tools por agentes queda diferida hasta
un puerto autoritativo y transaccional que acredite esas tres autoridades.

## Presupuesto y verificación reproducible

El sujeto productivo queda en P=448, bajo el techo 450: 382 líneas físicas de
`tool_execution.go`, 53 altas de `ports.go` y 13 altas de `orchestrator.go`.
El sujeto completo de verificación queda en V=500 exactas: 350 unitarias y 150
de aceptación. Los tests untracked carecen de baseline Git; estos censos
físicos sustituyen cualquier delta parcial como autoridad presupuestaria.

```bash
wc -l internal/application/tool_execution.go \
  internal/application/tool_execution_test.go \
  acceptance/v26_tool_execution_test.go
git diff --numstat -- internal/application/ports.go \
  internal/application/orchestrator.go

(
  cd internal/application
  mapfile -t v26_application_files < <(rg --files -g '*.go' | sort | rg -v '^provider_.*_test\.go$')
  go test -mod=vendor -count=1 "${v26_application_files[@]}" -run '^TestInvokeTool.*$'
  go test -mod=vendor -race -count=1 "${v26_application_files[@]}" -run '^TestInvokeTool.*$'
)
(
  cd acceptance
  mapfile -t v26_acceptance_files < <(rg --files -g '*.go' | sort | rg -v '^v25_preventive_handoff_test\.go$')
  go test -mod=vendor -count=1 "${v26_acceptance_files[@]}" -run '^(TestV26ToolExecutionContractBindsRunningSQLiteSubjectAndReplaysAdmission|TestV26TLS12CuratedCatalogNeverResolvesUnselectedOrCrossScopeSkill)$'
  go test -mod=vendor -race -count=1 "${v26_acceptance_files[@]}" -run '^(TestV26ToolExecutionContractBindsRunningSQLiteSubjectAndReplaysAdmission|TestV26TLS12CuratedCatalogNeverResolvesUnselectedOrCrossScopeSkill)$'
)
```

La regex focal ejecuta todos los `TestInvokeTool.*`. Application y aceptación
V26 pasaron normal y race con listas explícitas que excluyen solo el test WIP
V25 incompatible. Compilación, application/acceptance, vet, arquitectura y
`diff --check` se revalidaron. `go test .` permanece rojo por el ancla ajena de
`ports.go`, receipts Codex obsoletos y el script legacy untracked;
`./internal/... ./cmd/orquesta` falla solo en perfiles Codex no disponibles y
el toolchain Go configurado. V26 no aparece en esos fallos y no se corrigieron
fuera del write-set.

`HEAD` no contiene `internal/tooling/**` ni `sdk/tools/**`: este incremento
consume esos contratos concurrentes y solo será reproducible cuando registro,
catálogo curado y SDK formen el mismo candidato. No existe composición
productiva de `ToolExecutor`, connector, binding público ni ledger
transaccional de observación; faltan además el test/fixture canónicos del AC.

No existe connector productivo, E2E público, receipt de release ni evidencia
del mismo candidato. No se atribuye wiring, ejercicio productivo, acreditación
de `TLS-12` ni cierre de V26.
