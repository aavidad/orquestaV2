# Handoff: inventario legacy de agentes pausado

Fecha: 2026-07-30.

## 1. Corte operativo

El inventario queda **pausado por orden del operador** después del lote 06.
No abrir el lote 07 hasta recibir una autorización nueva. El siguiente frente
es V38 `agent_runtime_elastic`; no se mezcla con este inventario.

Este handoff no crea trabajo, no cambia estado de capabilities, no acredita
conductas y no autoriza commit, push ni ejecución de Orquesta.

Snapshot local al redactarlo:

- rama `integracion/v23-intake-durable`, HEAD `7ae98026`;
- existen cambios ajenos en `internal/bootstrap/**` y otros ficheros untracked;
- preservarlos: no reset, checkout, limpieza amplia ni staging incidental.

## 2. Autoridad y estado honesto

Orden vigente:

1. `AGENTS.md`;
2. `product/roadmap.json`: 257 capabilities y 38 verticales;
3. `product/capabilities.json` y `product/evidence/**`: acreditación;
4. `docs/reconstruccion/ruta_total_100.md`;
5. `docs/reconstruccion/handoff_continuacion_agente_2026-07-30.md`;
6. este documento, solo como relevo operativo del inventario.

La fotografía `HANDOFF_PARADA_ORQUESTAV2_2026-07-26.md` es histórica donde
contradiga el handoff vigente.

Estado relevante:

- V1-V22 están acreditadas; V23-V38 no;
- V38 es la prioridad vigente, su contrato
  `AC-V38-AGENT-RUNTIME-ELASTIC` sigue `planned`;
- V38 posee únicamente ORC-28, hoy `declared`;
- su gate A es núcleo elástico neutral sin KVM ni Firecracker;
- B añade el adaptador Firecracker opt-in, una microVM por agente y sin
  fallback;
- C ejecuta escalones físicos 1, 5, 10, 16 y 20 sobre el mismo candidato;
- solo A+B+C pueden acreditar V38.

El inventario legacy es un read-model de caracterización. No es backlog,
roadmap, implementación, wiring, ejercicio ni acreditación.

## 3. Estado de los lotes 01–06

Los seis fixtures reúnen 29 caracterizaciones y 124 `TASKENTRY-*` únicos. No
hay refs ni `characterization_ref` repetidos entre lotes.

En todos los registros:

```text
review_state = bootstrap_first_review_pending_independent_counterreview
disposition = not_evaluated
canonical_state_change = false
creates_work_item = false
closes_capability = false
claims_accreditation = false
```

Por tanto, “confirmado” abajo significa únicamente que refs, hashes, líneas y
forma quedaron fijados por el contrato local. El orden exacto permanece en los
commits y datos actuales, pero el contrato del lote 01 no impide todos los
intercambios entre sus grupos. Nada de ello significa aceptación semántica ni
cierre de capability.

| Lote | Estado | Refs | Conductas (`behavior_key`: capability) |
| --- | --- | ---: | --- |
| 01 | versionado `c6bc7788` | 24 | `disponibilidad_y_cuota_vivas`: AGT-12 (7); `pools_sesiones_y_reservas`: ORC-28 (5); `decision_durable_previa_al_lanzamiento`: ORC-28 (6); `relevo_preventivo_de_sesion`: ORC-29 (6) |
| 02 | versionado `4e72aa7b` | 20 | `lanzamiento_neutral_con_contexto_resuelto`: AGT-01; `ciclo_goal_codex_lanzar_y_observar`: AGT-03; `supervision_y_recuperacion_codex`: AGT-03; `parada_exacta_solicitada_y_confirmada`: ORC-16; `aislamiento_concurrente_y_recuperacion_de_parada`: ORC-16 |
| 03 | versionado `51ecea47` | 20 | `catalogo_vivo_y_gestion_de_modelos`: AGT-02; `hermes_externo_api_mcp`: AGT-04; `conector_ollama_y_gestion_de_modelos`: AGT-07; `modelos_ollama_por_perfil_de_tarea`: AGT-07; `revisiones_independientes_del_mismo_candidato`: EVD-06 |
| 04 | versionado `c1448b0a` | 20 | `protocolo_comun_de_dispatch`, `peticion_neutral_de_runtime`, `solicitud_durable_a_launch_validado`, `runtime_de_proceso_controlado_y_aislado`, `adaptador_externo_opt_in`: AGT-01 |
| 05 | **untracked** | 20 | `composicion_codex_opt_in_y_compatibilidad`: AGT-03; `codex_real_aislamiento_y_apagado`: AGT-03; `escalado_de_modelo_por_salud_y_evidencia`: ORC-27; `normalizacion_y_contrato_runtime_neutrales`: AGT-01; `observacion_recuperacion_y_paridad_de_runtime`: AGT-01 |
| 06 | **untracked** | 20 | `lease_y_timeout_sin_reloj_oculto`, `identidad_fuerte_para_replay_y_cas`, `tick_y_supervision_idempotentes`: ORC-12; `control_activo_por_handle_y_orden`, `stop_exacto_pendiente_hasta_confirmacion`: ORC-16 |

