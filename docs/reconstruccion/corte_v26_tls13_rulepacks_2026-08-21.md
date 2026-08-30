# Corte V26 TLS-13: rule/workflow packs versionados

Fecha: 2026-08-21.

## Alcance y autoridad

Capability `TLS-13`; `AC-V26-TOOLS-SKILLS-SDK` continúa `planned` y TLS-13
continúa `declared`. No se modifican roadmap, capabilities ni evidence y no se
atribuye implementación, wiring, ejercicio o acreditación de V26.

Write-set exclusivo:

- `internal/tooling/rulepacks.go`;
- `internal/tooling/rulepacks_test.go`;
- `acceptance/v26_rulepacks_test.go`;
- este documento.

El roadmap fija las dependencias `credentials`, `identity_projects_rbac`,
`budgets_effects` y `command_registry`. `ruta_total_100.md` expresa V21 en vez
de la dependencia directa V20 del roadmap; el corte no resuelve ni aprovecha
esa discrepancia y aplica la lectura más estricta para cualquier gate futuro.

`RulePackCatalog` solo gobierna descriptores. No añade DB, store, writer,
watcher, loader, instalación, ejecución, autorización o lifecycle. Application
conserva la política y `Goal` sigue siendo el único lifecycle.

## Preflight y caracterización

Antes de editar se ejecutó:

```text
scripts/preflight_reutilizacion_legacy.sh \
  --capability TLS-13 \
  --path internal/tooling/rulepacks.go \
  --operation reauditar-corregir \
  --task 'Reauditar revocación, identidad, historia, firmas, trust y restore puro sin sobreafirmar CAS o durabilidad' \
  --function 'NewRulePackCatalog|RefreshRulePackCatalog|RevokeRulePack|Resolve'
```

Resultado advisory: `reimplement`, sin función exacta ni permiso de copia. Se
abrieron en la copia legacy las pistas GOV-004/GOV-005 del mismo documento. Se
conservaron scope explícito, versión/hash, cuarentena y ausencia de activación
automática; se descartaron catálogo documental como runtime, tags como permiso,
refresh parcial y workflow como lifecycle.

## Contrato corregido

- Cada pack fija ID, versión entera canónica, clave i18n, scopes, precedencia e
  items `rule|workflow` referidos por digest de artefacto; no contiene cuerpos.
- La firma Ed25519 de publicación liga ID, versión, digest, `Origin.Ref`,
  `RevisionDigest` y `SignerRef`. La revisión no puede trasplantarse.
- El mismo pack ID conserva `SignerRef` y digest de trust root entre versiones.
  No existe rotación implícita por parámetros del llamador.
- La revocación requiere otra autoridad explícita, exacta por
  `OriginRef+SignerRef+clave`, y firma target, revisión, ref y razón.
- `Refresh` exige snapshot completo, digest/origen previos exactos, mismo ref de
  origen y revisión no observada. Produce una transición pura e inmutable.
- Omitir un `ID+version` conserva tombstone. Reintroducir esa identidad queda
  prohibido incluso con contenido, firmante y clave idénticos.
- Una identidad revocada debe seguir revocada; no se puede retirar, reescribir
  o reactivar su hecho histórico.
- `Snapshot` devuelve copias completas de packs, orígenes e historia.
  `RestoreRulePackCatalog` solo reconstruye ese snapshot si coincide con un
  digest esperado externo y si publicaciones, revisiones, trust y revocaciones
  históricas vuelven a verificarse. Un current pack no puede contradecir su
  tombstone, aunque el atacante recalcule el digest interno.
- `History`, `List`, `Snapshot` y `Resolve` son defensivos frente a aliasing.
- `Resolve` filtra scope/revocación y proyecta precedencia. No autoriza, carga,
  instala ni ejecuta reglas o workflows.

`NewRulePackCatalog` crea deliberadamente una raíz nueva y no es restore. El
restore tampoco lee o escribe disco: el llamador aún debe obtener de una futura
autoridad durable el snapshot completo y su digest esperado. Si entrega una
raíz nueva en vez de restaurar, este contrato puro no puede descubrir historia
omitida. No se afirma persistencia tras reinicio.

`Refresh` y `RestoreRulePackCatalog` no publican, serializan entre procesos,
persisten ni realizan compare-and-swap. Dos ramas puras sobre el mismo receptor
pueden ser válidas; un futuro writer de application deberá elegir y persistir
una con revisión esperada e idempotencia.

## Verificación

Matriz del cierre:

```text
go test -mod=vendor -count=1 ./internal/tooling -run '^TestRulePack'
go test -mod=vendor -race -count=1 ./internal/tooling -run '^TestRulePack'
go test -mod=vendor -count=1 acceptance/v26_rulepacks_test.go
go test -mod=vendor -race -count=1 acceptance/v26_rulepacks_test.go
GOFLAGS=-mod=vendor go vet ./internal/tooling
go test -mod=vendor -count=1 . -run '^TestRebuildArchitecture$'
go test -mod=vendor -count=1 .
go test -mod=vendor -count=1 ./internal/... ./cmd/orquesta
GOFLAGS=-mod=vendor go vet ./internal/... ./cmd/orquesta
git diff --check
```

Como los cuatro archivos están sin seguimiento, `git diff --check` se
complementa con `git diff --no-index --check /dev/null RUTA` y salida vacía.
Los negativos cubren restore con resurrección, digest esperado alterado,
omisión/reintroducción exacta, sustitución de signer/clave entre versiones,
revocador general o de otro origen, firmas/revisiones alteradas, historia
defensiva, replay, downgrade, precedencia ambigua y UTF-8 inválido.

Pasaron focales normales y `-race`, aceptación aislada, vet focal/transversal,
arquitectura, compilación de todos los paquetes y diff-check. El primer race
coincidió con un renombrado ajeno de un test de plugins; la repetición estable
pasó. La suite raíz quedó roja por
hashes ajenos de `internal/application/ports.go`, receipts Codex obsoletos y
guardas de trazabilidad sobre un script ajeno sin seguimiento. `internal/cmd`
dejó `internal/tooling` verde y falló solo por locks Codex compartidos y el
toolchain privado de bootstrap. No se corrigieron fuera del write-set.

## Cierre honesto

```text
hecho: snapshot/restore puros, historia monotónica y resolución scoped de packs
invariante restaurado: identidad omitida no vuelve; revocación restaurada no resucita; trust no rota implícitamente
autoridad final: RulePackCatalog proyecta; application autoriza y Goal conserva lifecycle
puertos/adaptadores afectados: ninguno
tests/negativos/mutaciones/E2E: unitarios, aceptación, race, vet y arquitectura; sin E2E de carga/ejecución
receipts y revisión acreditada: ninguno; TLS-13 declared y AC-V26 planned
código o decisión retirados: ninguno
legacy retirado o bloqueo: solo trazabilidad; sin censo que autorice retirada
riesgos/P0/P1: P0 ninguno; P1 faltan writer/store durable, CAS real, wiring y gate V26
siguiente causal: reconciliar dependencia V20/V21 y componer TLS-11..14 desde application sin promover V26
```
