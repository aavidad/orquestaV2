<!--
Revisión de arquitectura / deuda — Orquesta
Autor del análisis: revisión asistida (Claude Code)
Fecha: 2026-06-28
ESTADO: EN CURSO (fichero incremental — se va ampliando)
-->

# Revisión de arquitectura y deuda técnica — Orquesta

Fecha: 2026-06-28. Foco pedido: **arquitectura / deuda** (no bugs de lógica ni build).
Documento incremental: cada hallazgo se añade a medida que se analiza, para que el
progreso sobreviva a cualquier corte de sesión.

## Contexto del repo
- Monorepo Go, **un solo `go.mod`** (`orquesta`), ~**3200 ficheros `.go`**, **85 paquetes**.
- Módulos mayores por nº de ficheros: `cmd/orquesta-server` (327), `orquesta-app-codex-stack` (321),
  `orquesta-mcp` (237), `orquesta-core-workflow` (230), `orquesta-orchestration-core` (215).
- Doctrina viva en `docs/BIBLIA_APP_ORQUESTA.md`; arquitectura objetivo en `ARQUITECTURA.md`.

---

## Hallazgos

### H1 — Convención de nombres `_v0` en casi todo el código (deuda de naming) · severidad: baja
- **2969 de ~3200** ficheros `.go` llevan sufijo `_v0` (`guardian_check_v0.go`, etc.).
- No son ficheros generados ni con build tags: es una convención de versión incrustada en el
  nombre de fichero. Todos están en `_v0`; no hay `_v1`.
- **Riesgo:** si nunca habrá `_v1`, el sufijo es ruido en 2969 nombres y dificulta lectura/grep.
  Si la intención es versionar por fichero, no escala (acoplamiento del nº de versión al nombre).
- **Recomendación:** decidir doctrina explícita: o se elimina el sufijo (rename masivo, mecánico)
  o se documenta qué significa y cuándo nace `_v1`. Hoy es deuda silenciosa.

### H2 — Disciplina de fronteras existe y está testeada (fortaleza, con matices) · severidad: info
- `architecture_boundaries_test.go` prohíbe que el núcleo neutral (`orquesta-core`,
  `orquesta-core-workflow`, `orquesta-domain-work`) importe adaptadores de producto, `net/http`,
  `os/exec`, `cmd/`, etc. **Buena práctica.**
- Pendiente de verificar: que el test pase hoy y que cubra TODOS los paquetes que deberían ser
  neutrales (no solo los 3 listados).

### H3 — `orquesta-core-workflow` es un "god package" central · severidad: media-alta
- Paquete interno **más importado**: **762 referencias** desde el resto del repo.
- **154 ficheros `.go` (sin tests) en UN ÚNICO paquete plano**, sin subpaquetes. Es el corazón
  del sistema y a la vez el punto de mayor acoplamiento entrante.
- Le sigue `orquesta-orchestration-core` (424 imports). Dos hubs concentran casi toda la gravedad.
- **Riesgo:** cualquier cambio aquí impacta a casi todo el repo; los tests de este paquete son
  un cuello de botella; difícil de razonar y de evolucionar de forma aislada.
- **Recomendación:** dividir por subdominios en subpaquetes con superficie pública mínima
  (p. ej. `.../workflow/state`, `.../workflow/transitions`), reduciendo la API exportada que el
  resto del repo puede tocar. Es la deuda estructural más rentable de atacar.

### H4 — Frontera neutral verificada (fortaleza confirmada) · severidad: info
- `architecture_boundaries_test.go` **pasa hoy** (`ok 0.157s`). El núcleo neutral no importa
  adaptadores de producto, `net/http` ni `os/exec`. Disciplina real y barata de mantener.
- Matiz: la lista de paquetes "neutrales" cubiertos por el test es corta (core, core-workflow,
  domain-work). Conviene revisar si `orquesta-orchestration-core` (segundo hub) debería estar
  también bajo la misma frontera.

