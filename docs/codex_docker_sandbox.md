# Codex En Docker Limitado Al Proyecto

Este entorno ejecuta `codex` dentro de un contenedor, pero no monta el repo real como directorio de trabajo. Primero crea una copia persistente del proyecto en `./.codex-sandbox-workspace/`, y esa copia es la que se monta en `/workspace`.

## Garantías

- `codex` trabaja sobre `./.codex-sandbox-workspace/`, no sobre el árbol real del repo.
- No puede borrar nada fuera de esta carpeta porque no hay más rutas del host montadas.
- El repo real en la raíz del proyecto queda intacto salvo que luego tú sincronices cambios manualmente.
- El estado y la autenticación de `codex` se guardan en `./.codex-docker-home/`.
- La configuración activa dentro del contenedor es:

```toml
approval_policy = "never"
sandbox_mode = "danger-full-access"
```

- La red queda habilitada; no hay aislamiento de red en este contenedor.

## Uso

Primera ejecución o ejecución normal:

```bash
./scripts/run_codex_docker.sh
```

Antes de arrancar, el script sincroniza al sandbox local del proyecto el estado útil de `~/.codex/`:

- `auth.json`
- `history.jsonl`
- `state_5.sqlite*`
- `logs_1.sqlite*`
- `sessions/`
- `memories/`
- `rules/`
- `skills/`
- `shell_snapshots/`
- `log/`

Eso permite reentrar en Docker con la misma base persistida de sesiones y contexto de `codex`.

Además:

- en el primer arranque crea `./.codex-sandbox-workspace/` como copia persistente del repo
- en arranques posteriores reutiliza esa copia y no la machaca
- si quieres refrescarla desde el repo real, usa:

```bash
ORQUESTA_REFRESH_WORKSPACE=1 ./scripts/run_codex_docker.sh
```

Pasar argumentos a `codex`:

```bash
./scripts/run_codex_docker.sh --help
```

Abrir una shell dentro del mismo contenedor:

```bash
LOCAL_UID="$(id -u)" LOCAL_GID="$(id -g)" \
docker compose -f docker-compose.codex.yml run --rm codex-sandbox bash
```

## Notas

- El script sincroniza el `config.toml` plantilla a `./.codex-docker-home/config.toml` en cada arranque y deja copia previa en `config.toml.bak` si hubo cambios.
- El script copia `~/.codex/auth.json` a `./.codex-docker-home/auth.json` si existe y aún no se ha copiado.
- El workspace real dentro del contenedor es `./.codex-sandbox-workspace/`, montado como `/workspace`.
- Si no quieres sincronizar el estado del host en una ejecución concreta:

```bash
ORQUESTA_SYNC_HOST_CODEX=0 ./scripts/run_codex_docker.sh
```

- Si quieres aislamiento total de red más adelante:
  - cambia `network_access = false` en `./.codex-docker-home/config.toml`
  - y añade `network_mode: "none"` al servicio `codex-sandbox`
