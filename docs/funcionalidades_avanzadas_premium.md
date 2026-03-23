# Análisis de Funcionalidades Avanzadas "Premium" (Missed Features)

Tras un segundo barrido profundo en los 8 frameworks (especialmente LangGraph, Gastown y CrewAI), estas son las 3 joyas que Orquesta aún no tiene y que la elevarían a nivel "Enterprise":

## 1. Memoria de Entidades (Inspiración: CrewAI)
- **Qué es:** No es solo "recordar el chat". Es tener una base de datos de conocimientos del proyecto (`project_knowledge`) que los agentes alimentan.
- **Ejemplo:** Si el Agente A descubre que "la base de datos de producción solo acepta conexiones TLS 1.3", lo guarda como una `Entidad`. Semanas después, el Agente B (que no estuvo en esa sesión) consulta la memoria antes de configurar un conector y ya conoce esa restricción.
- **En Orquesta:** Crear una tabla `entidades_conocimiento` que los agentes consulten por defecto.

## 2. "Time Travel" y Edición de Estado (Inspiración: LangGraph)
- **Qué es:** La capacidad de pausar un agente, volver a un `checkpoint` anterior, **editar un archivo o una variable** que el agente confundió, y decirle: *"Reanuda desde aquí con este cambio"*.
- **En Orquesta:** Aprovechar nuestros `runtime_checkpoints` para permitir que Alberto, desde la UI web, modifique el contexto antes de un `resume`. Esto evita que el agente repita el mismo error tras un Nudge fallido.

## 3. El "Refinery" o Puerta de Seguridad (Inspiración: Gastown)
- **Qué es:** Un agente nunca hace `merge` a la rama principal directamente. Los cambios van a una "Cola de Refinería". Un agente supervisor de "Calidad" (o el propio Alberto) debe ver el reporte de tests verdes antes de que Orquesta ejecute el merge físico.
- **En Orquesta:** Implementar la `Fase de Revisión` como un bloqueo técnico real del Git Worktree controlado por el Daemon.

## 4. A2UI: Generación de UI Declarativa (Inspiración: OpenClaw)
- **Qué es:** Que el agente pueda "pintar" un botón o una gráfica en el panel de Orquesta para pedirle algo a Alberto, sin tener que programar el CSS.
- **En Orquesta:** Permitir que el Agente mande un JSON de "UI Schema" que el panel `serve` renderice automáticamente (formularios de aprobación, selectores, etc.).

---
**¿Cuál de estas te parece más prioritaria para que la formalice en una OP?**
