# Codex En Docker Limitado Al Proyecto

Este entorno ejecuta `codex` dentro de un contenedor y solo monta esta carpeta en `/workspace`.

## Garantías

- `codex` puede leer y escribir únicamente en esta carpeta montada.
- No puede borrar nada fuera del repositorio porque no hay más rutas del host montadas.
- El estado y la autenticación de `codex` se guardan en `./.codex-docker-home/`.
- La configuración activa dentro del contenedor es:

```toml
approval_policy = "never"
sandbox_mode = "workspace-write"

[sandbox_workspace_write]
network_access = true
```

## Uso

Primera ejecución o ejecución normal:

```bash
./scripts/run_codex_docker.sh
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

- El script copia `~/.codex/auth.json` a `./.codex-docker-home/auth.json` si existe y aún no se ha copiado.
- Si quieres aislamiento total de red más adelante:
  - cambia `network_access = false` en `./.codex-docker-home/config.toml`
  - y añade `network_mode: "none"` al servicio `codex-sandbox`
