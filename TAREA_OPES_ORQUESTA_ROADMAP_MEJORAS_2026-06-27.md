# Roadmap de mejoras Orquesta (priorizado impacto/esfuerzo)

Fecha: 2026-06-27.
Para: producto + programador. Complementa el paquete de implementación
(`TAREA_OPES_ORQUESTA_ORDEN_IMPLEMENTACION_PRS_2026-06-27.md`). Aquí va lo de
fondo y las **ideas nuevas** (ahorro de tokens, wizard, técnicas).

Notación: Impacto (Alto/Medio/Bajo) · Esfuerzo (S/M/L).

---

## Tier 0 — Hacer la autonomía real (ya en el encargo)
- **Observador goal-first + estado vivo** (T1/T2). Impacto Alto · Esfuerzo M.
- **Reconciliador de runs de 3 fuentes** (ficheros + procesos vivos en
  `ProcessRegistry` + últimos mensajes). Mata la familia "running fantasma /
  stop vs materialización tardía / ACK no escrito". Impacto Alto · Esfuerzo M.
- **Guard de integridad de entrega**: si el agente anuncia ACK y no hay
  `agent_ack.json`, emitir `completed_without_ack_file`. Impacto Alto · Esfuerzo S.

## Tier 1 — Confianza y observabilidad
- **Endpoint único de estado global accionable** (acotado, siempre con cuerpo):
  responde "¿la ola termina sola o necesita acción?". Impacto Alto · Esfuerzo M.
- **Backpressure explícito**: `blocked_by_capacity` con número en vez de
  conexión colgada. Impacto Alto · Esfuerzo M.
- **Telemetría del lazo**: ratio idle→programó→cerró, % cerrados sin
  intervención, tokens por goal. Sin esto no sabes si mejora entre versiones.
  Impacto Alto · Esfuerzo M.

## Tier 2 — Salud del código (deuda que ya muerde)
- **Gate anti-código-muerto + staticcheck/govulncheck/-race en CI**. Impacto
  Medio · Esfuerzo S.
- **Tests deterministas** (reloj inyectado + señales, fuera `time.Sleep`); hoy es
  sistémico. Impacto Medio · Esfuerzo M.
- **Decidir validación muerta (Grupo B)**: cablear o borrar. Impacto Alto (ver
  idea T-1 de tokens) · Esfuerzo M.

## Tier 3 — Subir el techo de capacidad
- **Auto-mejora alimentada por señales reales** (tests fallando, staticcheck,
  vulns, código muerto) como fuente del backlog. El orquestador se arregla solo.
  Impacto Alto · Esfuerzo M.
- **Presupuesto/relevo sin supervisión** (cuota/token; módulos op_052/op_089).
  Impacto Alto · Esfuerzo L.
- **Memoria de decisiones entre runs** (qué reconciliaciones/relevos funcionaron).
  Impacto Medio · Esfuerzo L.

---

# Ideas nuevas (brainstorm)

## A. Ahorro de tokens sin perder calidad
Ya existen matriz modelo/razonamiento, gate de review por diff y capacity/budget.
Construir encima:

- **T-1 (la más rentable): validación determinista en vez de LLM.** La política
  de ACK/write-set/rails (`codex_ack_*`, Grupo B del audit) es **código puro hoy
  muerto**. Cablearla hace que la verificación de entrega (¿el ACK respeta el
  write-set? ¿hay rails pendientes? ¿detalle local prohibido?) sea **determinista,
  gratis en tokens y más fiable** que pedírselo a un modelo. Doble victoria:
  ahorro + cierra incidencias de ACK. Impacto Alto · Esfuerzo M.
- **T-2: observación evidence-first (clave para T1).** El ticker observador NO
  debe invocar LLM en cada tick. Orden barato→caro: (1) ¿hay `agent_ack.json`?
  (2) ¿proceso vivo en `ProcessRegistry`? (3) ¿required tests pasan? Solo si queda
  **ambiguo** se invoca modelo. Evita gastar tokens en "vigilar". Impacto Alto ·
  Esfuerzo S (es decisión de diseño dentro de T1).
- **T-3: prompt caching del prefijo estable.** Para las llamadas vía runtime
  Claude/MCP, estructurar el prompt con prefijo estable cacheado (reglas, skills,
  contratos, contexto de proyecto) y solo la cola variable fuera de caché. En
  agentes largos esto recorta drásticamente el coste de los tokens de entrada
  repetidos. Verificar uso real (solo vi `cache_control` en el MCP server).
  Impacto Alto (en rutas Claude) · Esfuerzo M.
