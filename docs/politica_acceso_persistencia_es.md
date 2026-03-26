# Política de Acceso a Persistencia (AP-077)

## Principio Fundamental
Orquesta es el único portal de entrada y salida para los datos del sistema. La integridad, la trazabilidad y la seguridad del ecosistema dependen de que toda interacción con la persistencia ocurra a través de la API y de los contratos de aplicación expuestos por el servidor.

## Reglas Operativas

1. **Escritura y Mutación:** Todas las operaciones de creación, modificación o borrado de datos (tareas, propuestas, agentes, sesiones, etc.) deben realizarse exclusivamente a través de la API/servicios de Orquesta. Queda terminantemente prohibida la escritura directa en la base de datos (p. ej. mediante `sqlite3`).
2. **Lectura e Inspección:** La lectura directa de persistencia solo se tolera de forma **excepcional** para diagnóstico o mantenimiento técnico mientras la API no cubra una necesidad real.
3. **Sin Operación Normal sin API:** Ninguna superficie de la aplicación debe operar normalmente contra la BD local. Si existe API, debe usarse la API; el flujo local deja de ser válido como comportamiento diario.
4. **Cierre de Brechas:** Ante cualquier necesidad operativa que obligue a una consulta directa (SQL), el agente debe abrir inmediatamente una tarea técnica para implementar dicha función en la API/CLI server-first de Orquesta.
5. **Bloqueo Operativo:** Una vez que la cobertura de Orquesta sea total en las áreas mencionadas, se procederá al bloqueo técnico del acceso externo operativo a la persistencia.

## Independencia del Backend
Esta política se formula contra la capa de aplicación y no contra el motor de base de datos concreto. Esto permite que el sistema pueda migrar de SQLite a otros motores (como MySQL/MariaDB) en el futuro sin romper la operativa de los agentes ni comprometer el gobierno de los datos.

## Estado de Transición

A fecha de `2026-03-24`, la dirección del proyecto ya es estable:

- la CLI y la API cubren la mayor parte de lecturas y mutaciones operativas frecuentes
- el objetivo sigue siendo `cero acceso directo` como flujo normal
- el acceso excepcional de diagnóstico sigue tolerado solo mientras existan huecos técnicos reales

Los bloqueos que impiden activar ya el bloqueo técnico total son:

- consolidar un único daemon oficial para operación continua
- sacar `runtime_connector` y `agente_console` del camino operativo principal
- cerrar la capa web como cliente fino completo
- completar humo extremo a extremo con runtime vivo real

El inventario operativo de estos huecos y su orden recomendado de cierre se mantiene en [Inventario de pendientes para orquestación autónoma](inventario_pendientes_orquestacion_autonoma_2026-03-24.md).

Mientras dure la transición, la regla para agentes y programadores es simple:

- no usar `orquesta.db` como backend operativo manual
- no introducir nuevas rutas locales de negocio
- mover cada capacidad a API y retirar su fallback local en cuanto exista cobertura
