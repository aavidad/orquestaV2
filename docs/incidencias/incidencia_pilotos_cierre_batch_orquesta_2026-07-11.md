# Incidencia: pilotos de cierre batch Orquesta

Fecha: 2026-07-11  
Alcance: `BUG-ORQ-20260711-255` a `BUG-ORQ-20260711-268`
Estado: en reparacion; no declarar autonomia completa hasta replay final

## Objetivo del piloto

Ejecutar por `POST /api/v0/autoprogramming/prepare-run` dos tareas reales de
limpieza con write-sets disjuntos, worktrees fisicos, Codex real, atestacion
independiente, integracion Git, gate unico y replay idempotente. No se uso una
app desechable ni se toco OPES/remoto.

## Resultado acumulado

- replay3 probo CWD fisico correcto y checkout canonico limpio, pero encontro
  watcher fuera del receipt dir, rename+edicion no reconocido y test bloqueado
  por sandbox del implementador;
- replay4 observo ambos receipts automaticamente; g02 cerro accepted con
  atestacion independiente real, mientras g01 fallo porque el runner hermetico
  convertia HOME ausente en `.codex` relativo;
- replay5 probo `go test ./cmd/orquesta-server` verde en entorno hermetico tras
  `67b475fc3`, pero expuso que el receipt se consume antes de `task_complete`;
  el snapshot prematuro cambio, abrio rework y ese rework quedo solo en marker;
- replay6 probo el shutdown corregido: la API encontro el backend de
  autoprogramacion y su marker cuarentenado, devolvio `shutdown_ready=true` y
  termino servidor y tmux en dos segundos. El batch no pudo cerrar porque el
  sandbox del worktree permite escribir el checkout pero no su admin dir Git
  externo; `git mv` fallo al crear `index.lock`;
- replay7 confirmo por el perfil efectivo de Codex que la raiz Git adicional
  se declara pero sigue protegida. El agente completo el rename en filesystem;
  el segundo goal completo su cambio y focal, pero quedo blocked porque el
  alias `external_test_environment_restriction` no activo al atestador para el
  test global dependiente. El checkout canonico permanecio limpio y el shutdown
  volvio a cerrar servidor/backend en dos segundos.
- replay8 probo la propiedad VCS corregida y cerro g02 `complete` con su test
  global delegado. G01 completo rename y checks de filesystem, pero uso
  `sandbox_unix_socket_operation_not_permitted`; al faltar aun en el catalogo
  tipado, consumio tres continuaciones y cerro blocked. El canonico siguio
  limpio y el shutdown termino en el primer sondeo.
- replay9 normalizo el alias y ejecuto atestacion independiente para g01. Dos
  checks pasaron, pero el test Go fallo: el snapshot de modulos configurado no
  incluia `golang.org/x/text@v0.38.0`; diez tests anidados con `GOPROXY=off`
  expusieron que el preflight no validaba cierre completo de dependencias. G02
  cerro accepted, g01 quedo con cierre bloqueado y no hubo integracion.
- replay10 supero el preflight con un snapshot completo de 52 MB, pero el
  comando `go mod download all` anadio checksums al `go.sum` canonico. Se paro
  antes de integrar y se restauro solo ese diff generado. Tres shutdown forced
  quedaron `stop_pending` mientras tests/turns seguian activos; SIGINT del PID
  propio cerro servidor y tmux. El preflight queda reemplazado por dos `go list`
  readonly y el shutdown activo se registra como BUG-267.
- replay11 supero el preflight readonly sin ensuciar `go.sum`. G01 termino el
  cambio, pero su atestacion volvio a fallar en diez tests anidados; G02 termino
  su focal y quedo blocked por el test dependiente de `cmd/orquesta-server`.
  Ambos cierres quedaron correctamente en rework y el batch en `goals_running`:
  no era un falso atasco del coordinador porque ninguna run habia cerrado. La
  causa comun fue `TestMain`, que reemplazaba el `GOMODCACHE` privado por uno
  vacio y borraba el seed. Tras corregirlo, la misma suite hermetica pasa
  completa. El shutdown terminal del replay devolvio `ready` y cerro proceso y
  backend por API en el primer intento valido.
- los shutdown de replay3/4/5 quedaron `stop_pending` con contadores cero y un
  tmux propio vivo; el fallback acotado uso SIGINT del PID del piloto y elimino
  exclusivamente su sesion `orquesta-goal-*` tras varios intentos HTTP.

## Evidencia durable no versionada

- `/tmp/orquesta-live-bug255-replay3-20260711`
- `/tmp/orquesta-live-bug255-replay4-20260711`
- `/tmp/orquesta-live-bug255-replay5-20260711`
- `/tmp/orquesta-live-bug255-replay6-20260711`
- `/tmp/orquesta-live-bug255-replay7-20260711`
- `/tmp/orquesta-live-bug255-replay8-20260711`
- `/tmp/orquesta-live-bug255-replay9-20260711`
- `/tmp/orquesta-live-bug255-replay10-20260711`
- `/tmp/orquesta-live-bug255-replay11-20260711`

Se retienen hasta extraer el recibo final. No contienen autoridad documental y
se eliminaran de forma gobernada al cerrar la incidencia. No versionar
transcripts, CODEX_HOME, sockets ni caches.

## Commits ya cerrados

- `61b3940a1`: goals batch usan su workspace fisico;
- `776c952f4`: observer resuelve el root durable del goal;
- `d57f7daf8`: provision y launcher reutilizan la identidad worktree tipada;
- `19d2ef78b`: watcher incluye el receipt dir causal;
- `f8fc97ba4`: rename tipado admite contenido modificado con destino exacto;
- `b6ed60d0d`: test no ejecutable por sandbox puede delegarse al atestador sin
  saltarse el guard de write-set;
- `67b475fc3`: HOME ausente no produce CodeHomeDir relativo.
- `2cdec492e`: shutdown incluye el backend de autoprogramacion;
- `f945b2981`: cierre espera terminacion real y conserva residuos verificables;
- `b012a88f9`: rework usa un sucesor causal persistido.

## Criterio de cierre restante

1. No consumir un receipt terminal antes de que el provider turn termine.
2. Persistir y observar rework con una sola generacion causal tras restart.
3. Mantener indice/staging/commit bajo propiedad de Orquesta; el agente solo
   modifica filesystem, incluido rename tipado.
4. Normalizar el alias estructurado de entorno de tests y delegar el test
   bloqueado al atestador independiente.
5. Validar en preflight que el snapshot Go resuelve `go mod download all` sin
   red y conservarlo en todos los tests anidados sin mutar el modulo.
6. Repetir el batch desde estado limpio: dos accepted, dos commits encadenados,
   gate unico, checkout canonico limpio y batch closed.
7. Reenviar exactamente la request: cero threads, commits y gates nuevos.
8. Suite amplia verde y documentacion/inventario actualizados con commits de
   cierre. Hasta entonces `BUG-255`, `259`, `260` y `263` siguen abiertos;
   `BUG-261` y `BUG-262` quedan cerrados por replay6.