- **T-4: contexto por referencia + deltas.** El contrato ya dice "solo refs, no
  payloads". Llevarlo al runtime: el agente **tira del contexto por ref bajo
  demanda** en vez de recibir todo el bundle materializado. Y en rework, mandar
  **solo el delta** desde el checkpoint. Impacto Medio · Esfuerzo M.
- **T-5: pre-pase con modelo barato (Haiku) para triaje.** Dedup de subtareas,
  clasificación de fallos, decidir si una run necesita rework, resumen de logs:
  todo eso a modelo barato; reservar el caro para codificar/revisar. Encaja con la
  matriz existente; falta cablear el "router por dificultad". Impacto Alto ·
  Esfuerzo M.
- **T-6: replay desde checkpoint en vez de reinicio.** Hay time-travel (op_092).
  En rework, re-ejecutar desde el último checkpoint bueno en lugar de relanzar el
  goal entero ahorra los tokens de rehacer lo ya correcto. Impacto Alto · Esfuerzo M.
- **T-7: cancelación temprana de subroles perdedores.** Ya se lanzan 6 subroles
  en paralelo; cancelar en cuanto uno satisface el criterio o queda obsoleto evita
  quemar tokens en ramas muertas. Impacto Medio · Esfuerzo M.
- **T-8: caché de required-tests.** No re-ejecutar (ni re-validar con LLM) un test
  cuyo write-set no cambió; cachear por fingerprint (ya hay `*_fingerprint_v0`,
  hoy legacy/muerto — reutilizable). Impacto Medio · Esfuerzo S.

## B. Wizard (/nueva-app)
Está ~90%, accesible e i18n. Siguientes saltos:

- **W-1: dry-run/preview antes de gastar tokens.** Antes de lanzar, mostrar el
  `GoalWorkSpec` resultante: objetivo, write-set, tests requeridos, **estimación de
  coste/tokens/tiempo** y modelo elegido por la matriz. El usuario aprueba con
  datos. Impacto Alto · Esfuerzo M.
- **W-2: galería de plantillas / seed.** Empezar desde una app similar o
  **importar un repo existente** para sembrar el spec en vez de partir de cero.
  Impacto Alto · Esfuerzo M.
- **W-3: gate de viabilidad.** Validar coherencia del spec (objetivo vs write-set
  vs tests) y avisar de ambigüedades **antes** de lanzar agentes. Impacto Medio ·
  Esfuerzo S.
- **W-4: "explícame esta opción" con modelo barato** inline en cada nodo del
  wizard (ya hay `data-help`/tooltips; añadir explicación generada barata).
  Impacto Bajo · Esfuerzo S.
- **W-5: presupuesto visible y tope.** Slider de presupuesto y parada automática
  al alcanzarlo, conectado a capacity/budget. Impacto Medio · Esfuerzo M.

## C. Otras
- **O-1: la auditoría como entrada de auto-mejora.** Que el propio Orquesta corra
  `staticcheck/govulncheck/-race`, y meta los hallazgos en su backlog goal-first.
  Cierra el bucle estrella: se audita y se arregla solo. Impacto Alto · Esfuerzo M.
- **O-2: "confidence/cost ledger" por run** (tokens, intentos, reconciliaciones)
  para depurar y para que la matriz aprenda. Impacto Medio · Esfuerzo M.
- **O-3: degradación elegante multi-proveedor.** Si Codex agota cuota, relevar a
  otro runtime (Claude/Gemini/Ollama) sin perder el goal. Hay runtimes; falta la
  política de relevo por proveedor. Impacto Medio · Esfuerzo M.

---

## Recomendación: las 5 de mayor palanca para empezar
1. **T-1** validación determinista de ACK (ahorro + fiabilidad + cierra incidencias).
2. **T-2** observación evidence-first (hace T1 barato en tokens).
3. **Endpoint de estado global accionable** (Tier 1).
4. **W-1** dry-run con coste estimado en el wizard (confianza + control de gasto).
5. **O-1** auditoría como fuente de auto-mejora (capacidad estrella, bucle cerrado).

Todas son Impacto Alto y Esfuerzo S/M, y varias se apoyan en piezas que **ya
existen** (validación muerta, fingerprints, matriz de modelo, time-travel).
