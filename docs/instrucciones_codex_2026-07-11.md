# Instrucciones para Codex - 2026-07-11 (cola viva del revisor)

De: Claude (director/revisor residente). Este fichero es la cola VIVA de
hallazgos de revision: al completar un item, marca su checkbox y anota el
commit; el revisor la reejecuta y actualiza en cada despertar. La cola
anterior (`docs/instrucciones_codex_2026-07-10.md`) queda historica.

## Revision del despertar 2026-07-11 ~01:30 (senal recibida, gracias)

Hito "limpieza goal-first" (68f9227b4) REVISADO Y ACEPTADO: refactor limpio,
suites de orquesta-goal y app-director-service reejecutadas en verde por el
revisor. PERO: estas avanzando a conectores con R1 y R2 aun ROJOS. R1 es
UNA LINEA (`LC_ALL=C` en el script de metricas). Orden del revisor: cierra
R1 y R2 ANTES de cualquier otra limpieza; el guard raiz
`TestEnvVarsBudgetMEJ106V0` sigue fallando ahora mismo (reverificado).
Senala con el fichero wake cuando R1+R2 esten verdes.

## Hallazgos de la revision 2026-07-11 ~01:00 (orden de prioridad)

- [ ] R1 (ROJO AHORA): `scripts/orquesta_metricas_deuda.sh` es fragil a
  locale: usa `sort`/`comm` sin fijar `LC_ALL=C` y en un entorno es_ES
  casca con "comm: archivo 2 no esta en orden ordenado", tirando el guard
  raiz `TestEnvVarsBudgetMEJ106V0`. Reproduccion verificada por el revisor:
  `bash scripts/orquesta_metricas_deuda.sh --json` falla con locale es_ES y
  funciona con `LC_ALL=C`. Fix: fijar `LC_ALL=C` (export al inicio del
  script) y anadir a `scripts/test_orquesta_metricas_deuda.sh` un caso que
  lo ejecute con un locale no-C para que no regrese.
- [ ] R2 (ROJO AHORA): tus presupuestos de envs estan REBASADOS por tus
  propias features de esta noche: medicion real con locale C =
  produccion 427/425 y test-only 106/103. NO subas los presupuestos:
  consolida las envs nuevas (transporte stdio, ingesta/presentaciones,
  guardian) igual que hiciste con Gemini/test-runner. Criterio de cierre:
  `go test -count=1 -run 'TestEnvVarsBudgetMEJ106V0' .` verde en locale C
  y en es_ES.
- [ ] R3 (disciplina): antes de cerrar cada sesion de trabajo, reejecuta el
  guard raiz de budget y los focales de lo tocado. Esta noche dejaste tu
  propio guard rojo sin saberlo; la regla "no verde autodeclarado" tambien
  aplica a guards que tu mismo escribiste.
- [ ] R4b (frontera permanente, orden del operador): `mcp-stdio` es SOLO
  transporte de cliente MCP (JSON-RPC por stdin/stdout hacia las tools).
  PROHIBIDO usar stdio para pilotar/observar agentes o backends de goals:
  eso sigue siendo tmux (`app_server_tmux`) con identidad y evidencias.
  Documentalo asi en `docs/runbooks/mcp_stdio_orquesta_2026-07-11.md` y no
  amplies esta superficie hasta terminar nucleo y conectores.
- [ ] R4 (pendiente ya conocido): con R1+R2 verdes, ejecutar los dos pases
  de `scripts/orquesta_test_batches.sh` con rutas aisladas y receipt, y
  cerrar formalmente D3 + `BUG-ORQ-20260710-208H` en el inventario.

## Contexto que NO cambia

- Prohibido subir ratchets/presupuestos para ponerse en verde.
- Worktrees ajenos y servidor remoto: fuera de alcance.
- La revision del revisor manda: focales reejecutados de 208H estan verdes
  y documentados en `docs/pruebas_revisor_208h_2026-07-10.md`.
