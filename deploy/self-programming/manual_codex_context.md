# Contexto para Codex manual remoto en Orquesta

Eres un Codex manual arrancado por el operador en el servidor remoto
`berserk@uso.dipgra.cloud`, dentro del contenedor aislado
`orquesta-self-programming`.

## Objetivo

Continuar el trabajo de Orquesta para que llegue a autoprogramarse con Goal,
tmux y evidencias, y cerrar sus frentes principales: nucleo, web Nueva App,
autoprogramacion, runtime Codex Goal, OPES/DomainWork en aislado y
documentacion.

## Seguridad obligatoria

- Trabaja solo en `/workspace/project`.
- Estado, runtime, cache y artefactos de prueba viven bajo `/workspace` o
  `/srv/orquesta-self` visto desde el host.
- No toques `uso-app`, `opes-api`, postgres, nginx, contenedores productivos ni
  `/home/berserk/deploy/opes`.
- No montes `/var/run/docker.sock` dentro del contenedor.
- No abras puertos externos. Orquesta solo publica `127.0.0.1:19039` en el host.
- No uses root dentro del contenedor. El usuario esperado es `10001:10001`.
- Si haces operaciones Docker en el host, usa `sudo` solo para el contenedor
  aislado `orquesta-self-programming`.
- No copies credenciales personales. GitHub remoto no tiene acceso directo
  salvo que el operador instale una deploy key limitada.
- No borres artefactos de pruebas de temario o conectores. Se conservan para
  revision y posible promocion manual.

## Runtime

- Para Orquesta goal-first, el backend valido es `app_server_tmux`.
- No uses `stdio` para Goal.
- No uses `app_server_proxy`.
- El loop legacy solo se toca en pruebas legacy aisladas y explicitas.
- Si falta auth, cuota, tmux, socket o GitHub, documenta bloqueo accionable.

## Estado remoto conocido

- Repo remoto aislado: `/srv/orquesta-self/worktrees/orquesta` en host,
  `/workspace/project` dentro del contenedor.
- Contenedor: `orquesta-self-programming`.
- Orquesta responde por tunel/loopback en `127.0.0.1:19039`.
- La imagen incluye `codex`, `tmux`, `go`, `git`, `jq`, `python3`.
- GitHub directo desde `berserk` no esta configurado: usar bundle o pedir
  deploy key limitada al repo `aavidad/orquestador`.

## Primeras comprobaciones

```bash
pwd
git status --short
git log --oneline -5
id
codex --version
tmux -V
go version
curl -fsS http://127.0.0.1:19039/api/v0/server/status | jq '.status,.startup_status'
```

## Frentes que hay que cerrar

1. Perfil remoto aislado:
   verificar usuario no-root, `cap_drop=ALL`, `no-new-privileges`, binds solo
   `/srv/orquesta-self`, puerto solo `127.0.0.1:19039`.
2. Self-programming:
   corregir cualquier bloqueo que impida que Orquesta lance automejora con
   Goal por `app_server_tmux`. No aceptar regresion a `stdio`.
3. Reparacion automatica:
   si una prueba o smoke falla, Orquesta debe convertirlo en tarea causal,
   relanzar con contexto compacto y dejar evidencia.
4. Web Nueva App:
   tooltips en todas las opciones, validacion en castellano, wizard
   conversacional, modo experto de datos/integraciones, arquitecturas amplias,
   hexagonal por defecto cuando encaje, calidad/accesibilidad seleccionable en
   ejecucion y manual profundo.
5. OPES/DomainWork:
   solo fakes, temporales o rutas aisladas bajo `/srv/orquesta-self`; no OPES
   productivo. Si se prueba temario, conservar artefactos y no promocionar.
6. Documentacion:
   actualizar runbooks, matriz, estado actual y arquitectura con evidencia real.

## Forma de trabajo

- Lee `AGENTS.md` y los docs vigentes que cite antes de cambios transversales.
- Mantén write-set estrecho.
- Usa `rg` para explorar.
- Ejecuta pruebas focales antes de cerrar cambios.
- Para cambios transversales intenta:
  `git diff --check` y `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-runtime-codex-goal ./modulos/orquesta-goal`.
- Commit local claro. Push solo si el operador ha instalado credencial GitHub
  limitada y lo ha autorizado.
- Responde al operador en castellano, con estado, pruebas, bloqueos y siguiente
  accion.

