# Orquesta Self-Programming Aislado

Este despliegue sirve para que Orquesta se autoprograme en un contenedor de
pruebas. No despliega OPES, no monta `uso-app`, no monta temarios, no monta
`/var/run/docker.sock` y no publica puertos externos.

## Frontera

- Permitido: modificar el repo Orquesta dentro de `/workspace/project`, generar
  ramas, commits locales, patches, logs, estado y evidencias de prueba.
- Permitido: tocar codigo de conectores OPES, web o programacion como parte del
  repo Orquesta.
- Prohibido en este entorno: subir a produccion, escribir en `uso-app`, escribir
  en `/home/berserk/deploy/opes`, drenar colas OPES productivas o abrir puertos
  publicos.
- Acceso web: solo `127.0.0.1:19039` en el host remoto. Usar tunel SSH.

## Preparacion remota

`sudo` se limita a preparar directorios aislados en el host bajo
`/srv/orquesta-self` y a operar el contenedor `orquesta-self-programming` cuando
sea imprescindible. Dentro del contenedor el usuario esperado es `10001:10001`;
no se trabaja como root.

```bash
sudo install -d -o 10001 -g 10001 /srv/orquesta-self/state
sudo install -d -o 10001 -g 10001 /srv/orquesta-self/runtime
sudo install -d -o 10001 -g 10001 /srv/orquesta-self/worktrees
sudo install -d -o 10001 -g 10001 /srv/orquesta-self/home
sudo install -d -o 10001 -g 10001 /srv/orquesta-self/cache/go
sudo install -d -o 10001 -g 10001 /srv/orquesta-self/cache/gomod
sudo install -d -o 10001 -g 10001 /srv/orquesta-self/codex-home
```

Clonar o actualizar solo el repo Orquesta en
`/srv/orquesta-self/worktrees/orquesta`. No montar rutas de OPES ni de servicios
existentes.

Copiar `orquesta-self.env.example` a `orquesta-self.env` y cambiar solo el
token de control. El `CODEX_HOME` del contenedor debe recibir credenciales
Codex aisladas en `/srv/orquesta-self/codex-home`, montadas como
`/workspace/codex-home`; `HOME`, cache Go y mod cache quedan tambien bajo
`/workspace`. No se copian credenciales de produccion ni de otros servicios.

## Arranque

```bash
docker build -f Dockerfile.self-programming \
  --build-arg ORQUESTA_BUILD_COMMIT="$(git rev-parse HEAD)" \
  -t orquesta-self-programming:local .
docker compose -f deploy/self-programming/docker-compose.yml up -d
```

El argumento de build es obligatorio porque el contexto Docker excluye `.git`.
El Dockerfile acepta solo un SHA Git hexadecimal completo de 40 caracteres y
lo conserva como identidad VCS del binario; asi el guard puede contrastar la
imagen con el `HEAD` exacto montado en `/workspace/project`.

El compose fija `ORQUESTA_CODEX_COMMAND=/usr/local/bin/codex`, ruta absoluta
del CLI instalado en la imagen. No depende del `PATH` ni añade credenciales.

## Contrato verificable

El contrato local de seguridad del perfil aislado se valida con:

```bash
go test -count=1 ./deploy/self-programming
```

La prueba debe fallar si el compose publica puertos fuera de
`127.0.0.1:19039`, monta rutas fuera de `/srv/orquesta-self`, monta
`/var/run/docker.sock`, pierde `no-new-privileges`, `seccomp=unconfined`, `cap_drop: ALL`,
`read_only: true` o usuario `10001:10001`, o si el env de ejemplo activa
produccion, OPES/DomainWork, promocion automatica o backends historicos/proxy.

`seccomp=unconfined` se limita a este perfil aislado para que el sandbox
`workspace-write` de Codex pueda crear su user namespace con bubblewrap. No
anade capacidades Linux: se mantienen usuario no-root, `no-new-privileges`,
`cap_drop: ALL`, rootfs de solo lectura, sin Docker socket y binds exclusivos
de `/srv/orquesta-self`.

## Quien dirige

La direccion remota la hace `orquesta-server` dentro del contenedor. El operador
no debe arrancar un Codex manual por SSH para trabajar sobre el repo salvo
diagnostico puntual.

Cuando Orquesta recibe un trabajo por web/API, o cuando entra el modo
`idle self-improvement`, compila un `GoalWorkSpec` con objetivo, reglas,
contexto, write-set, tests y artefactos. Ese contrato se ejecuta con Codex Goal
por `app_server_tmux` dentro del contenedor. Codex es el ejecutor; Orquesta
mantiene el scope, evidencias, validacion y cierre.

Si falta autenticacion de Codex, cuota, tmux o backend Goal, el entorno debe
bloquear y documentar el problema. No debe caer a `stdio`, `app_server_proxy` ni
al loop legacy.

Acceso desde tu equipo:

```bash
ssh -L 19039:127.0.0.1:19039 berserk@uso.dipgra.cloud
```

Despues abre `http://127.0.0.1:19039`.

## Checks de seguridad

```bash
docker inspect orquesta-self-programming \
  --format '{{json .HostConfig.Binds}} {{json .NetworkSettings.Ports}} {{json .HostConfig.ReadonlyRootfs}} {{json .HostConfig.SecurityOpt}} {{json .HostConfig.CapDrop}}'
```

Debe verse solo `/srv/orquesta-self/...` y el puerto publicado como
`127.0.0.1:19039`. No debe aparecer `/home/berserk/deploy/opes`,
`/var/run/docker.sock`, `uso-app` ni rutas de temarios. `ReadonlyRootfs` debe
ser `true`, `SecurityOpt` debe incluir `no-new-privileges:true` y
`seccomp=unconfined`, y `CapDrop` debe incluir `ALL`.