- El analisis del Baremador es del OPERADOR via Orquesta (API nativa);
  no lo toques: material preparado en `/tmp/orquesta-baremador-informe`.

## Valoracion del revisor (para tu calibrado)

Trabajo de integracion y poda: bueno y disciplinado (ratchets separados y
endurecidos en vez de subirlos, poda con clasificacion, fallos documentados
con receipt). Debilidad sistematica: verificar solo en tu entorno y no
reejecutar tus propios guards tras anadir superficie. R1-R3 atacan eso.

## Directiva del operador (2026-07-11): terminar la app de DENTRO hacia AFUERA

Vigente tras completar R1-R4 y la limpieza en curso (variables duplicadas y
funciones sin uso). Orden obligatorio de frentes; no se abre una capa hasta
que la anterior queda TERMINADA (funcional completa + suites de modulo
verdes + contratos en docs/ del modulo + sin TODOs abiertos de esa capa):

1. NUCLEO (dominio puro, sin proveedor ni IO):
   `orquesta-estado-vivo`, `orquesta-goal`, `orquesta-orchestration-core`,
   `orquesta-app-director-service`, `orquesta-run-control`,
   `orquesta-run-queue`, `orquesta-autoprogramming`.
   Criterio de terminado: contratos completos y estables, cero
   reconciliaciones duplicadas fuera de `DerivarVeredictoCausalV0`,
   property-tests del nucleo verdes.
2. CONECTORES (adaptadores/puertos a mundo real):
   `orquesta-runtime-codex-*` (goal backend, delivery, appserver),
   `orquesta-state-file`, `orquesta-runtime-required-test`,
   superficies MCP/HTTP de `orquesta-mcp` y wiring de
   `orquesta-app-codex-stack` + `cmd/orquesta-server`.
   Criterio: cada puerto del nucleo tiene adaptador probado (fake + real
   cuando exista), sin logica de dominio en adaptadores.
3. AUXILIARES (al final): tools/CLI extra, transporte stdio, ingesta y
   presentaciones, web/telegram, scripts de operacion no criticos.
   Nada auxiliar se amplia mientras 1 o 2 tengan huecos.

Regla del revisor: features nuevas de capa 3 (como las de la noche del
10-11: stdio, ingesta CSV/JSON, presentaciones) quedan CONGELADAS en cuanto
cierres la limpieza actual, hasta que 1 y 2 esten terminadas. El revisor
comprobara en cada despertar que los commits respetan el orden.

## Protocolo de despertar al revisor (vigente desde 2026-07-11 ~01:30)

Cuando TERMINES una tarea de esta cola (o un hito de la directiva
dentro->fuera), avisa al revisor escribiendo una linea de resumen en el
fichero senal (el revisor lo consume y se despierta en segundos):

    echo "R1 cerrado en <commit>: <resumen de una linea>" > /home/alberto/Trabajo/orquesta/.orquesta-revisor-wake

- Una senal por hito terminado (no por commit intermedio).
- Tambien usa la senal si te BLOQUEAS y no puedes avanzar (di en la linea
  "BLOQUEADO: <causa>"): el revisor lo arregla y te deja instrucciones aqui.
- El fichero senal NO se commitea (esta fuera del contrato del repo).
- Si no hay senal, el revisor tiene ademas un despertar fijo de respaldo
  (~60 min) por si te quedas colgado sin poder avisar.

## Plan final (cuando la directiva dentro->fuera este completa)

Al terminar nucleo + conectores + auxiliares, el cierre del ciclo es que
ORQUESTA cree una app REAL de verdad (no smoke): el operador quiere, por
ejemplo, un modulo nuevo para Orquesta o una tool nueva, creado por
Orquesta via su API nativa con goal acotado y revisor validando cierre.
Candidatos se decidiran con el operador; no lanzar nada sin su visto bueno.
