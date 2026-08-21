# Corte V25 AGT-10: pipeline secuencial de proveedores

Fecha: 2026-08-21.

Estado honesto: contrato puro revisado y ejercitado. `AGT-10` permanece
`declared` y `AC-V25-PROVIDER-ADAPTERS` permanece `planned`. No existe wiring
de lanzamiento, adapter real ni E2E real de provider; este corte no acredita
V25.

## Autoridad y reutilización

- capability: `AGT-10`;
- dependencias: `director_lease`, `mailbox`, `controls`, `test_attestor`,
  `command_registry`, `i18n` y `codex_e2e`;
- writer de lifecycle: ninguno en este corte; `Orchestrator` sigue siendo el
  writer único y el scheduler existente conserva creación y claim de
  ejecuciones;
- preflight: `reimplement`; sin función legacy exacta ni permiso de copia;
- excepción bootstrap: exactamente tres subagentes de solo lectura auditaron
  receipts/sujeto, transplantes/duplicados y presupuesto/reducción de tests;
  solo el padre editó.

## Contrato corregido

`EvaluateProviderPipeline` acepta exactamente la plantilla causal
`author -> primary -> adversarial`. El sujeto liga `ProjectRef`, `GoalRef` y
revisión, generación de plan, `WorkItemRef` y revisión, generación de AppSpec,
`SpecHash` y digests exactos de source, diff y tests.

Cada receipt incluye el digest del pipeline completo, no solo de su etapa. Así,
un receipt consumido queda invalidado si se sustituye una etapa futura. Todos
los receipts ligan ejecución/intento y `LaunchReceiptRef`; primary y
adversarial ligan además `ReviewRecord.Ref`, review subject, assessment digest
y verdict. `LaunchReceiptRef` es único entre las tres etapas y `ReviewRef` es
único entre primary y adversarial; una reutilización se rechaza como replay.

El review subject no es un digest libre ni se valida solo por su forma. La
aplicación lo deriva canónicamente con dominio propio a partir del digest del
pipeline completo, el digest de la etapa author y el receipt canónico de
author. Primary y adversarial deben presentar exactamente ese valor. Se
rechazan el mismo digest arbitrario bien formado, el subject de otro Goal y el
subject derivado para otra etapa, aunque se recanonicen los receipts y la
cadena posterior.

El prefijo se consume en orden exacto, con fallback solo opt-in. El replay
exacto devuelve la misma decisión; se rechazan receipts transplantados entre
Goal o etapas, launch/review refs duplicados, receipts mutados, saltos,
repeticiones, futuros stages divergentes, roles duplicados/invertidos y cadenas
incompletas. `Complete` significa únicamente «prefijo causal completo»: no
equivale a review aprobada, efecto, integración ni cierre de Goal.

Los leases vivos no se incrustan en receipts históricos. Esta evaluación no
autoriza ni reclama el siguiente lanzamiento y no duplica scheduler.

## Verificación de esta revisión

Verde sobre el package de application y el acceptance AGT-10, ambos normal y
`-race`; también vet focal, compilación integral del árbol visible y build:

```text
go test -mod=vendor -count=1 ./internal/application -run '^TestProviderPipeline'
go test -mod=vendor -count=1 ./acceptance -run '^TestAcceptanceV25AGT10ProviderPipeline$'
go test -mod=vendor -race -count=1 ./internal/application -run '^TestProviderPipeline'
go test -mod=vendor -race -count=1 ./acceptance -run '^TestAcceptanceV25AGT10ProviderPipeline$'
GOFLAGS=-mod=vendor go vet ./internal/application
go test -mod=vendor -run '^$' ./...
go build -mod=vendor ./...
```

El gate `go test -mod=vendor -count=1 . -run '^TestRebuildArchitecture$'`
queda rojo por WIP ajeno: los adapters `ollama` y `openaicompat` importan el
adapter hermano `internal/adapters/agent/localhttp`. AGT-10 no aparece en el
fallo y este write-set no invade esos carriles. No se ejecutó ni se atribuye un
E2E real de provider, Firecracker o producción.

Presupuesto autónomo medido íntegramente contra `HEAD` —los tres ficheros Go
son nuevos—: `P=342` LOC de producción (`<=450`) y `V=464` LOC de verificación
(`271` unitarias + `193` acceptance, `<=500`). No se movieron helpers fuera del
write-set ni se abrió ADR. `gofmt -d`, `git diff --check` y el diff-check
`--no-index` de cada fichero nuevo no emitieron diagnósticos.

## Cierre honesto

- hecho: cadena canónica, review subject exacto, refs launch/review únicos y
  negativos de duplicación/transplante causal;
- invariante: una plantilla de providers no es scheduler, lifecycle ni review
  gate;
- autoridad final: Goal/application/scheduler existentes;
- receipts y revisión acreditada: ninguno de release; los refs recibidos son
  validados como contrato, no consultados aún en el store;
- retirada: ninguna; legacy no retirable sin equivalencia acreditada;
- tamaño autónomo contra `HEAD`: `P=342`, `V=464`, dentro del presupuesto del
  carril;
- P0: ninguno conocido;
- P1: el wiring posterior debe cargar `ExecutionRecord`/`ReviewRecord` del
  `GoalRecord`, aplicar CAS/replay histórico y dejar que el scheduler único cree
  y reclame la etapa siguiente;
- siguiente causal: composición durable y adapters reales aislados sobre el
  mismo candidato, sin promover roadmap/evidence desde este corte.
