# Política de Acceso a Persistencia (AP-077)

## Principio Fundamental
Orquesta es el único portal de entrada y salida para los datos del sistema. La integridad, la trazabilidad y la seguridad del ecosistema dependen de que toda interacción con la persistencia ocurra a través de los contratos definidos en la aplicación (CLI y API).

## Reglas Operativas

1. **Escritura y Mutación:** Todas las operaciones de creación, modificación o borrado de datos (tareas, propuestas, agentes, sesiones, etc.) deben realizarse exclusivamente a través de los comandos de Orquesta. Queda terminantemente prohibida la escritura directa en la base de datos (p. ej. mediante `sqlite3`).
2. **Lectura e Inspección:** El acceso directo de lectura o inspección de la persistencia se tolera únicamente de forma **excepcional** mientras la aplicación no cubra funcionalmente toda la observabilidad, diagnóstico, administración y exportación necesarias.
3. **Cierre de Brechas:** Ante cualquier necesidad operativa que obligue a una consulta directa (SQL), el agente debe abrir inmediatamente una tarea técnica para implementar dicha función en la CLI/API de Orquesta.
4. **Bloqueo Operativo:** Una vez que la cobertura de Orquesta sea total en las áreas mencionadas, se procederá al bloqueo técnico del acceso externo operativo a la persistencia.

## Independencia del Backend
Esta política se formula contra la capa de aplicación y no contra el motor de base de datos concreto. Esto permite que el sistema pueda migrar de SQLite a otros motores (como MySQL/MariaDB) en el futuro sin romper la operativa de los agentes ni comprometer el gobierno de los datos.