### H5 — Ficheros monolíticos (hotspots de complejidad) · severidad: media
- Top: `orquesta-web/nueva_app_html_render_v0.go` **1262 líneas**, `goal_first_v0.go` 909,
  `cmd/orquesta-server/codex_goal_app_server_v0.go` 897, y 8 más por encima de 700 líneas
  (mayoría en `orquesta-app-codex-stack`).
- **Riesgo:** ficheros de >700-1200 líneas concentran lógica difícil de testear y revisar;
  el de render HTML probablemente mezcla presentación y lógica.
- **Recomendación:** trocear los >700 líneas por responsabilidad; priorizar el render HTML y
  los `run_supervisor_*`/`run_coordinator_*` de app-codex-stack.

### Nota — duplicación entre `runtime*`: NO confirmada
- `orquesta-runtime` (40), `runtime-codex` (25), `runtime-codex-delivery` (37) **no comparten
  basenames**: no hay duplicación evidente de conceptos por nombre. Parecen capas (genérico →
  codex → entrega) más que copias. Se rebaja esta sospecha; revisión de contenido pendiente si se pide.

### H6 — Código muerto: muy poco a nivel de paquete (sano) · severidad: baja
- De **81 módulos**, solo **1** sin imports entrantes a nivel de módulo: `orquesta-work-profiles`,
  y resulta ser un módulo **solo-documentación** (AGENTS.md, README.md, docs/; **0 ficheros `.go`**).
  No es código muerto, es un paquete doc. Anotar como tal.
- A nivel de paquete-directorio, 5 de los 6 candidatos son **adaptadores test-only legítimos**
  (no borrar): `orquesta-agent-process-registry-memory` (24 importadores, todos en tests),
  `orquesta-run-memory` (14, todos tests), `orquesta-domain-work-sql` (6, tests),
  `orquesta-domain-work-memory` (3, tests), `orquesta-domain-work/contracttest` (3, tests).
  Son dobles de test / backends alternativos por inyección. Sanos.
- **ÚNICO candidato real a código muerto:** `orquesta-state-file/outbox` → **0 importadores,
  ni siquiera en tests**. Verificacion posterior 2026-07-01: sin build tags ni wiring dinamico
  detectado; retirado en el cierre `ARCH-ORQ-20260630-004`. El ledger file-based vivo queda en
  `orquesta-persistence`.
- **Conclusión:** el repo está sorprendentemente limpio de código muerto a nivel de paquete.

### H7 — Higiene de árbol de trabajo: 17 GB de datos de runtime (no es código) · severidad: media (operativa)
- `.orquesta-runtime/` ocupa **17 GB** y `.orquesta-purged-*` ~**845 MB** en el árbol de trabajo.
- **Están git-ignored** (no se commitean — correcto), pero conviven con el código fuente.
- No rompen `go test ./...` (el go tool ignora dirs que empiezan por `.`), pero **sí ralentizan**
  cualquier `find`/grep/herramienta que recorra el árbol y ocupan disco que puede contribuir a
  presión de memoria/IO en la máquina (relacionado con los cortes de sesión por recursos).
- **Cierre 2026-07-01:** `scripts/orquesta_runtime_retention.sh` inventaria candidatos con
  `dry-run` por defecto y solo borra con confirmacion literal; el runbook vivo es
  `docs/runbooks/limpieza_runtime_local_orquesta_2026-07-01.md`. Mantener runtimes fuera del cwd de
  trabajo sigue siendo recomendable para despliegues largos.

---

## Pendiente de analizar (si se continúa)
- [ ] Cohesión interna real de los dos hubs y plan de troceo de `core-workflow`.
- [ ] Superficie pública exportada de los hubs (cuánta API innecesaria expuesta).
- [ ] Contenido de los `runtime*` para confirmar/descartar solapamiento lógico.
- [ ] Paquetes/ficheros huérfanos (sin imports entrantes).
