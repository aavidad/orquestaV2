# orquesta-runtime-codex-delivery

Adaptador exterior para leer receipts Codex ya materializados y convertirlos en
`AgentDeliveryObservationV0` del nucleo.

Flujo de registro:

```text
ExternalProcessAgentLauncherV0
  -> CodexReceiptRecordingSpecResolverV0
  -> inner ExternalAgentLaunchSpecResolverV0
  -> CodexReceiptDescriptorRecorderPortV0
```

Flujo de observacion:

```text
store externo -> ack path + ExternalAgentLaunchSpecV0
  -> orquesta-runtime-codex.ReadCodexDeliveryObservationFileV0
  -> AgentDeliveryObservationProviderPortV0
  -> DeliveryCandidateProviderV0 -> RegisterDelivery
```

Flujo de supervision de progreso:

```text
store externo + registry run/agente -> process_ref + session_ref
  -> senales compactas de runtime sin ACK listo
  -> AgentProgressObservationProviderPortV0
  -> ProgressSupervisionCandidateProviderV0 -> AssessAgentWork / AskDirector
```

Flujo de revision:

```text
delivery registrada + descriptor ACK
  -> CodexReviewGateObservationSourceV0
  -> evidencia real de ficheros por puerto externo
  -> ReviewGateObservationProviderPortV0
  -> ReviewGateCandidateProviderV0 -> RequestReview / RecordReviewResult / AcceptReview
```

El modulo no pertenece al nucleo. La ruta del ACK, el spec y cualquier registro
durable los aporta un puerto externo. La observacion devuelta al nucleo solo
contiene refs compactas: no paths, logs, stdout/stderr, HOME, OAuth, modelo,
provider, DB ni transcripts.

Adaptadores locales:

- `CodexReceiptRecordingSpecResolverV0`: decorador de spec resolver; registra el
  descriptor del ACK cuando el launcher prepara el proceso.
- `InMemoryCodexReceiptDescriptorStoreV0`: store de tests/dry-run; el store
  productivo debe ser un conector externo.
- `FileCodexReceiptDescriptorStoreV0`: conector filesystem explicito para
  recuperar descriptors tras reinicio; no es default ni selecciona DB.
- `CodexReceiptWorktreeBaselineRecorderV0`: captura baseline del proyecto antes
  del launch mediante `orquesta-runtime-worktree`, si el operador inyecta store.
- `CodexReceiptWorktreeVerifierV0`: antes de aceptar un ACK como entrega,
  compara el diff real contra el write-set; cambios fuera de contrato quedan
  como rail blando, y borrados/requests invalidas siguen bloqueando.
- `CodexProgressObservationSourceV0`: observa agentes arrancados que aun no han
  escrito ACK. Usa `AgentProcessRegistryPortV0` para resolver identidad
  `process_ref + session_ref` y `CodexProgressStateStorePortV0` para evitar
  repetir el mismo aviso de estancamiento.
- `CodexReviewGateObservationSourceV0`: revisa deliveries ya registradas usando
  ACK, tests requeridos, write-set y evidencia real de ficheros. Acepta o pide
  cambios sin filtrar rutas al nucleo.
- `CodexReviewGateProjectFileEvidenceV0`: adaptador exterior que cuenta lineas
  reales y marca ficheros ausentes/no legibles o destinos del write-set no
  materializados con incidencias compactas para review/rework.
- `FileCodexProgressStateStoreV0`: conector filesystem explicito para conservar
  heartbeats compactos/reportes anti-bucle entre reinicios.
- `StaticCodexReceiptAckPathResolverV0`: resolver de path para tests; en
  produccion el path debe salir de configuracion/registro externo del conector.

Validacion local:

```bash
go test -count=1 ./modulos/orquesta-runtime-codex-delivery
git diff --check -- modulos/orquesta-runtime-codex-delivery
```

Smoke real opt-in con Codex:

```bash
ORQUESTA_CODEX_SMOKE=1 \
ORQUESTA_CODEX_COMMAND="$(command -v codex)" \
ORQUESTA_CODEX_HOME="$HOME" \
ORQUESTA_CODEX_CODE_HOME="${CODEX_HOME:-$HOME/.codex}" \
ORQUESTA_CODEX_PATH="$PATH" \
ORQUESTA_CODEX_APPROVAL_POLICY=never \
ORQUESTA_CODEX_SANDBOX=workspace-write \
ORQUESTA_CODEX_SMOKE_TIMEOUT_SECONDS=120 \
go test ./modulos/orquesta-runtime-codex-delivery \
  -run TestCodexReceiptDeliveryLoopV0SmokeCodexRealOptIn \
  -count=1 -timeout 150s
```

Este smoke queda desactivado por defecto para no gastar cuota real. Al activarlo,
Orquesta lanza un agente Codex real, espera un `agent_ack.json` valido y ejecuta
otro tick para registrar la entrega en el nucleo.
Con `workspace-write`, si se fija `ORQUESTA_CODEX_RUNTIME_WORKDIR`, debe apuntar
a un directorio oculto dentro de `ORQUESTA_CODEX_PROJECT_WORKDIR`, por ejemplo
`$ORQUESTA_CODEX_PROJECT_WORKDIR/.orquesta-codex-runtime`.

Smoke opt-in de mini app Go:

```bash
ORQUESTA_CODEX_APP_SMOKE=1 \
ORQUESTA_CODEX_COMMAND="$(command -v codex)" \
ORQUESTA_CODEX_HOME="$HOME" \
ORQUESTA_CODEX_CODE_HOME="${CODEX_HOME:-$HOME/.codex}" \
ORQUESTA_CODEX_PATH="$PATH" \
ORQUESTA_CODEX_APPROVAL_POLICY=never \
ORQUESTA_CODEX_SANDBOX=workspace-write \
go test ./modulos/orquesta-runtime-codex-delivery \
  -run TestCodexReceiptDeliveryLoopV0SmokeCodexRealAppOptIn \
  -count=1 -timeout 300s
```

Este segundo smoke valida una mini app Go con API REST y web, pero no debe usarse
como patron para apps completas: una app real debe entrar partida en microtareas
y varios agentes con ACK/progreso por corte.

Smoke opt-in de app pequena dirigida por Orquesta:

```bash
ORQUESTA_CODEX_PROGRAMMING_TEAM_SMOKE=1 \
ORQUESTA_CODEX_COMMAND="$(command -v codex)" \
ORQUESTA_CODEX_HOME="$HOME" \
ORQUESTA_CODEX_CODE_HOME="${CODEX_HOME:-$HOME/.codex}" \
ORQUESTA_CODEX_PATH="$PATH" \
ORQUESTA_CODEX_APPROVAL_POLICY=never \
ORQUESTA_CODEX_SANDBOX=workspace-write \
ORQUESTA_CODEX_SMOKE_TIMEOUT_SECONDS=420 \
go test ./modulos/orquesta-runtime-codex-delivery \
  -run TestProgrammingTeamCodexRealOptInV0 \
  -count=1 -timeout 480s -v
```

Duracion esperada: 4-8 minutos segun cuota y latencia local. Este smoke arranca
un director real desde `StartAppDirectorV0`, consume `director_decisions.json`,
lanza agentes de programacion por Orquesta y registra `PhaseArtifactRegistered`
para el director y `DeliveryRegistered` para API, web y documentacion minima.
