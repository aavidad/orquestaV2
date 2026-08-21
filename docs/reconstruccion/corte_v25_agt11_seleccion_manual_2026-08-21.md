# Corte V25 AGT-11: selección manual exacta por rol

Fecha: 2026-08-21.

Estado honesto: candidato correctivo autónomo de decisión pura, revisado y
ejercitado en sus gates locales.
`AGT-11` continúa `declared`; V25 y `AC-V25-PROVIDER-ADAPTERS` continúan
`planned`. No persiste preferencias, no lanza agentes, no añade wiring y no
acredita V25.

## Envelope y preflight

El write-set se limitó a la decisión de `application`, sus unitarios, un smoke
de aceptación público y este corte. No añadió puertos, adaptadores, stores,
helpers externos, ADR, configuración ni otro lifecycle.

El preflight obligatorio de AGT-11 devolvió `reimplement`: conserva como
contrato el catálogo observado, un par provider/model realmente disponible,
rol y effort tipados. No existe función legacy exacta ni permiso de copia.

## Autoridad fail-closed

La selección liga principal, proyecto, Goal/revisión, generación de plan,
WorkItem/revisión, generación de AppSpec, `SpecHash` y rol. Reutiliza:

- `AuthorizationReceipt` permitido para `goals.direct` y el Goal exacto;
- rol y revisión esperados, más `Membership` actual, activa y coincidente;
- membresía concedida como máximo al decidir la autorización, no solo antes de
  ejecutar la selección;
- la excepción existente de `platform_admin`, sin membresía y con revisión 0;
- `DirectorLeaseRecord` del Goal y principal exactos, token válido, fence no
  cero y `LeaseUntil > DecidedAt`.

La decisión conserva receipt ref, rol/revisión RBAC y fence. No contiene el
token del lease. UI y provider no escriben routing ni lifecycle.

## Exactitud, freshness y replay

`SelectProviderModelForRole` consulta un único candidato y prohíbe fallback.
Exige catálogo no futuro, provider fresco y disponible, cuota disponible,
modelo, capabilities y effort exactos. Provider/modelo, catálogo, uso y fence
quedan observables en la decisión.

`RequestRef` y el fingerprint ligan contexto, solicitud, catálogo, principal
completo —incluidos kind y método—, reason code de autorización, rol/revisión
esperados, membresía y lease. Los diez `time.Time` externos que participan en
decisión o fingerprint se capturan una vez como `Round(0).UTC()`. El replay
mantiene igualdad estructural exacta sobre esa forma canónica; cualquier cambio
bajo ese prior devuelve
`application.provider_manual_selection_replay_divergent`. Dieciséis lectores
concurrentes reprodujeron la misma decisión desde un catálogo, request, lease y
prior copiados como persistencia canónica bajo `-race`. La aceptación pública
cubre además el catálogo sin probes que originó el P1 monotónico/location.

Los negativos cubren principal/proyecto/Goal/permiso, receipt denegado o
temporalmente imposible, rol/revisión/membresía, grant posterior a la
autorización, lease token/fence/scope/expiry, contexto/generaciones, catálogo
futuro o stale, disponibilidad, cuota, modelo cercano, capability, effort y
replay divergente por hechos de autoridad.

## Verificación y bloqueos del WIP

Verdes sobre el candidato: unitarios y aceptación focal normales/`-race`, suite
completa de `./internal/application`, vet focal y global, `gofmt`,
`TestRebuildArchitecture`, compilación de todos los paquetes visibles y
`go build -mod=vendor ./...`.

Los mínimos globales no permiten atribuir verde al worktree recibido:

- `go test -mod=vendor -count=1 .` falla por anclas/hash de `ports.go`, receipts
  Codex obsoletos y el script ajeno no seguido
  `scripts/continuar_orquestav2_home_codex.sh`;
- `go test -mod=vendor -count=1 ./internal/... ./cmd/orquesta` conserva verde
  `internal/application` y `cmd/orquesta`, pero falla fuera del write-set por
  `codex.account_profile_unavailable` y
  `bootstrap.codex_go_toolchain_invalid`.

## Presupuesto íntegro contra HEAD

Los tres ficheros Go no existen en `HEAD`; por tanto todas sus líneas físicas
cuentan como añadidas: `P=425`; `V=500` (`399` unitarias + `101` acceptance).
Cumple `P<=450` y `V<=500` mediante reemplazo y compactación de líneas del
candidato recibido; el documento queda en `doc=98`. No hay ADR.

## Cierre honesto

- hecho: decisión pura exacta, causal, concurrente y con replay canónico;
- invariante restaurado: metadata monotónica/location no altera decisión,
  fingerprint ni replay; se conservan membership, RBAC y lease fail-closed;
- autoridad final: Goal, RBAC y lease del Director existentes;
- receipts/revisión acreditada: no se emite receipt de release;
- código o decisión retirados: ninguno; ningún legacy retirable;
- P0 AGT-11: ninguno; P1 AGT-11: replay temporal reproducido y cerrado por la
  regresión canónica. No se afirma ausencia de P1 en el resto del worktree;
- límite: los facts siguen siendo aportados por el caller y no acreditan su
  procedencia durable sin el caso de uso del writer;
- siguiente causal: wiring read-only y persistencia idempotente por CAS en el
  writer/store existentes, seguidos de composición y E2E real V25.
