# Incidencia 2026-06-11: estados falsos de agentes Orquesta en OPES A1 informática

## Resumen

Durante la creación del temario OPES A1 informática, Orquesta registró y supervisó numerosos padres de tema como `running` o aceptados por cola, pero la comprobación externa por proceso real mostró que muchos no tenían agente Codex vivo ni entrega verificable. El director OPES tuvo que desbloquear con lanzamientos directos documentados y comprobación por PID.

Esto no es una regla OPES: es una incidencia de runtime/proyección de Orquesta aplicable a cualquier consumidor que delegue agentes externos.

## Contexto

- Consumidor: OPES.
- Trabajo: informática A1, 72 padres de tema, 1 padre por tema con 6 subroles preferentes.
- Runtime Orquesta usado desde OPES:
  `/home/alberto/Trabajo/OPES/opes-salidas/coordinacion_temarios/comunes_cierre_integral_2026-06-11/orquesta_local/`
- Estado de cola:
  `/home/alberto/Trabajo/OPES/opes-salidas/coordinacion_temarios/comunes_cierre_integral_2026-06-11/orquesta_local/state/run-state/queue_v0.json`
- Auditoría:
  `/home/alberto/Trabajo/OPES/opes-salidas/coordinacion_temarios/comunes_cierre_integral_2026-06-11/orquesta_local/state/audit/orquesta_server_audit_v0.jsonl`
- Rescates Orquesta:
  `/home/alberto/Trabajo/OPES/opes-salidas/coordinacion_temarios/a1_maestros_todas_opes_2026-06-11/informatica_a1_rescate_runs_2026-06-11/`
- Fallback directo documentado:
  `/home/alberto/Trabajo/OPES/opes-salidas/coordinacion_temarios/a1_maestros_todas_opes_2026-06-11/informatica_a1_padres_directos_2026-06-11/`

## Síntomas observados

1. `supervise` devuelve `status:"running"` con `wait_unhandled_outbox` y sin `process_ref` en múltiples temas.
2. En otros casos `supervise` devuelve `status:"running"` con `process_ref`, pero el director no puede confirmar todos los procesos vivos mediante `ps`.
3. La cola contiene 143 registros relacionados con informática A1 para 72 temas, incluidos registros originales y rescates, pero muchos records no exponen un `status` público útil en `queue_v0.json`.
4. La comprobación externa por `ps` detectó solo 6 temas con proceso Codex vivo asociado a Orquesta en ese momento: 41, 42, 43, 44, 48 y 52.
5. El fallback directo secuencial registró PIDs para temas 53-55, 57-64 y 66-72, pero los PIDs ya no estaban vivos al verificarlos; varios logs muestran salida rápida con errores de skills o entregas parciales, por lo que un PID histórico no basta como señal de ejecución viva.
6. El lanzamiento directo masivo de Codex provocó carrera del gestor de skills:
   `failed to install system skills: io error while remove existing system skills dir: Permission denied (os error 13)`.

## Evidencia mínima

Comando usado para distinguir informe completo, temas pendientes y procesos vivos:

```bash
python3 - <<'PY'
from pathlib import Path
import subprocess, re, json
base=Path('/home/alberto/Trabajo/OPES/opes-salidas/coordinacion_temarios/a1_maestros_todas_opes_2026-06-11/informatica_a1_72_padres')
needed={'matriz_reutilizacion_tema.md','informe_calidad_tema.md','informe_partes_rehacer_tema.md','checkpoint_director.md'}
complete=[]
for n in range(1,73):
    d=base/f'tema_{n:03d}'
    files={p.name for p in d.glob('*') if p.is_file()} if d.exists() else set()
    if needed <= files:
        complete.append(n)
missing=[n for n in range(1,73) if n not in complete]
ps=subprocess.check_output(['ps','-eo','pid,ppid,stat,etime,cmd'], text=True)
active=[]
for line in ps.splitlines():
    if 'codex --ask-for-approval never' not in line:
        continue
    m=re.search(r'a1-informatica-t(\d{3})-padre|tema_(\d{3})', line)
    if m:
        active.append(int(m.group(1) or m.group(2)))
print(json.dumps({
  'complete_count': len(complete),
  'missing_count': len(missing),
  'missing': missing,
  'active_count': len(set(active)),
  'active_topics': sorted(set(active)),
}, ensure_ascii=False, indent=2))
PY
```

Resultado observado:

```json
{
  "complete_count": 48,
  "missing_count": 24,
  "missing": [36, 41, 43, 44, 48, 52, 53, 54, 55, 57, 58, 59, 60, 61, 62, 63, 64, 66, 67, 68, 69, 70, 71, 72],
  "active_count": 6,
  "active_topics": [41, 42, 43, 44, 48, 52]
}
```

