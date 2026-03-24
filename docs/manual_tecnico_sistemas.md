# Orquesta: Manual del Técnico de Sistemas (DevOps)

Guía de despliegue, mantenimiento y configuración de infraestructura para Orquesta.

## 1. Requisitos del Sistema
- **Lenguaje:** Go 1.21+ para la compilación del binario.
- **Base de Datos:** SQLite 3 (configurado en modo WAL para concurrencia).
- **Entorno:** Linux (Ubuntu 22.04+ recomendado).

## 2. Despliegue (Docker/Compose)
El despliegue recomendado es `docker compose` desde la raíz del repositorio.

```bash
docker compose up -d --build
```

Parámetros útiles:

- `ORQUESTA_HTTP_PORT`: puerto publicado hacia el host. Por defecto `16543`.
- `ORQUESTA_WORKSPACE_DIR`: ruta del workspace host que se monta en `/app/workspace`. Por defecto `..` respecto al repositorio de Orquesta.

El servicio publica:

- panel web y API en `http://localhost:16543`
- base SQLite persistida en el volumen `orquesta-data`
- logs persistidos en el volumen `orquesta-logs`

Comandos operativos básicos:

```bash
docker compose ps
docker compose logs -f orquesta
docker compose down
```

Si se necesita imagen suelta sin Compose:

```bash
docker build -t orquesta:local .
docker run --rm -p 16543:16543 \
  -e ORQUESTA_DB=/app/data/orquesta.db \
  -e ORQUESTA_WORKSPACE_ROOT=/app/workspace \
  -v "$(pwd)/data:/app/data" \
  -v "$(dirname "$(pwd)"):/app/workspace" \
  orquesta:local
```

## 3. Configuración del Servidor MCP
El servidor MCP se activa mediante variables de entorno:
- `ORQUESTA_MCP_ENABLED=true`
- `ORQUESTA_MCP_PORT=16543`
- `ORQUESTA_AUTH_TOKEN`: Token para clientes externos.

## 4. Mantenimiento de la Refinería (OP-093)
El agente `Refinery` requiere acceso de lectura/escritura al socket de Docker y a los repositorios Git locales.
- **Logs:** Revisa `/var/log/orquesta/refinery.log` para depurar fallos en la cola de merge.
- **Git Hooks:** Orquesta instala hooks automáticos para la validación previa de OPs.

## 5. Troubleshooting (Solución de Problemas)
- **Base de Datos Bloqueada:** Si recibes un error `database is locked`, comprueba que no haya procesos `go run` colgados. Usa `pkill -9 orquesta`.
- **Timeout de API:** Si la CLI tarda en responder, es probable que esté intentando conectar con un servidor API configurado incorrectamente antes de caer al modo local de DB.
- **Workspace no visible en contenedor:** verifica `docker compose config` y que `ORQUESTA_WORKSPACE_DIR` apunta al directorio host donde viven los proyectos gestionados por Orquesta.

---
*Manual de Operaciones v1.0*