Los focales y `-race` de 05/06 pasaron durante la autoría; 06 pasó además el
focal de paquete, JSON/LF y diff check. Ambos lotes necesitan contrarrevisión
independiente antes de decidir si se integran.

Los seis ficheros untracked que deben preservarse son los pares
`behavior_characterization_agent_batch_{05,06}_{test,support_test}.go` y los
fixtures `behavior_characterization_agent_batch_{05,06}.jsonl`.

Las refs exactas y su orden no se duplican en este handoff. La autoridad de
reanudación son los arrays `task_entry_refs` de cada fixture:

```bash
jq -r '[input_filename,.behavior_key,.capability_id,
        (.task_entry_refs|length),(.task_entry_refs|join(","))] | @tsv' \
  product/traceability/fixtures/behavior_characterization_agent_batch_0{1,2,3,4,5,6}.jsonl
```

## 4. Universo de `task_entries`

`product/traceability/task_entries.jsonl` contiene:

- 2108 líneas válidas;
- 2108 `entry_ref` únicos;
- 184 capabilities referenciadas;
- 2108 entradas con `closure_evidence=not_verified`.

Los lotes usan 124 refs; quedan 1984 refs no usadas por esos fixtures. Esa
resta no mide avance de producto ni garantiza que las restantes sean
pertinentes, independientes o agrupables.

Una entrada significa: fragmento histórico localizado, revisado
semánticamente y asignado a una capability en el ledger. No significa:

- capability nueva ni una de las 257 del roadmap;
- tarea pendiente, bug vivo, requisito aceptado o prioridad;
- implementación, wiring, ejercicio, receipt o acreditación;
- evidencia independiente si su rango se solapa con otro.

## 5. Algoritmo exacto para un eventual lote 07

No ejecutarlo mientras siga la pausa.

1. Releer autoridades y comprobar `git status`.
2. Declarar solo estos tres ficheros nuevos:
   `behavior_characterization_agent_batch_07_test.go`,
   `behavior_characterization_agent_batch_07_support_test.go` y
   `product/traceability/fixtures/behavior_characterization_agent_batch_07.jsonl`.
3. Incluir como usados también los lotes 05/06 aunque sigan untracked:

```bash
jq -r --slurpfile used <(
  jq -s 'map(.task_entry_refs[]) | unique' \
    product/traceability/fixtures/behavior_characterization_agent_batch_0{1,2,3,4,5,6}.jsonl
) '
  select(.entry_ref as $ref | ($used[0] | index($ref) | not))
  | [.capability_id,.entry_ref,.source_ref,.subject_first_line,
     .subject_last_line,.semantic_reason] | @tsv
' product/traceability/task_entries.jsonl
```

4. Elegir exactamente cinco conductas y cuatro refs ordenadas por conducta.
   Preferir otra familia funcional y capabilities repetidas entre grupos para
   poder mutar intercambios de igual capability.
5. Aceptar una ref solo si el ledger conserva `capability_decision=accept`,
   `semantic_review_state=reviewed` y `closure_evidence=not_verified`.
6. La capability se copia del ledger y se contrasta con `roadmap.json`; no se
   reasigna por intuición. Ante contradicción literal, descartar el candidato
   o detener ese grupo, no alterar autoridad desde el fixture.
7. Leer el bloque exacto únicamente en
   `/home/alberto/Trabajo/orquestaV2-legacy-consulta/<source_ref>`. No usar solo
   `semantic_reason`: puede resumir o generalizar más que el literal.
8. Por cada ref copiar sin transformación:
   `entry_ref`, `source_ref`, `source_sha256`, primeras/últimas líneas y
   `subject_sha256`.
9. Extraer una ancla literal independiente por ref, dentro de su rango, y
   fijar las cuatro anclas ordenadas en el support test.
10. Rechazar solape global cuando dos refs comparten `source_ref` y:
    `left.first <= right.last && right.first <= left.last`. Si no hubiera
    alternativa, declararlo y no contarlo como evidencia independiente.