Ejemplos de respuestas Orquesta que no deben proyectarse como ejecución viva sin comprobación adicional:

- `response_t070_rescate-20260611-01.json`: `status:"running"` con `wait_unhandled_outbox`.
- `response_t044_retry01.json`: `status:"running"` con `wait_unhandled_outbox`.
- `response_t053_20260611.json`: `status:"running"` con `process_ref`, pero requiere comprobación de proceso vivo, ACK o entrega durable.

Error de concurrencia Codex/skills observado en logs directos:

```text
failed to install system skills: io error while remove existing system skills dir: Permission denied (os error 13)
```

## Impacto

- El director humano puede creer que hay padres trabajando cuando solo hay registros de cola o outbox sin agente vivo.
- Se relanzan rescates innecesarios y se duplican registros.
- El trabajo OPES se desvía a fallback directo, que debe ser excepcional.
- La supervisión por `status:"running"` deja de ser fiable para decidir si cortar, esperar, relanzar o cerrar.

## Tarea técnica derivada

Ver tarea ejecutable:

`/home/alberto/Trabajo/orquesta/.orquesta-runtime/opes-a1-informatica-orquesta-runtime-incidents-20260611/task_T260_fix_false_running_external_agents.json`

También queda registrada como `T260` en:

`/home/alberto/Trabajo/orquesta/docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`

## Criterio de cierre

Orquesta no debe mostrar ni devolver `running` para un agente externo salvo que exista una de estas pruebas:

- proceso vivo verificable por adaptador runtime;
- ACK o heartbeat reciente asociado al `process_ref`;
- lease/outbox en ejecución con worker vivo y propietario identificable;
- estado explícito distinto: `queued`, `waiting_outbox`, `stalled`, `launch_failed`, `completed` o `needs_rescue`.

Además, el estado público debe separar:

- run registrado;
- tarea en cola;
- outbox pendiente;
- proceso lanzado;
- proceso vivo;
- entrega recibida;
- cierre validado.

## Corte local Orquesta 2026-06-11

Cambios aplicados localmente para T260/T261:

- `wait_unhandled_outbox` deja de proyectarse como `running`: el supervisor
  Codex devuelve `waiting_outbox` sin `process_ref` cuando solo hay outbox
  pendiente.
- Un `process_ref` sin proceso verificable por el adaptador se proyecta como
  `stalled`, no como agente vivo.
- El estado público vivo se separa como `running_live` y exige evidencia de
  snapshot de proceso `running` o `stopping`.
- La respuesta MCP de supervisor transporta `agent_ref` para enlazar estado,
  proceso y entrega sin depender solo del run.
- El supervisor del servidor separa contadores públicos de `registered`,
  `waiting_outbox`, `running_live` y `stalled` sin duplicar el mismo outbox
  pendiente como ejecución viva.
- El wrapper Codex añade lock atómico de arranque sobre `CODEX_HOME` compartido
  para reducir carreras de instalación/refresh de skills. La ventana por
  defecto es cero en runtimes fake o `CODEX_HOME` aislado bajo el runtime del
  agente, y queda activa para el binario `codex` real con home compartido.

Estado operativo T261 para planners/scanners: el lock de skills de Codex esta
cerrado localmente con evidencia focal y queda marcado como
`estado_documental=sincronizado`. La unica deuda viva es el smoke Codex real
concurrente opt-in con `CODEX_HOME` compartido; nuevas coincidencias
documentales del error de skills no deben abrir otro padre de codigo salvo
regresion causal nueva.

Pruebas locales ejecutadas:

```bash
go test -count=1 ./modulos/orquesta-runtime-codex ./modulos/orquesta-mcp ./modulos/orquesta-app-codex-stack
go test -count=1 ./modulos/orquesta-server
go test -count=1 ./cmd/orquesta-server
```

Pendiente para cierre productivo:

- smoke Codex real acotado con varios agentes concurrentes y `CODEX_HOME`
  compartido, confirmando que no aparece `Permission denied` de skills;
- regresión OPES temporal o fake-runtime de 72 temas/rescates, confirmando que
  el director puede distinguir agente vivo, outbox pendiente, fallido y sin
  intento activo sin usar `ps`;
- T262 queda cubierto localmente en la capa neutral de cola/coordinador/MCP:
  `ProjectRunQueueAttemptsV0` agrupa intentos por refs opacas de
  consumidor/objetivo/item/write-set, calcula `active_attempt_ref` y expone
  `parent_run_ref`, `supersedes_run_ref` y `rescue_reason` en summaries y vista
  friendly. Hay regresión fake de 72 items y 143 registros. Pendiente solo una
  pasada OPES temporal contra datos reales si se exige cierre productivo de la
  incidencia fuera del núcleo.
