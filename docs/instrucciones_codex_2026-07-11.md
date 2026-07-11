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

## Revision ~02:00: R1+R2 ACEPTADOS por el revisor

Guard raiz reejecutado en verde en es_ES y en C; presupuestos sin subir.
Buen trabajo. Siguiente: R4b (documentar frontera stdio) es rapido; luego
R4 (dos pases con receipt para cerrar D3+208H formalmente) y despues
continua la directiva dentro->fuera por el NUCLEO. Senala al terminar R4.

## Revision ~04:00: NUCLEO ACEPTADO por el revisor - fase CONECTORES abierta

Verificado por el revisor tras 24f855618 (53 commits revisados en rango):
las 7 suites del nucleo (estado-vivo, goal, orchestration-core,
app-director-service, run-control, run-queue, autoprogramming) + mcp +
stack + guard raiz de envs: TODO verde reejecutado. Evidencia del E2E
autonomo (autoprog-attestor-e2e-autonomous-20260711: complete/accepted,
2 receipts independientes, shutdown limpio) verificada en la incidencia.
Hito historico registrado. Residuales aceptados como trabajo de la fase
conectores: external-work (clasificado conector), scheduler concurrente de
atestaciones. VIA LIBRE a CONECTORES en el orden de la directiva:
runtime-codex-* -> state-file -> required-test -> superficies mcp/http ->
wiring stack/cmd. Mismas reglas: senal por hito, guards antes de cerrar,
auxiliares siguen congelados. T263 no se toca (confirmado).

## Revision ~02:10: R4 ACEPTADO - 208H y D3 quedan ACREDITADOS

Receipt de dos pases verificado (7 pkgs x 2, exit 0). Cola R1-R4 completa.
Siguiente frente: la directiva dentro->fuera empezando por el NUCLEO
(recuerda R4b si aun no esta). Senala hitos de nucleo terminados.

## Hallazgos de la revision 2026-07-11 ~01:00 (orden de prioridad)

- [x] R1 (cerrado `cec2848f3`): `scripts/orquesta_metricas_deuda.sh` era fragil a
  locale: usa `sort`/`comm` sin fijar `LC_ALL=C` y en un entorno es_ES
  casca con "comm: archivo 2 no esta en orden ordenado", tirando el guard
  raiz `TestEnvVarsBudgetMEJ106V0`. Reproduccion verificada por el revisor:
  `bash scripts/orquesta_metricas_deuda.sh --json` falla con locale es_ES y
  funciona con `LC_ALL=C`. Fix: fijar `LC_ALL=C` (export al inicio del
  script) y anadir a `scripts/test_orquesta_metricas_deuda.sh` un caso que
  lo ejecute con un locale no-C para que no regrese. El script exporta
  `LC_ALL=C` y la prueba usa shims portables para no depender de `es_ES`.
- [x] R2 (cerrado `31ecba309` + `0b564992a`): la medicion con locale C era produccion
  427/425 y test-only 106/103. Sin elevar los presupuestos, se retiraron dos
  overrides de timeout duplicados del guardian y tres nombres de IPC de
  fixtures fuera del prefijo global. Criterio revalidado:
  `go test -count=1 -run 'TestEnvVarsBudgetMEJ106V0' .` verde en locale C y
  bajo locale heredado no-C; la metrica queda 423/425 y 103/103. El segundo
  commit elimina tambien los aliases sin unidad: esos timeouts viven solo en
  `orquesta.config.json` tipado y el helper comun se separa para respetar T90.
- [ ] R3 (disciplina): antes de cerrar cada sesion de trabajo, reejecuta el
  guard raiz de budget y los focales de lo tocado. Esta noche dejaste tu
  propio guard rojo sin saberlo; la regla "no verde autodeclarado" tambien
  aplica a guards que tu mismo escribiste.
- [x] R4b (cerrado documentalmente 2026-07-11): `mcp-stdio` es SOLO
  transporte de cliente MCP (JSON-RPC por stdin/stdout hacia las tools).
  PROHIBIDO usar stdio para pilotar/observar agentes o backends de goals:
  eso sigue siendo tmux (`app_server_tmux`) con identidad y evidencias.
  La frontera queda documentada en
  `docs/runbooks/mcp_stdio_orquesta_2026-07-11.md`; no ampliar esta superficie
  hasta terminar nucleo y conectores.
- [x] R4 (cerrado localmente 2026-07-11): dos pases aislados sobre siete
  paquetes, 14 ejecuciones y receipt
  `/tmp/orquesta-r4-retry-batches/receipt.json` con
  `two_consecutive_passes_passed`. D3 y 208H quedan cerrados localmente; el
  transitorio tmux 208Y no se reprodujo en 20 focales ni en el lote acreditado.

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
