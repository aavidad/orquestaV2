# Handoff de continuación de Orquesta — 2026-07-30

## Objetivo invariable

Terminar Orquesta al 100 %, sin volver a convertir V23 en un bloque
indivisible. La prioridad vinculante corregida por el operador es:

1. abrir V38 `agent_runtime_elastic` sobre sus prerrequisitos ya acreditados,
   sin depender del cierre de V23 y con `ORC-28` como su única capacidad;
2. pasar la compuerta A del núcleo elástico neutral, sin exigir KVM ni
   Firecracker y sin acreditar todavía V38;
3. pasar la compuerta B del adaptador Firecracker mediante activación
   explícita, sin sustitución automática, con una microVM por agente y sin
   acreditar todavía V38;
4. pasar la compuerta C mediante una ola física real sobre el mismo candidato
   de A y B; solo A+B+C acreditan V38;
5. demostrar por separado cohortes lógicas de 1, 16, 70 y 500 y escalones
   físicos de 1, 5, 10, 16 y 20, sin afirmar 70 o 500 agentes físicos cuando
   no existan recursos medidos;
6. usar la propia Orquesta elástica para completar V23 y los demás frentes;
7. ejecutar gates globales, revisar y cerrar el producto.

`ORC-15` permanece en V27 y `OPS-16`/`OPS-17` permanecen en V32. La
continuidad de mensajes, la parada exacta y la conservación del entorno son
conductas estrechas exigidas por V38, no capacidades que V38 reabra o acredite.
`EVD-13` y `TestAttestor` siguen siendo prerrequisitos ya acreditados; el
adaptador Firecracker de agentes de la compuerta B no los sustituye. El runtime
general no se difiere a V39: V39 no existe.

No se debe declarar terminado un corte por porcentaje, documentación o pruebas
locales: hacen falta change-set, atestación, revisiones, gobernanza aplicable,
integración y evidencia durable.

## Primera acción del siguiente agente

La primera acción operativa es abrir una microtarea de la compuerta A de V38
sobre `ORC-28`, con write-set estrecho y sin KVM, Firecracker ni cambios de
roadmap. Antes de diseñarla o editar:

```bash
cd /home/alberto/Trabajo/orquestaV2
git status --short --branch
git log -5 --oneline --decorate

go test -mod=vendor -count=1 . \
  -run '^(TestProductRoadmapIsExhaustiveAndCausal|TestProductRoadmapV38OwnsElasticAgentRuntimeWithoutReopeningPrerequisites)$'

go test -mod=vendor -count=1 ./acceptance \
  -run '^(TestV38AgentRuntimeElasticPlanMatchesCanonicalRoadmap|TestV38AgentRuntimeElasticPlanRejectsSemanticDrift|TestV38AgentRuntimeElasticPlanRejectsInvalidJSON|TestV38AgentRuntimeElasticPlanRequiresExactRunPassEvidence)$'
```

Estas pruebas ratifican únicamente el contrato planificado: no implementan ni
acreditan V38. La primera microtarea de A debe partir de la fixture y demostrar
una conducta neutral del núcleo elástico detrás de los puertos existentes, sin
añadir otra autoridad de scheduling o lifecycle.

V23 queda preservada como frente posterior e independiente. No se relanza ni se
mezcla con el write-set de V38. Solo cuando se retome V23 se consulta su estado
vivo antes de editar:

```bash
STATE=/home/alberto/Trabajo/.orquesta-runtime-v2-v23/Codex12/state/orquesta-v23-microtasks-r3.sqlite
sqlite3 -header -column "$STATE" "
SELECT g.ref AS goal_ref,g.state,g.revision,w.state AS work_state,
       e.ref AS execution_ref,e.state AS execution_state,e.purpose,e.failure_code
FROM goals g
LEFT JOIN work_items w ON w.goal_ref=g.ref
LEFT JOIN executions e ON e.goal_ref=g.ref
WHERE g.ref IN (
  'goal:949f11c9635a1dc1b88d75a9103392d1',
  'goal:5a18a5071ae6bc9b0e13dc5c35f6eeda',
  'goal:d5daacfd3dba14624f50bdf89e0c89e1'
)
ORDER BY g.ref,e.created_at;"

sqlite3 -header -column "$STATE" "
SELECT o.goal_ref,o.kind,o.effect_intent_ref,o.last_error_code,i.digest
FROM outbox o
LEFT JOIN effect_intents i ON i.ref=o.effect_intent_ref
WHERE o.completed_at IS NULL AND o.retired_at IS NULL
ORDER BY o.available_at;"
```

No relanzar un frente hasta comprobar si su agente, change-set o artefacto ya
existe.

## Repositorios y runtime

