# Contratos: orquesta-runtime-codex-delivery

## CodexReceiptDescriptorStorePortV0

Puerto externo que lista ACKs Codex candidatos para un run.

Entrada:

- `run_id`
- `started_agents`
- `deliveries`
- `phase_artifacts`
- `correlation_id`
- `evidence_refs`

Salida:

- `descriptor_ref`
- `spec`
- `ack_path`
- `project_work_dir`
- `worktree_baseline_ref`

Invariantes:

- `ack_path` nunca sale hacia el nucleo.
- `project_work_dir` y `worktree_baseline_ref` son detalles del conector; solo
  los consume el adaptador de delivery/worktree.
- `deliveries` y `phase_artifacts` permiten omitir ACKs ya consumidos antes de
  leerlos de nuevo.
- El store puede usar DB, filesystem, broker o memoria, pero siempre como
  conector externo.
- El spec debe estar correlado con el ACK y no debe contener secretos.

## FileCodexReceiptDescriptorStoreV0

Conector filesystem explicito para `CodexReceiptDescriptorStorePortV0` y
`CodexReceiptDescriptorRecorderPortV0`.

Invariantes:

- Requiere directorio absoluto configurado por operador.
- Persiste snapshot JSON con schema versionado y reemplaza por `descriptor_ref`.
- Al recrear la instancia recupera descriptors pendientes para que Orquesta
  pueda observar ACKs tras reinicio.
- No es conector por defecto y no selecciona motor de DB.
- Los errores publicos son compactos y no incluyen `ack_path`,
  `project_work_dir` ni path del fichero de store.

## CodexReceiptDescriptorRecorderPortV0

Puerto externo que registra un descriptor cuando se prepara un lanzamiento.

Entrada:

- `descriptor_ref`
- `run_id`
- `agent_ref`
- `spec`
- `ack_path`

Invariantes:

- `ack_path` es detalle operacional del conector y solo se usa dentro de este
  adaptador.
- Si el registro falla, el lanzamiento se bloquea antes de registrar
  `AgentStarted`; asi evitamos agentes arrancados que luego no puedan entregar
  receipt.

## CodexReceiptRecordingSpecResolverV0

Decorador de `ExternalAgentLaunchSpecResolverPortV0`.

Invariantes:

- No modifica el spec ni el resolver de comando.
- Registra el descriptor del ACK por puerto externo despues de resolver el spec.
- El path del ACK se resuelve por `CodexReceiptAckPathResolverPortV0`, no se
  inventa dentro del core.
- Si falta inner resolver, recorder o path resolver, falla explicitamente.
- Si se inyecta `CodexReceiptWorktreeBaselineRecorderPortV0`, captura baseline
  antes del launch y registra `worktree_baseline_ref` junto al descriptor.
- Si el snapshot de baseline supera presupuesto de lectura, el recorder guarda
  un baseline parcial con recibos de exclusion en vez de dejar al agente sin
  baseline, siempre sin exponer `project_work_dir`.

## CodexDeliveryObservationSourceV0

Implementa `orquestacionnucleoapp.AgentDeliveryObservationProviderPortV0`.

Invariantes:

- Lee ACKs solo a traves de descriptors del store.
- Usa `orquesta-runtime-codex` para validar el ACK y crear una observacion
  neutral.
- Omite deliveries ya registradas en el run.
- Omite artefactos de fase ya registrados en el run, aunque el store externo no
  los haya filtrado.
- Devuelve al nucleo solo `artifact_ref`, `delivery_ref`, `phase_id`,
  `task_id`, `agent_ref`, `summary` y `evidence_refs` compactas.
- `artifact_ref` usa el ACK validado como ref estable para fases que no son
  `programacion`; `delivery_ref` se mantiene para entregas de programacion.
- Si se inyecta `CodexReceiptWorktreeVerifierPortV0`, verifica el diff real del
  proyecto: cambios fuera del write-set y borrados se proyectan como evidencia
  de revision/rework y no tiran la entrega; requests invalidas o errores de
  filesystem siguen bloqueando.
- No filtra paths, logs, transcripts, provider, modelo, HOME, OAuth ni DB.

## CodexReceiptWorktreeVerifierV0

Adaptador externo que usa `orquesta-runtime-worktree` para cargar un baseline y
verificar cambios reales antes de aceptar un ACK.

Invariantes:

- El baseline se captura antes de arrancar el agente.
- El verificador compara filesystem real contra `spec.agent_packet.task.write_set`.
- Los prefijos de control, por ejemplo runtime aislado dentro del proyecto, se
  inyectan como `ignore_prefixes`.
