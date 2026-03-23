# Orquesta: Manual del Técnico de Sistemas (DevOps)

Guía de despliegue, mantenimiento y configuración de infraestructura para Orquesta.

## 1. Requisitos del Sistema
- **Lenguaje:** Go 1.21+ para la compilación del binario.
- **Base de Datos:** SQLite 3 (configurado en modo WAL para concurrencia).
- **Entorno:** Linux (Ubuntu 22.04+ recomendado).

## 2. Despliegue (Dockerización OP-085)
Para desplegar Orquesta en un contenedor:
```bash
docker build -t orquesta:v1 .
docker run -d -p 3000:3000 -v $(pwd)/data:/app/data orquesta:v1
```
*Asegúrate de mapear el volumen para que `orquesta.db` persista.*

## 3. Configuración del Servidor MCP
El servidor MCP se activa mediante variables de entorno:
- `ORQUESTA_MCP_ENABLED=true`
- `ORQUESTA_MCP_PORT=3000`
- `ORQUESTA_AUTH_TOKEN`: Token para clientes externos.

## 4. Mantenimiento de la Refinería (OP-093)
El agente `Refinery` requiere acceso de lectura/escritura al socket de Docker y a los repositorios Git locales.
- **Logs:** Revisa `/var/log/orquesta/refinery.log` para depurar fallos en la cola de merge.
- **Git Hooks:** Orquesta instala hooks automáticos para la validación previa de OPs.

## 5. Troubleshooting (Solución de Problemas)
- **Base de Datos Bloqueada:** Si recibes un error `database is locked`, comprueba que no haya procesos `go run` colgados. Usa `pkill -9 orquesta`.
- **Timeout de API:** Si la CLI tarda en responder, es probable que esté intentando conectar con un servidor API configurado incorrectamente antes de caer al modo local de DB.

---
*Manual de Operaciones v1.0*