- Producto: `/home/alberto/Trabajo/orquestaV2`.
- Rama: `integracion/v23-intake-durable`.
- Al escribir este handoff, HEAD es `cad35928` y la rama está dos commits por
  delante del remoto:
  - `2f3e9f4a docs: congela el alcance acotado de V23`
  - `cad35928 i18n: completa la ayuda del asistente V23`
- No hacer `push` sin una orden nueva del propietario.
- Semilla Git de Orquesta:
  `/home/alberto/Trabajo/.orquesta-runtime-v2-v23/Codex12/repositories/orquestaV2-v23-microtasks`.
- Estado SQLite:
  `/home/alberto/Trabajo/.orquesta-runtime-v2-v23/Codex12/state/orquesta-v23-microtasks-r3.sqlite`.
- Servicio de usuario: `orquesta-v23-pool.service`.
- API local: `http://127.0.0.1:18225`.
- Binario desplegado:
  `/home/alberto/Trabajo/.orquesta-runtime-v2-v23/tools/orquesta-804f519e`.
- La credencial se referencia mediante `--credential-file`; nunca se copia en
  documentación, prompts, logs ni comandos versionados.

## Trabajo vivo preservado

### ORC-03d: contrato nuclear de microtareas

- Goal: `goal:949f11c9635a1dc1b88d75a9103392d1`.
- Ejecución:
  `execution:1b96a3ad6ca2ead7230894a56cc3b164`.
- Estado al cerrar: autor activo.
- Write-set:
  - `docs/reconstruccion/contrato_minitareas_nucleo_2026-07-29.md`
  - `docs/reconstruccion/worksets/orc03_minitask_core_v1.json`
  - `orc03_minitask_core_plan_test.go`
- Prueba focal:

```bash
go test -mod=vendor -count=1 -v . \
  -run '^TestORC03MinitaskCorePlan$'
```

Debe corregir los hallazgos ya preservados en:

- `artifact:sha256:0669e611da025fc7de9ceb01231bce769a2f4556cbc0e35602779a76db5b4ca7`
- `artifact:sha256:36e38c0370f1bdedf22f1eb6e2fe734716e256202901ff032d4aaed18c2563b5`

No programar producción desde este Goal: solo contrato, manifiesto y test.

### BUG611b: límites de payload de revisiones

- Incidencia:
  `BUG-ORQ-20260730-611`.
- Goal: `goal:5a18a5071ae6bc9b0e13dc5c35f6eeda`.
- Ejecución:
  `execution:95f3efc68a5f2494ab06ca80bd5f3e0c`.
- Estado al cerrar: autor en `awaiting_integration`, atestación Firecracker
  `passed` con `exit_code=0`; revisiones principal y adversarial en cola.
- Candidato base preservado:
  `change-set:60b53618de802cfe892cadc6c46f1e39`.
- Change-set corregido:
  `change-set:b3fe43c194156e13d524ded6cdd9e426`, head
  `6d3990266f47b9f0fdf96c56b11d7453c0173d41`.
- Única corrección esperada: conservar el cambio y renombrar el test a
  `TestReviewerLaunchRequestPublishesReviewPayloadLimits`.
- Prueba focal:

```bash
go test -mod=vendor -count=1 -v ./internal/application \
  -run '^TestReviewerLaunchRequestPublishesReviewPayloadLimits$'
```

Está prohibido repetir `./internal/...` o `./...` durante esta autoría.

### V23-02: snapshot exacto del Wizard

- Goal revisado:
  `goal:d5daacfd3dba14624f50bdf89e0c89e1`.
- Change-set preservado:
  `change-set:ba9a164f39f2d321310e50c94667a895`.
- Head candidato: `17b59f7e57a59733e6c263938332c3a5cec25e77`.
- Revisión principal: `approve`.
- Revisión adversarial: `changes_requested`, artefacto
  `artifact:sha256:19341db540967c964e83bc47f8b6f19147f3d1251d171aa01a468d8dfb0bd4ed`.

El defecto es real: el candidato construye el binding después de persistir el
receipt y no lo transporta en `WizardGapsInputRecord` ni en las reservas. Tras
reinicio se recalcula; no existe persistencia causal atómica.

No hacer otro parche monolítico. Continuar el DAG ya diseñado en
`docs/reconstruccion/worksets/v23_agent_microtasks_v1.json`:

1. **V23-02 aplicación**, máximo seis ficheros: crear binding antes de reservar;
   transportarlo en record y reservas; validar ref, digest y bytes; distinguir
   replay durable de legacy.
2. **V23-03/04 SQLite**, migración forward-only y store: persistir receipt y
   snapshot en la misma transacción; no cambiar el hash del receipt ni inventar
   evidencia legacy.
