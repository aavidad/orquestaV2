# Handoff V22 — cierre ordenado de sesión

Fecha: 2026-07-23. Estado: **checkpoint WIP seguro; V22 no está cerrado,
sellado ni acreditado**.

## Reanudación exacta

- Worktree:
  `/home/alberto/Trabajo/orquesta-rebuild-worktrees/v22-codex-e2e-integrated`
- Rama: `reconstruccion/v22-codex-e2e-integrated`
- Base V21 acreditada:
  `665ef446a32a7f0512255640fa99b2e7edd9f29d`
- El commit que contiene este documento es un checkpoint de continuidad, no el
  commit P del protocolo P/S/E.
- Repo histórico `/home/alberto/Trabajo/orquesta`: no tocar ni mezclar con este
  cierre.

El siguiente agente debe leer, en este orden:

1. `AGENTS.md`;
2. `docs/reconstruccion/analisis_y_contrato_v22_codex_e2e.md`;
3. `docs/reconstruccion/manual_revision_claude_v22.md`;
4. este handoff;
5. `acceptance/fixtures/v22_codex_e2e.json`;
6. `docs/inventario_bugs_orquesta_2026-06-30.md`, filas 390-396.

Auditoría externa útil, pero no acreditación:
`/home/alberto/Trabajo/orquesta-rebuild-worktrees/analisis_forense_orquesta_v1_v22_2026-07-23.md`.

## Implementado

- Prompt Codex propiedad del catálogo i18n ES/EN; renderer tipado.
- Autoridad material-free ligada al tuple durable de ejecución.
- Un único `CredentialStore` compartido por runtime Codex y broker efímero.
- Bearer MCP solo hacia endpoint loopback exacto, sin query, redirect ni
  persistencia en prompt/argv/Goal/journal/receipt.
- Principal `execution_service` sin membership humana ni acceso principal-wide.
- Lectura de artefacto limitada al envelope entregado al principal/ejecución
  exactos.
- Continuación post-artefacto por la outbox existente y llamada MCP pública
  `orquesta.mailbox.admit`, sin scheduler ni lifecycle paralelo.
- Claim/delivery/consume/ACK del destinatario validados por receipt
  execution-bound, envelope, ejecución y fence exactos; restart y negativos.
- Proyección pública de Goals, DAG, ejecuciones, artefactos, attestations,
  reviews, controles, integración y mailbox.
- Fuente E2E real opt-in con binario externo, cuatro Goals A/B/C/D, MCP, stop B,
  backup/verify/restore V09, SIGKILL/restart y censo de procesos.
- Allowlist V22 exacta en `scripts/check_rebuild_write_set.sh`.

## Verificación ya ejecutada

Verde:

```bash
scripts/check_rebuild_write_set.sh 665ef446a32a7f0512255640fa99b2e7edd9f29d
git diff --check
go test -mod=vendor -tags v22_real_e2e -run '^$' ./acceptance
go test -mod=vendor -count=1 ./acceptance \
  -run '^(TestV22SimplicityBudgetMeasuresPhysicalDelta|TestV22PreflightContractIsStructurallyValid|TestAcceptanceV22CodexE2E)$'
go test -mod=vendor -count=1 . \
  -run '^(TestProductRoadmapV22ScopeAndLifecycleContract|TestV22EvidenceBelongsOnlyToCodexE2ECapabilities|TestV22AcceptanceCommandCannotPassWithoutOwnedRealTests)$'
go test -mod=vendor -count=1 ./internal/adapters/state/sqlite \
  -run '^(TestCodexChildDeliveryUsesSameExecutionServicePrincipalAfterArtifactPersistence|TestExecutionServicePrincipalExactScopeRevocationAndRestart|TestMailboxRestartPreservesEveryCausalFrontier|TestMailboxHandoffRequirementSurvivesRestartAndGatesAdmission)$'
go test -mod=vendor -race -count=1 -timeout=270s \
  ./internal/adapters/agent/codex ./internal/adapters/state/sqlite ./internal/bootstrap \
  -run '^(TestCodexChildDeliveryUsesSameExecutionServicePrincipalAfterArtifactPersistence|TestCodexRuntimeUsesCatalogOwnedPrompt|TestExecutionServicePrincipalExactScopeRevocationAndRestart|TestProductionBuildCreatesDurableExecutionAuthorityResolver)$'
```

También pasó SQLite completo en una revisión inmediatamente anterior; después
se endureció la rederivación de receipts y se repitieron focales y `-race`.
Por eso el siguiente agente debe volver a ejecutar el gate completo, no asumir
que ese full anterior acredita los bytes finales.

No ejecutado:

- E2E Codex real A/B/C/D;
- gate literal completo del fixture sobre los bytes finales;
- P, S o E;
- receipt/output V22;
- promoción de roadmap.

## Deuda aceptada por el operador

- `BUG-ORQ-20260723-395`: objetivo inicial 2.800/3.800 LOC superado; techo
  explícito 3.000/4.200. No recortar ficheros ahora si funcionan y conservan
  legibilidad/cobertura.
- `BUG-ORQ-20260723-396`: revalidación SQLite de receipts execution-bound
  O(R×E). No falló en alcance local V22. Antes de V31/Postgres/multihost debe
  persistirse el `execution_ref` exacto en el receipt existente y validar con
  JOIN O(1), sin store nuevo.

No elevar de nuevo los límites ni declarar cumplido el objetivo original.

## Siguiente secuencia obligatoria

1. Confirmar worktree limpio respecto al checkpoint y cero procesos de prueba.
2. Ejecutar el gate normal completo sin el E2E:

   ```bash
   go test -mod=vendor -count=1 . ./acceptance ./internal/... ./cmd/... ./sdk/...
   ```

3. Ejecutar los dos E2E reales con el tag y conservar salida JSON:

   ```bash
   go test -mod=vendor -tags=v22_real_e2e -json -count=1 -timeout=900s \
     ./acceptance \
     -run '^(TestV22RealCodexFourGoalsSelectiveStopCrashRestartAndCloseThroughMCP|TestV22RealCodexNoTerminalContradictionAndNoOwnedProcess)$'
   ```

4. Si falla, corregir solo bloqueo directo del E2E; no abrir V23, proveedores
   alternativos, web, wizard ni autoprogramación.
5. Repetir el argv literal de `acceptance/fixtures/v22_codex_e2e.json`.
6. Solo con todo verde: crear P, después S y ejecutar E desde detached-clean.
   No reutilizar este checkpoint como P ni escribir un OID inventado.
7. Generar receipt/output, promover únicamente `AC-V22-CODEX-E2E` y sus cuatro
   capabilities, actualizar 390/391/394 con evidencia y cerrar sesión.

## Estado operativo al cortar

- Cero `orquesta serve`, `orquesta-server run`, E2E V22 o
  `codebase-memory-mcp` vivos.
- 63 GiB libres en `/home`; caches temporales de esta sesión están bajo
  `/tmp/orquesta-v22-root-gocache` y `/tmp/orquesta-v22-root-gotmp`.
- No existe receipt V22 ni evidencia que pueda confundirse con PASS.
