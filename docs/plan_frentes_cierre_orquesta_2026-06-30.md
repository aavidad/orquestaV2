# Plan de frentes de cierre Orquesta - 2026-06-30

Este documento es una guia operativa para cerrar Orquesta por piezas pequenas
sin perder la vision global. No sustituye a `AGENTS.md`,
`docs/estado_actual_2026-05-17.md` ni al backlog de autoprogramacion; resume el
orden practico para que cada agente elija un frente acotado, deje evidencia y no
reabra trabajo cerrado.

## Principio

Orquesta no se cierra por porcentaje declarado. Se cierra cuando cada frente
tiene contrato, runtime opt-in, prueba focal, evidencia durable y una ruta clara
para que el siguiente agente no repita el mismo arreglo.

Cada cambio debe responder a una pregunta pequena:

- que bug o hueco global reduce;
- que fichero o modulo queda como owner;
- que prueba demuestra que no es solo documentacion;
- que queda fuera del frente.

## Frentes vigentes

### F1. Goal-first y loop delgado

Objetivo: el modo nuevo debe preparar `GoalWorkSpecV0`, lanzar por adaptador
goal opt-in, observar resultado y validar cierre por evidencias. El loop
historico solo entra con `director_execution_mode=legacy_director_loop`.

Criterio de cierre:

- modo vacio normaliza a `goal_first`;
- sin backend Goal devuelve `goal_backend_unavailable`;
- no hay fallback silencioso al loop legacy;
- el backend Codex usa `app_server_tmux`, no stdio;
- status y runbooks muestran el perfil self-programming aislado sin rutas
  sensibles.

### F2. Autoprogramacion autosuficiente

Objetivo: Orquesta debe detectar bloqueos propios, abrir tareas pequenas de
reparacion y dejar evidencia antes de cerrar.

Criterio de cierre:

- incidencias recurrentes se promocionan a backlog ejecutable;
- cada automejora tiene write-set, tests y receipt;
- los bloqueos por quota, runtime, permisos o puerto quedan como frontera
  externa documentada;
- no se convierte un workaround manual en flujo normal.

### F3. OPES como consumidor aislado

Objetivo: OPES debe usar Orquesta por conectores y refs opacas, sin tocar
produccion ni compartir filesystem interno como contrato del nucleo.

Criterio de cierre:

- efectos reales solo sobre instancia temporal confirmada o scope opt-in;
- `job_ref`, `job_type`, `program_id`, `topic_id` o correlacion equivalente
  acotan el trabajo;
- cierre OPES valida artefactos reales, no presencia superficial;
- RAG regenerable solo cierra con `rag/corpus/chunks.jsonl`,
  `rag/corpus/summary.json` y manifest coherente;
- audio cuenta manifiestos por `html_final/tema_*.html` y
  `html_ampliado/tema_*.html`, no por `index.html` o portadas.

### F4. Web Nueva App

Objetivo: la web debe poder orientar al usuario hasta un contrato de app sin
exigir que conozca todas las opciones de entrada.

Criterio de cierre:

- wizard conversacional pregunta dudas y rellena contrato progresivamente;
- formulario experto permite datos, sensibilidad, bases relacionales,
  vectoriales u objeto, multiples integraciones y arquitectura no obligatoria;
- hexagonal queda por defecto si no rompe contrato;
- validaciones y mensajes visibles estan en castellano;
- cada opcion compleja tiene tooltip y documentacion profunda enlazada.

### F5. Conectores y contratos

Objetivo: cada conector traduce dominio externo a trabajo orquestable y devuelve
evidencia verificable al dominio propietario.

Criterio de cierre:

- no hay reglas de producto dentro del core;
- cada conector tiene contrato documental y test focal;
- los artefactos esperados se validan por tipo, refs y checksums cuando aplique;
- los errores recuperables se normalizan o pasan a rework, no a descarte.

### F6. Operacion remota segura

Objetivo: permitir que un agente Codex remoto continue el cierre de Orquesta sin
tocar produccion ni abrir superficie externa.

Criterio de cierre:

- usuario no-root, sudo solo cuando proceda;
- worktree aislado y estado/logs fuera de servicios productivos;
- sin puertos publicos nuevos;
- sin docker.sock/root ni acceso a OPES productivo;
- cada ciclo remoto exporta patch, summary, pruebas y siguiente gap.

## Orden recomendado

1. Cerrar P0/P1 de seguridad y no produccion antes de cualquier smoke real.
2. Cerrar goal-first/tmux antes de ampliar autoprogramacion.
3. Convertir incidencias recurrentes en checks ejecutables, no solo criterios de
   texto.
4. Mejorar la web sobre contratos ya existentes, no inventando flujo paralelo.
5. Ejecutar smokes reales solo con instancia temporal o fixture aislada.

## Regla de fragmentacion

Un frente pequeno es valido si se puede terminar con commit, tests y una frase
clara de "siguiente hueco". Si no se puede explicar asi, hay que partirlo antes
de programar.