3. **V23-05 recovery**: XOR estricto entre fila legacy sin snapshot y fila nueva
   con snapshot completo; tamper, restart y migración desde schema 21.
4. **V23-06 público**: `evaluation_replay_exact=true` solo al restaurar bytes
   durables exactos.

Después siguen V23-07, V23-09, V23-10, revisiones V23-11/12 y sello V23-13.
No afirmar que V23-03..06 ya están cerradas: el árbol actual solo contiene la
migración `021_wizard_gaps_inputs.sql`.

## Disciplina para cada microtarea

1. Leer `AGENTS.md`, instrucciones locales y fuentes vigentes.
2. Ejecutar preflight legacy aplicable.
3. Declarar un write-set disjunto y pequeño.
4. Usar Orquesta para crear y dirigir el Goal.
5. Ejecutar únicamente pruebas focales durante autoría.
6. Inspeccionar diff y comprobar `git diff --check`.
7. Aprobar commit y atestación Firecracker solo con evidencia real.
8. Exigir revisión principal y adversarial.
9. Resolver Council cuando la política lo requiera.
10. Integrar explícitamente y trasladar el commit al repo producto.
11. Commit pequeño, en castellano y con una sola responsabilidad.
12. Registrar cualquier fallo o falso verde en
    `docs/inventario_bugs_orquesta_2026-06-30.md`.

No usar `codebase-memory-mcp` por defecto, no lanzar indexadores, no esperar
todos los agentes vivos del run y no matar procesos por patrón. Para higiene:

```bash
ps -eo pid,ppid,stat,pcpu,pmem,etimes,comm,args --sort=-pcpu |
  rg 'orquesta|firecracker|jailer|codex|go test'
```

Un proceso solo se termina con owner/ejecución comprobados.

## Aprobación de efectos

Usar variables locales y un `request-ref` nuevo; nunca pegar el token:

```bash
BIN=/home/alberto/Trabajo/.orquesta-runtime-v2-v23/tools/orquesta-804f519e
TOKEN_FILE=/ruta/privada/al/token

"$BIN" command \
  --url http://127.0.0.1:18225 \
  --credential-file "$TOKEN_FILE" \
  --max-credential-bytes 4096 \
  --max-response-bytes 1048576 \
  --timeout 20s \
  --request-ref request:unico \
  --project-ref project:default \
  --payload '{
    "goal_ref":"goal:...",
    "intent_ref":"effect-intent:...",
    "expected_intent_digest":"...",
    "decision":"approved",
    "reason":"Evidencia inspeccionada y alcance acotado."
  }' \
  -- effects decide
```

El `intent_ref` y digest deben obtenerse del outbox vivo, no de este documento.

## Firecracker y V38 sin esperar al cierre de V23

El TestAttestor Firecracker está activo y sirve para atestar pruebas. Todavía no
es runtime de agentes.

V23 no depende de Firecracker, KVM ni microVM. A la inversa, V38 y su adaptador
Firecracker tampoco esperan a que V23 termine: avanzan mediante dependencias y
conjuntos de escritura separados.

Las tres compuertas canónicas son:

1. **A, núcleo neutral**: observación, reserva y liberación de capacidad,
   despacho global, prioridad de parada, progreso de observación, reinicio y
   recuperación sin KVM ni Firecracker. Esta compuerta no acredita V38.
2. **B, adaptador Firecracker**: activación explícita sin sustitución
   automática, una microVM por agente, `rootfs` inmutable con Codex, lease/CID
   durable, broker y proxy controlados por `vsock`, transporte aislado,
   credenciales efímeras, parada y sellado. Esta compuerta tampoco acredita
   V38.
3. **C, ola física**: ejecutar sobre el mismo candidato de A y B los escalones
   1, 5, 10, 16 y 20. Solo después de A+B+C puede acreditarse V38.

Las cohortes lógicas 1, 16, 70 y 500 prueban cálculo completo de demanda, no
prometen esas cantidades físicas. Si la capacidad no basta, la misma
`Execution` espera sin consumir intento.

No usar NAT, TAP, bridge, NIC guest, Internet directo ni Git del host. El guest
recibe un bundle/snapshot, trabaja en filesystem aislado y devuelve un
change-set al broker. El proxy host por vsock aplica allowlist y bloquea redes
locales, metadata, SSRF y comunicación lateral entre agentes. No reutilizar el
guest del TestAttestor como rootfs Codex.

## Regla de cierre

Preservar todo trabajo útil, incluso candidatos rechazados. No borrar legacy;
consultarlo mediante el índice vigente, sin mezclarlo en el árbol productivo.
Cuando una revisión encuentre un defecto real, crear una hija pequeña: no
ensanchar el WorkItem ni declarar que el control es un bloqueo circular.