- Si hay cambios fuera de write-set o borrados, devuelve evidencia compacta
  `gate-issue:file_outside_write_set` o `gate-issue:removed_path` sin filtrar
  `project_work_dir`, `ack_path` ni rutas absolutas al nucleo. Requests
  invalidas siguen devolviendo error compacto.
- Los limites de presupuesto de snapshot son observabilidad: si el snapshot
  verificable es parcial, se devuelven refs de evidencia compactas y no se
  fabrican borrados por paths omitidos.
- No usa Git, DB, HOME, OAuth, provider ni modelo.

## CodexProgressObservationSourceV0

Implementa `orquestacionnucleoapp.AgentProgressObservationProviderPortV0`.

Entrada:

- `OrchestrationRunV0` con `run_id`, agentes arrancados, deliveries y
  artefactos ya registrados.
- `CodexReceiptDescriptorStorePortV0` para localizar descriptors del runtime.
- `AgentProcessRegistryPortV0` para resolver `run_id + agent_request_id` a
  `process_ref + session_ref`.
- `CodexProgressStateStorePortV0` para guardar heartbeat compacto por agente.

Salida:

- `AgentProgressObservationV0` solo cuando el agente no tiene ACK listo y no hay
  avance observable durante el umbral configurado.
- `AgentProgressReportV0` neutral, sin rutas ni logs.

Invariantes:

- Si el ACK ya esta listo, no produce supervision de progreso: la prioridad pasa
  a `CodexDeliveryObservationSourceV0`.
- Si el ACK aun no existe, `delivery_ref` queda vacio: la evaluacion pertenece
  al agente/tarea, no a una entrega inexistente.
- Si no existe registro de proceso, falla; no se inventan procesos ni se paran
  agentes sin identidad `process_ref + session_ref`.
- Las senales locales de progreso se reducen a firma compacta interna. El
  nucleo solo recibe refs opacas y el estado `stalled`, `loop_detected` o
  `stopped`.
- Una misma firma puede emitir primero `stalled` y despues `loop_detected` si
  sigue repitiendose; no repite indefinidamente el mismo aviso.
- No escanea procesos por nombre y no usa PID, HOME, OAuth, modelo, provider ni
  DB hardcodeada.

## CodexReviewGateObservationSourceV0

Implementa `orquestacionnucleoapp.ReviewGateObservationProviderPortV0`.

Entrada:

- run en fase `revision`;
- descriptors ACK del store externo;
- ACK validado por `orquesta-runtime-codex`;
- evidencias de ficheros aportadas por un puerto externo.

Salida:

- `ReviewGateObservationV0` con `review_request_id`, `review_result_ref`,
  `delivery_ref`, `status`, `summary`, `quality_gate_ref` y `evidence_refs`.

Invariantes:

- lista descriptors sin filtrar por deliveries ya registradas, porque revision
  solo tiene sentido sobre entregas ya reflejadas en el run;
- despues filtra por `run.Deliveries`, agente elegible y review no proyectado;
- usa `orquesta-autoprogramming.EvaluateAutoprogrammingReviewGateV0` para
  validar ACK completado, tests requeridos, write-set y limite de lineas;
- proyecta `ack-pending-rail:*` del ACK como issues advisory del gate, para que
  queden como follow-up no bloqueante y no como razon de relanzar otro agente;
- una entrega valida produce `accepted` y `accepted_review_ref`;
- una entrega invalida produce `changes_requested` por defecto y evidencia de
  incidencias compactas;
- no registra quality gates durables por si mismo: solo adjunta
  `quality_gate_ref` al resultado de revision.

## CodexReviewGateProjectFileEvidenceV0

Adaptador externo de evidencias de ficheros para review gate.

Invariantes:

- lee solo paths relativos declarados en el ACK y bajo `project_work_dir`;
- rechaza paths absolutos, `..`, HOME, URLs o caracteres de control;
- cuenta lineas reales de ficheros regulares;
- marca ficheros ausentes o no legibles como incidencias de revision;
- no expone rutas absolutas ni detalles de filesystem al nucleo.

## FileCodexProgressStateStoreV0

Conector filesystem explicito para `CodexProgressStateStorePortV0`.

Invariantes:

- Requiere directorio absoluto configurado por operador.
- Persiste heartbeat compacto, firma observada y ultimo reporte emitido.
- Al recrear la instancia conserva contadores y evita repetir avisos ya emitidos.
- No es conector por defecto y no selecciona DB.
- El snapshot no contiene rutas, PID, HOME, OAuth, provider, modelo, prompt ni
  transcripts.
