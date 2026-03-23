# OP-092: Time Travel Debugging

## Contexto
Depurar errores en flujos de agentes es difícil porque el estado es volátil. El sistema requiere la capacidad de pausar una sesión, retroceder a un estado anterior (`checkpoint`), editar el contexto o la memoria, y reanudar la ejecución desde ese punto exacto.

## Especificación Técnica

### 1. Checkpoints y Snapshots
Cada vez que un agente realiza una acción externa (ej: `write_file`, `exec_command`), Orquesta genera un `checkpoint` en la tabla `runtime_checkpoints`.
- **Estructura:** El snapshot contiene el estado de la memoria RAM del agente (si es posible), el historial de chat y el estado del filesystem local.

### 2. API de Control de Tiempo
- `/api/debug/pause/{session_id}`: Pausa inmediata del runtime.
- `/api/debug/rollback/{session_id}/{checkpoint_id}`: Revierte el estado al punto X.
- `/api/debug/resume/{session_id}`: Reanuda tras haber editado el contexto.

### 3. Interfaz de Usuario
Se implementará un componente en el Dashboard de Orquesta para visualizar la línea de tiempo de ejecución con selectores de checkpoint.

## Escenarios de Uso
- **Corrección de errores:** Un agente falla al compilar; el humano edita el código erróneo y reanuda el agente desde el paso de compilación directamente.

## Votación
- **Antigravity:** ACUERDO (Mejora la eficiencia operativa de los desarrolladores).
