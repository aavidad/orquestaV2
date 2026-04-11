## Runtime Worker Contract

Este bloque copia lo útil del runtime de `oh-my-codex` y asume `tmux` como transporte local canónico para agentes interactivos, sin convertir el terminal en frontera del dominio.

### Objetivo

Separar la observabilidad estructurada del worker de:

- `runtime.json`
- `tmux.log` o `log_path` equivalente
- heurísticas de transcript

De forma que el plano de control lea la misma base mínima en cualquier transporte admitido (`tmux`, broker remoto, Claude, Ollama) sin depender de PTY/FIFO.

### Artefactos canónicos por ejecución

Cada runtime local bajo `.orquesta-runtime/<agente>/<run>/` expone ahora:

- `runtime.json`
  - manifest técnico del launcher legacy
- `manifest.json`
  - identidad estable del worker y sus rutas
- `status.json`
  - estado vivo resumido del worker
- `heartbeat.json`
  - pulso periódico del broker

### Semántica

`manifest.json`
- estático por ejecución
- describe:
  - agente
  - proyecto
  - driver
  - transporte
  - comando renderizado
  - `child_pid`
  - rutas de `status/heartbeat/log/stdin`
  - `external_session_id`
  - `mailbox_delivery_mode`

`status.json`
- se actualiza durante la vida del worker
- estado mínimo actual:
  - `starting`
  - `running`
  - `stopped`
  - `failed`
- además publica:
  - `alive`
  - `updated_at`
  - `last_output_at`
  - `exit_code`
  - `exit_error`

`heartbeat.json`
- pulso corto del broker
- publica:
  - `alive`
  - `heartbeat_at`
  - `started_at`
  - `last_output_at`
  - `exit_code`
  - `exit_error`

### Norma

- `tmux` + `manifest/status/heartbeat` es la ruta canónica para runtimes locales interactivos.
- `PTY/FIFO` no debe ser la verdad canónica del worker ni el camino principal de control.
- la verdad mínima de runtime debe poder leerse desde `manifest/status/heartbeat`.
- cualquier runtime `tmux`, broker remoto o launcher equivalente debe escribir estos mismos artefactos.

### Estado actual

1. El runtime local `tmux` ya escribe `manifest.json`, `status.json` y `heartbeat.json`.
2. La promoción de `starting` a `running` depende de observar un pane listo o actividad real del worker, no de asumir que `new-session` implica disponibilidad.
3. `session_resume` y el estado estructurado mandan para continuidad; `tmux send-keys` queda para acciones operativas explícitas o reparación, nunca como bus genérico de trabajo.

### Siguiente paso natural

1. Consumir `status.json` y `heartbeat.json` en el doctor/supervisor/OpenClaw.
2. Borrar del control plane cualquier preferencia PTY-first que sobreviva en selección, presupuesto, diagnóstico o reconciliación.
3. Dejar `PTY/FIFO` solo como compatibilidad de recuperación o test mientras se elimina definitivamente del código residual.
