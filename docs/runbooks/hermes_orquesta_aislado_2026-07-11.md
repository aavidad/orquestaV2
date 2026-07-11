# Hermes 0.18.2 aislado con Orquesta

Runbook L2 para una instalación local reproducible. El contenedor se ejecuta
desde `.orquesta-runtime/hermes`; el repositorio es el único montaje visible.
No pongas credenciales, tokens, cookies ni valores sensibles en este documento
ni en el entorno del contenedor.

## Preparación y pin de imagen

Usa únicamente la imagen oficial `nousresearch/hermes-agent`, release Hermes
0.18.2 (`v2026.7.7.2`). Antes del primer arranque, resuelve el digest del
registro oficial y guárdalo en la variable local `HERMES_IMAGE`; nunca uses
`latest`, `main` ni un tag sin digest:

```sh
mkdir -p .orquesta-runtime/hermes/{home,tmp,run}
docker buildx imagetools inspect nousresearch/hermes-agent:v2026.7.7.2
# Copia el digest de MANIFEST para la plataforma usada:
export HERMES_IMAGE='nousresearch/hermes-agent:v2026.7.7.2@sha256:<digest-manifest-verificado>'
case "$HERMES_IMAGE" in *@sha256:[0-9a-fA-F][0-9a-fA-F]*) ;; *) echo 'digest requerido' >&2; exit 2;; esac
```

El operador debe conservar el digest completo obtenido del registro como parte
de su instalación local; el runbook no contiene valores secretos ni pretende
inventar un digest. Repite `imagetools inspect` y compara el digest antes de
cada actualización.

## Arranque restringido

`REPO` es la raíz del checkout. El socket Unix es el único canal MCP: el
servidor MCP de Orquesta debe publicarlo en
`.orquesta-runtime/hermes/run/orquesta-mcp.sock` con permisos de usuario.

```sh
REPO=$(pwd)
RUNTIME="$REPO/.orquesta-runtime/hermes"
test -S "$RUNTIME/run/orquesta-mcp.sock"
docker rm -f hermes-orquesta 2>/dev/null || true
docker run -d --name hermes-orquesta \
  --user 1000:1000 --read-only --cap-drop=ALL --security-opt=no-new-privileges:true \
  --network hermes-egress \
  --mount type=bind,src="$REPO",dst=/workspace,readonly \
  --mount type=bind,src="$RUNTIME/home",dst=/hermes-home \
  --mount type=bind,src="$RUNTIME/tmp",dst=/tmp \
  --mount type=bind,src="$RUNTIME/run",dst=/run/hermes \
  -e HERMES_HOME=/hermes-home -e TMPDIR=/tmp \
  -e ORQUESTA_MCP_SOCKET=/run/hermes/orquesta-mcp.sock \
  "$HERMES_IMAGE" gateway run
```

`hermes-egress` debe ser una red preexistente del runtime que permita Internet
público y bloquee RFC1918, loopback del host, metadata y redes internas. El MCP
por UDS funciona porque el socket está montado; no se expone Docker socket,
`HOME` del host, puertos ni redes privadas. No añadas un segundo montaje ni un
socket TCP. Si el runtime no puede expresar esta política, bloquea el arranque
y corrige el adaptador de red, no la restricción del contenedor.

## Comprobaciones

```sh
docker exec hermes-orquesta hermes --version
docker inspect hermes-orquesta --format '{{.Config.User}} {{.HostConfig.ReadonlyRootfs}}'
docker inspect hermes-orquesta --format '{{json .HostConfig.CapDrop}}'
docker exec hermes-orquesta sh -lc 'test "$HERMES_HOME" = /hermes-home && test "$TMPDIR" = /tmp'
test ! -S /var/run/docker.sock
docker exec hermes-orquesta hermes status
docker exec hermes-orquesta hermes smoke
```

El smoke debe probar MCP Orquesta por el socket Unix, escritura solo en
`HERMES_HOME`/`TMPDIR`, lectura del repositorio y rechazo de un destino privado.
Debe fallar si aparece un montaje adicional, `HOME`, una capacidad, rootfs
escribible, una red privada o el Docker socket. Guarda únicamente el resultado
resumido y la referencia de ejecución; nunca logs con configuración sensible.

## Operación y reparación

La política es **Orquesta primero**: plan, contexto, write-set, tests y cierre
se gobiernan por Orquesta; Hermes ejecuta el trabajo dentro de ese alcance.
No se reanima el loop histórico ni se amplía el write-set desde el contenedor.

Tras un fallo, conserva el runtime y el diagnóstico compacto. Reintenta por
Orquesta; solo después de un bloqueo repetido y documentado se permite una
reparación directa acotada (pin, socket, permisos, mounts o política de red).
La reparación no puede introducir secretos, acceso al host, capacidades,
montajes nuevos ni saltarse tests. Vuelve a ejecutar version/status/smoke y
retorna el control a Orquesta.

## Limpieza

```sh
docker rm -f hermes-orquesta
rm -rf .orquesta-runtime/hermes/{home,tmp,run}
```

Antes de limpiar, verifica que no quedan procesos o sesiones que usen el
socket; conserva evidencia de fallos y elimina solo este runtime aislado.
