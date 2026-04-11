# Arquitectura de Resolución de Modelos y Gestión de Flota

Este documento detalla los cambios realizados para garantizar que la plataforma Orquesta utilice modelos locales adecuados (Qwen) en lugar de modelos legacy, integrando la lógica con la fase activa del proyecto y la arquitectura hexagonal.

## 1. Resolución Dinámica de Modelos (Hexagonal)

Se ha implementado un desacoplamiento entre la solicitud de ejecución y la decisión del modelo final, permitiendo que el sistema sea consciente del contexto del proyecto.

### Componentes Clave:
*   **Provider de Fase**: `db.ObtenerFaseActivaProyecto` consulta en tiempo real la fase configurada en la tabla `progreso_fases`.
*   **Servicio de Capacidad**: `capacidadService` (inyectado en `agentesService`) resuelve el modelo base si no se especifica uno explícitamente.
*   **Resolución en DB**: `db.ResolverPerfilEjecucionLanzamiento` integra ahora la variable `fase`. Si la fase es `desarrollo`, el sistema prioriza modelos como `qwen2.5-coder:7b`.

## 2. Protección contra Payloads "Envenenados" (Legacy)

Existía un problema de persistencia donde los agentes, al recuperarse de un fallo (`local_runtime_failed`), reasumían el modelo de la sesión anterior (típicamente `gpt-5.4`).

### Correcciones Aplicadas:
*   **Saneamiento en Control Plane**: `cmd/controlplane_support.go` limpia modelos que contengan "gpt-5" durante el encolado de órdenes de recuperación.
*   **Saneamiento en Preparación**: `db/controlplane_entities.go` (`prepararStartRuntimeOrder`) ignora modelos legacy provenientes del payload persistido de la sesión. Esto fuerza al sistema a llamar de nuevo al resolvedor de políticas.

## 3. Nueva Flota de Agentes Locales

Se ha realizado una limpieza profunda de la flota para centrarse en modelos locales ejecutados vía Ollama.

### Agentes Retirados (Deshabilitados):
*   `Codex1` al `Codex8`, `Gemma1`, `LlamaWorker`, `QwenWorker`.

### Agentes Activos (Habilitados):
*   **QwenCoder1** (Supervisor/Programador)
*   **QwenCoder2** (Programador)
*   **QwenCoder3** (Programador)

### Configuración Automática:
El servidor ha sido reconfigurado para auto-arrancar esta flota en el proyecto `orquestador`:
```bash
orquesta config set server_autobootstrap_supervisor_agent QwenCoder1
orquesta config set server_autobootstrap_worker_agents QwenCoder1,QwenCoder2,QwenCoder3
```

## 4. Monitorización y Operación

### Comandos Útiles:
*   **Estado Global**: `./orquesta_server status` (muestra agentes activos, tareas y bloqueos).
*   **Consola de Agentes**: `tmux ls` (los agentes corren en sesiones de tmux aisladas).
*   **Gestión de Agentes**: `orquesta config agente-nuevo/retirar`.

### Verificación de Modelo:
Para confirmar qué modelo está usando un agente en su sesión actual:
```sql
SELECT resume_payload_json FROM sesiones WHERE agente = 'QwenCoder1' ORDER BY id DESC LIMIT 1;
```
*(Debería mostrar `qwen2.5-coder:7b` en la fase de desarrollo).*
