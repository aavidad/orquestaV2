# Corte V26 / TLS-11: disposición gobernada de plugins

Fecha: 2026-08-21. Estado: incremento local verificado, sin cambiar estados
canónicos. `TLS-11` permanece `declared` y `AC-V26-TOOLS-SKILLS-SDK`,
`planned`; no hay acreditación de TLS-11 ni de V26.

## Autoridad, carril y presupuesto

- capability: `TLS-11`; consume TLS-10/TLS-12 y no atribuye TLS-13/14;
- invariante: el sujeto es plugin ID/revisión/digest y catálogo curado exactos,
  nunca `latest`, wildcard ni catálogo completo;
- autoridad: `PluginOperationState` es un snapshot in-process inmutable de
  disposición deseada; no escribe Goal ni sustituye application;
- puertos, adaptadores, stores, configuración y procesos: ninguno;
- write-set: `internal/tooling/plugin_operations.go`, su test,
  `acceptance/v26_plugin_operations_test.go` y este documento;
- dependencias: PluginCatalog TLS-10 y CuratedCatalog TLS-12 locales, ambos aún
  sin acreditación;
- presupuesto: 441 LOC productivas, 497 de verificación, documento menor de
  140 líneas, cero dependencias y menos de 200 MiB persistentes.

Quedan fuera autenticación/autorización, persistencia/restart durable, staging,
descarga, firma, filesystem, publicación física, secretos, procesos,
conectores, sesiones en uso, UI/SDK y E2E de composición.

## Preflight y caracterización legacy

Se ejecutó antes de editar:

```text
scripts/preflight_reutilizacion_legacy.sh --capability TLS-11 \
  --path internal/tooling/plugin_operations.go --operation reauditar_corregir \
  --task 'Reauditar y corregir install, upgrade, disable y remove de plugins como disposición inmutable gobernada por catálogo curado exacto, CAS por revisión, tombstone/reinstall fencing, replay idempotente o conflictivo, actor/proyecto/tiempo/razón y receipts ligados a digest/review, sin afirmar autorización, persistencia ni efecto físico' \
  --function 'NewPluginDisposition/ApplyPluginOperation'
```

Resultado advisory: `reimplement`, una pista léxica y ninguna función exacta o
verde histórico. La fuente congelada T261 caracteriza una carrera del refresh
de skills Codex, no este lifecycle. Se conservan revisión esperada, no-upgrade
in-place, receipt y disable-before-remove; se rechazan lock, paths, wrapper y
smoke antiguos. No se copió ni modificó legado.

## Contrato corregido

`PluginOperationRequest` exige request/idempotency refs, actor, proyecto,
capability TLS-11, scope del proyecto, razón, tiempo solicitado, catálogo
curado digest exacto, plugin ID/revisión/digest y expected revision. El futuro
writer aporta `committedAt`; una operación nueva rechaza tiempo cero, request
posterior al commit o commit anterior al último cambio.

Las transiciones son default-deny:

- `install`: solo ausencia/tombstone; queda enabled y no admite downgrade ni
  cambiar el digest de la misma identidad histórica;
- `upgrade`: solo revisión numérica mayor, conserva enabled/disabled y no
  modifica la versión anterior in-place;
- `disable`: solo desde enabled sobre el plugin exacto;
- `remove`: solo desde disabled y conserva tombstone revisionado;
- la reinstalación exige revisión esperada y versión/digest compatibles con el
  tombstone.

Install/upgrade resuelven exclusivamente en TLS-12. Disable/remove permiten
retirar una versión ya deseleccionada, pero exigen capability/scope y un
catálogo del mismo ID que no retroceda ni reescriba su digest en igual versión.

Cada transición produce un sucesor separado e incrementa revisión. Un stale
falla sin mutación. Dos propuestas desde la misma base pueden bifurcar; esto es
una precondición optimista, no CAS atómico ni persistencia. El futuro writer
transaccional debe elegir un sucesor.

El P1 corregido estaba en el replay: antes validaba catálogo y `committedAt`
antes de consultar el receipt. Ahora el fingerprint se calcula primero y se
acota por proyecto+idempotency. Un replay exacto devuelve estado actual y
receipt original sin depender del catálogo histórico ni crear otro commit; una
clave divergente falla por conflicto. Solo una clave nueva valida catálogo,
contexto, tiempo y transición completos.

El receipt liga request digest, before/after, actor, proyecto, razón, tiempos,
plugin, catálogo y review ref/digest. Es un hecho de disposición deseada: no
contiene ni acredita bytes, descarga, filesystem, firma, autorización, intento
físico o efecto aplicado.

## Verificación

Verde en esta revisión local:

```text
go test -mod=vendor -count=1 ./internal/tooling ./acceptance -run '^(TestPluginOperation|TestV26TLS11)'
go test -mod=vendor -race -count=1 ./internal/tooling ./acceptance -run '^(TestPluginOperation|TestV26TLS11)'
go test -mod=vendor -count=1 ./internal/tooling ./acceptance
GOFLAGS=-mod=vendor go vet ./internal/tooling
go test -mod=vendor -count=1 . -run '^TestRebuildArchitecture$'
git diff --check
git diff --no-index --check /dev/null RUTA_TLS11
```

Los negativos cubren catálogo/contexto/scope, stale y bifurcación concurrente,
upgrade in-place, remove prematuro, tiempo inválido, tombstone/downgrade/digest
reescrito/reinstall, replay tras rotación o sin catálogo, conflicto por
actor/razón/tiempo, UTF-8, binding del receipt y ausencia de hechos físicos.

El vet transversal también quedó verde. Los tests transversales conservan
rojos fuera del carril: perfiles Codex no disponibles, toolchain bootstrap,
anchors/receipts globales obsoletos y un script compartido no versionado. No se
alteraron esos owners ni se emitió evidencia para ocultarlos.

## Cierre honesto

- hecho: contrato inmutable install→upgrade→disable→remove y replay histórico;
- invariante restaurado: catálogo registrado nunca instala implícitamente;
- autoridad final: snapshot tooling revisionado, no Goal/application;
- tests/E2E: unitarios, aceptación in-process, concurrencia y race; sin E2E;
- receipts/revisión: sintéticos de disposición; ninguna evidencia de release;
- código/decisión retirados: ninguno;
- legacy retirado: ninguno; no existe función exacta reutilizable;
- complejidad: dos mapas in-memory, cinco bucles finitos, cero loop residente,
  puerto, store durable, goroutine productiva o writer adicional;
- P0/P1 internos: ninguno pendiente tras corregir replay y presupuesto;
- P1 causal fuera del corte: faltan autorización y persistencia atómica en
  application y separación intent/aprobación/intento/receipt del efecto físico;
- siguiente dependencia: writer transaccional application-side y adapter físico
  de staging/publicación/restart; después TLS-14 sobre composición real.
