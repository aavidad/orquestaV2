# Hermes Orquesta aislado: instalación local vigente

Runbook L2 para la instalación local actual. La entrada canónica es
`.orquesta-runtime/hermes/hermes-orquesta.sh`; no sustituyas este flujo por
otra imagen, tag, comando de Hermes o contenedor persistente. No escribas
credenciales, tokens, cookies ni valores sensibles en este documento.

## Arranque

Ejecuta desde la raíz del checkout:

```sh
./.orquesta-runtime/hermes/hermes-orquesta.sh
```

El script usa la imagen derivada local `orquesta-hermes:0.18.2`, fijada al
digest oficial que ya forma parte de la instalación. No uses `latest`, una
imagen upstream sin derivar ni un tag sin digest. El script prepara y retira
un contenedor efímero por sesión; no reutilices el nombre o el contenedor de
una sesión anterior.

El checkout se expone dentro del contenedor en `/workspace` con escritura
permitida. Esa es la única zona de trabajo editable del agente: no confundas
la ruta del host con una ruta editable desde fuera del contenedor.

El estado escribible de Hermes queda bajo el runtime aislado:

- `HERMES_HOME` apunta a `.orquesta-runtime/hermes/home`.
- `TMPDIR` apunta a `.orquesta-runtime/hermes/tmp`.
- El helper `host-orquesta-socket.sh` prepara el puente local.
- La API de Orquesta se publica como `.orquesta-runtime/hermes/orquesta-api.sock`.

No crees otro socket, montaje o variable equivalente. El socket API es el
canal local para la integración; no se publica un puerto TCP.

## TUI y operación

La sesión interactiva/TUI se abre con la entrada canónica:

```sh
./.orquesta-runtime/hermes/hermes-orquesta.sh
```

Si la sesión ya está abierta, vuelve a usar el script para reconectar a su
socket; no arranques Hermes directamente con Docker. Las acciones de trabajo
se mantienen dentro del write-set que Orquesta entregó al goal y cualquier
fallo se devuelve a Orquesta para rework.

## Aislamiento efectivo

La instalación arranca el proceso como UID 1000, sin capacidades efectivas,
con rootfs de solo lectura y sin privilegios nuevos. No hereda `HOME` del host,
no monta el Docker socket y no recibe acceso al filesystem interno del host.
La única escritura autorizada es `/workspace` y el estado bajo `HERMES_HOME` y
`TMPDIR` del runtime.

La red permite Internet público para el trabajo autorizado, pero bloquea
loopback del host, metadata, RFC1918 y las demás redes privadas. Si el helper
no puede aplicar simultáneamente esa política y el socket API, detén la
sesión: no relajes el aislamiento ni inventes una variante de red.

## Comprobación y cierre

Antes de aceptar la sesión, verifica mediante el flujo del script que el TUI
responde, que Orquesta es accesible por `orquesta-api.sock`, que el checkout
se puede leer y modificar dentro de `/workspace`, y que `HERMES_HOME` y
`TMPDIR` quedan dentro de `.orquesta-runtime/hermes`. Comprueba también que no
hay Docker socket, `HOME` heredado, capacidades efectivas, rootfs escribible
ni acceso a destinos privados.

Al terminar, la sesión debe desaparecer como contenedor efímero y conservar
solo el estado necesario bajo `.orquesta-runtime/hermes`. No borres ese estado
si Orquesta lo necesita para evidencia o rework; limpia únicamente después de
confirmar que no quedan procesos usando `orquesta-api.sock`.
