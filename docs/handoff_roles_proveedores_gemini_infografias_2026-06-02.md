# Handoff: roles de proveedores y Gemini para infografias - 2026-06-02

## Estado

Primer corte implementado y probado offline. Hay adaptador Gemini CLI opt-in y
router de proveedor en la composicion actual. No hay smoke real con Gemini CLI
ni credenciales: queda como prueba opt-in de operador.

Correccion arquitectonica aceptada: OPES no debe saber nada de Gemini, Codex,
Claude ni agentes. OPES debe conservar solo contratos de dominio como
`generate_visual_asset`, `visual_asset`, refs opacas, validacion y ensamblado.
Orquesta debe decidir que agente/proveedor ejecuta cada rol.

## Decision de frontera

- No meter Gemini en OPES.
- No meter Codex/Gemini/Claude en `orquesta-core-workflow`,
  `orquesta-domain-work` ni Director puro.
- La seleccion `Codex programa`, `Gemini crea infografias`,
  `Claude documenta` pertenece a composicion/adaptadores de Orquesta.
- Mantener Codex como default legacy del stack actual.
- Habilitar Gemini solo opt-in y por rol/capacidad/configuracion explicita.

## Contexto revisado

Fuentes vigentes leidas:

- `AGENTS.md`
- `docs/estado_actual_2026-05-17.md`
- `docs/guia_nucleo_orquestacion_2026-05-17.md`
- `docs/principio_orquesta_piensa_director.md`
- `docs/corte_cierre_generico_director_operativo_2026-05-17.md`
- `docs/matriz_pruebas_reales_y_smoke_2026-05-17.md`
- `modulos/orquesta-runtime/AGENTS.md`
- `modulos/orquesta-runtime-codex/AGENTS.md`
- `modulos/orquesta-runtime-codex-delivery/AGENTS.md`
- `modulos/orquesta-app-codex-stack/AGENTS.md`
- `cmd/orquesta-server/AGENTS.md`

Tambien se comprobo OPES solo para descartar la ruta equivocada. OPES tiene
generador visual local/reglas y ComfyUI, pero no debe recibir el conector
Gemini como conocimiento propio.

## Piezas relevantes

- `modulos/orquesta-core-workflow/work_profile_v0.go`: perfiles neutrales
  actuales: `code_study`, `implementation`, `refactor`, `required_tests`,
  `documentation`, `review`, `domain_work`.
- `modulos/orquesta-orchestration-core/workflow_task_profile_resolver_v0.go`:
  resuelve perfil neutral a rol/capacidad, no proveedor.
- `modulos/orquesta-runtime/runtime_launch_request_types_v0.go`:
  `RuntimeBindingV0` ya transporta refs opacas de runtime/proveedor/modelo/home.
- `modulos/orquesta-runtime/external_agent_connector_types_v0.go`:
  contrato neutral de spec de agente externo.
- `modulos/orquesta-runtime-codex`: patron de adaptador concreto opt-in.
- `modulos/orquesta-runtime-gemini`: adaptador Gemini CLI opt-in, control files,
  wrapper headless y prompt compatible con ACK durable.
- `modulos/orquesta-runtime-codex-delivery`: receipts/ACK/observacion de
  entregas. Gemini reutiliza el recibo `codex_agent_ack.v0` como contrato de
  observacion existente.
- `modulos/orquesta-app-codex-stack/gemini_provider_resolver_v0.go`: router de
  proveedor. Codex sigue por defecto; Gemini solo entra si esta habilitado y el
  trabajo externo resuelve a `visual_asset`.
- `modulos/orquesta-app-codex-stack/dispatchers_v0.go`: `agentBatchDispatcherV0`
  usa `recordingSpecResolverV0(config)` con ack path consciente de proveedor.
- `cmd/orquesta-server/codex_runtime_config_v0.go` y
  `cmd/orquesta-server/gemini_runtime_config_v0.go`: superficies env actuales
  Codex/Gemini.

## Implementado

- `modulos/orquesta-runtime-gemini`:
  perfil `GeminiConnectorProfileV0`, validacion, wrapper `gemini --prompt ''`
  con prompt por stdin, ficheros `agent_packet.json`, `agent_prompt.txt`,
  `agent_ack.json`, logs stdout/stderr y prompt visual accesible.
- `modulos/orquesta-app-codex-stack`:
  `providerLaunchSpecResolverV0` mantiene Codex como fallback y rutea Gemini
  por senal neutral:
  `domainWorkArtifactTypeForWorkKindV0(work.WorkKind) == visual_asset`.
  No depende de OPES ni de strings exactos del usuario final; aprovecha los
  aliases ya aceptados por la entrega de dominio.
- `cmd/orquesta-server`:
  `ORQUESTA_GEMINI_ENABLED`, `ORQUESTA_GEMINI_COMMAND`,
  `ORQUESTA_GEMINI_PROJECT_WORKDIR`, `ORQUESTA_GEMINI_RUNTIME_WORKDIR`,
  `ORQUESTA_GEMINI_HOME`, `ORQUESTA_GEMINI_PATH`,
  `ORQUESTA_GEMINI_MODEL`, `ORQUESTA_GEMINI_APPROVAL_MODE`,
  `ORQUESTA_GEMINI_OUTPUT_FORMAT` y `ORQUESTA_GEMINI_EXTRA_ARGS`.
  `ORQUESTA_GEMINI_APPROVAL_MODE` cae a `auto_edit` solo cuando Gemini esta
  habilitado.

## Pruebas ejecutadas

- `go test -count=1 ./modulos/orquesta-runtime-gemini ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`

Pendiente de cierre transversal tras este documento: `git diff --check` y
`go test -count=1 ./...`.

## Nota operativa

La documentacion oficial revisada de Gemini CLI indica modo no interactivo con
`gemini -p/--prompt`, seleccion de modelo con `--model/-m`, salida con
`--output-format` y modo de aprobacion con `--approval-mode`. Es una base viable
para un adaptador CLI headless. El wrapper usa `--prompt ''` y pasa el prompt
por stdin para no poner el prompt completo en argv. Smoke real recomendado solo
con instancia controlada, credenciales opt-in y trabajo `visual_asset` acotado.