11. Deduplicar semántica: `behavior_key` y `characterization_ref` nuevos; las
    cuatro fuentes deben sostener el mismo invariante, no solo compartir
    capability o palabras.
12. Redactar `worked`, `did_not_work` y `attempts` siempre como afirmaciones
    históricas calificadas. Mantener propuesta no canónica, sin trabajo,
    cierre ni acreditación.
13. El validador debe exigir 5×4, orden exacto, 20 refs inéditas, procedencia
    contra ledger, campos no vacíos, anclas, rangos disjuntos y refs de
    caracterización nuevas.
14. Negativos mínimos: 3/5 refs, capability incorrecta, ref anterior,
    intercambio de todos los pares de grupos con igual capability, ancla,
    autoridad/flags, campo desconocido, clave duplicada, valor posterior,
    representación no canónica y LF ausente.

Antes de editar, consultar lecciones por cada capability elegida:

```bash
scripts/consultar_lecciones_legacy.sh \
  --capability ID \
  --path product/traceability/fixtures/behavior_characterization_agent_batch_07.jsonl \
  --operation caracterizar_CONDUCTA
```

Una salida sin patrones es un hueco advisory; no autoriza inventar semántica.

## 6. Verificación de un eventual lote 07

```bash
gofmt -w \
  behavior_characterization_agent_batch_07_test.go \
  behavior_characterization_agent_batch_07_support_test.go

go test -mod=vendor -count=1 . \
  -run '^TestBehaviorCharacterizationAgentBatch07'

go test -mod=vendor -race -count=1 . \
  -run '^TestBehaviorCharacterizationAgentBatch07'

jq -e . \
  product/traceability/fixtures/behavior_characterization_agent_batch_07.jsonl \
  >/dev/null

test "$(wc -l < product/traceability/fixtures/behavior_characterization_agent_batch_07.jsonl)" -eq 5
test "$(tail -c 1 product/traceability/fixtures/behavior_characterization_agent_batch_07.jsonl |
  od -An -t u1 | tr -d ' ')" = 10

git diff --check -- \
  behavior_characterization_agent_batch_07_test.go \
  behavior_characterization_agent_batch_07_support_test.go \
  product/traceability/fixtures/behavior_characterization_agent_batch_07.jsonl
```

Como los tres ficheros serán nuevos, `git diff --check` no basta. Verificar
cada uno también con `git diff --no-index --check /dev/null FICHERO`; código
1 sin salida significa “diferencias sin error de whitespace”, código mayor que
1 o cualquier salida es fallo.

El focal Go es quien ratchea JSON canónico mediante `json.Marshal`,
`DisallowUnknownFields`, rechazo de valores posteriores y LF final.

## 7. Bloqueos y prohibiciones

- La pausa es el bloqueo explícito del lote 07.
- No editar, borrar, formatear ni “corregir” lotes 01–06 durante la pausa.
- No commitear 05/06 antes de contrarrevisión independiente.
- No activar Orquesta, hacer push, deploy, publicación ni efectos externos.
- No desactivar sparse-checkout ni materializar `modulos/**`.
- La copia legacy es solo lectura; no copiar paquetes ni crear bridges.
- No tratar las 2108 entradas como backlog o porcentaje de completitud.
- No cambiar roadmap, capabilities ni evidence desde una caracterización.
- No mezclar inventario con V38, Firecracker, V23 ni cambios ajenos activos.
- No usar indexadores ni `codebase-memory-mcp`; `rg`, `jq` y lectura parcial
  bastan.

## 8. Cambio de contexto a V38

La siguiente microtarea debe venir de una asignación nueva. Seguir
`handoff_continuacion_agente_2026-07-30.md`: primero una conducta estrecha del
gate A sobre ORC-28, sin KVM ni Firecracker; después gate B podrá trabajar el
adaptador Firecracker opt-in. No atribuir al inventario ninguna autoridad para
ese diseño.

## 9. Cierre de este relevo

hecho: inventario pausado y punto de reanudación preservado
invariante: 124 refs únicas; ninguna crea estado canónico ni acreditación
autoridad final: roadmap/capabilities/evidence; fixtures solo propuesta
tests: no se reejecutaron lotes durante este relevo
receipts y revisión: ninguno; contrarrevisión independiente pendiente
código retirado: ninguno
legacy retirado: ninguno; consulta preservada en solo lectura
riesgos P0/P1: no perder ni integrar sin revisar los seis ficheros untracked
siguiente dependencia causal: nueva microtarea V38; lote 07 continúa pausado
