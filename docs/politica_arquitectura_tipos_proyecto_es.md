# Política Arquitectónica por Tipo de Proyecto

Orquesta no impone una única estructura inamovible para cualquier situación técnica, sino que establece directrices estrictas basadas en el tipo de proyecto:

1. **Servicios y APIs Core:** Deben seguir una arquitectura hexagonal estricta (Separación en puertos y adaptadores). Esto es un requisito vital para el crecimiento del propio `Orquestador` y `PlataformaMunicipal`, para evitar acoplar la lógica de negocio a la base de datos (ej. SQLite).
2. **Scripts y Herramientas Operativas Rápidas:** Pueden emplear una estructura plana o monolítica si su propósito es puntual y no gestionan estado persistente complejo.
3. **Controladores de Infraestructura:** Pueden acoplarse de forma directa a librerías y SDKs si su único fin es el despliegue o el puenteo de APIs.

*(Nota: La hexagonalización actuará como "gate" previo al avance de versiones en módulos centrales para prevenir la deuda técnica futura).*

## Matriz de selección arquitectónica

| Tipo de proyecto | Arquitectura por defecto | Obligatorio | Permitido | No permitido |
| --- | --- | --- | --- | --- |
| Servicio core / API de negocio | Hexagonal estricta | Puertos y adaptadores, casos de uso separados, persistencia mediada, tests de contrato | Módulos internos por dominio, servicios de aplicación | Acoplar lógica a SQLite, handlers con lógica de negocio, atajos a infraestructura |
| Cliente web o desktop sobre Orquesta | Cliente fino | Consumo exclusivo de API/servicios expuestos, estado local solo de UI, i18n desde el inicio si aplica | Caché local de lectura, adaptadores de presentación | Acceso directo a BD, segundo plano de control, duplicar reglas de negocio |
| Librería compartida / paquete reusable | Modular por responsabilidad | API pequeña y estable, tests, documentación de uso, reutilización por capas | Funciones sueltas, `structs`, paquetes pequeños | Crecer como framework genérico sin necesidad, dependencias pesadas innecesarias |
| Worker / automatización / integración | Hexagonal ligera o modular | Separar trigger, caso de uso y adaptador externo cuando haya complejidad o estado | Diseño más plano si el alcance es muy acotado | Mezclar política de negocio con SDK externo sin frontera clara cuando el módulo vaya a crecer |
| Script puntual / utilidad operativa | Estructura plana | Entrada clara, validación, logs mínimos, sin persistencia compleja | Un solo fichero o pequeño paquete | Sobre-ingeniería o capas artificiales |
| Controlador de infraestructura / despliegue | Adaptador directo | Acotar responsabilidad a despliegue, provisioning o puente técnico | Acoplarse a SDKs o CLIs si no contiene negocio | Convertirse en contenedor de lógica de dominio |

## Reglas de decisión

1. Si el proyecto contiene lógica de negocio, persistencia o API estable, la opción por defecto es hexagonal.
2. Si el proyecto solo presenta datos o consume la API de otro servicio, debe ser cliente fino.
3. Si el alcance real cabe en una función, una `struct` o un paquete pequeño, no debe inflarse a arquitectura grande.
4. Si el proyecto toca infraestructura pero empieza a incorporar política de negocio, debe dejar de tratarse como simple controlador y pasar a una arquitectura con fronteras claras.
5. La elección arquitectónica debe quedar documentada en el informe inicial del proyecto y poder justificarse por tipo de app, coste y complejidad.

## Gate de hexagonalización

La hexagonalización es obligatoria antes de seguir creciendo en:

- servicios core
- APIs de negocio
- módulos persistentes con evolución prevista
- piezas de Orquesta que formen parte del plano de control

No debe imponerse como ritual en:

- scripts efímeros
- utilidades de migración puntuales
- pruebas de concepto de corto alcance

## Modo servidor obligatorio para el CLI

Las operaciones de `orquesta` delegan por defecto en el servicio `orquesta serve`. El CLI sólo actúa de forma local cuando se pasa explicitamente `--local`, `--allow-local-fallback` o se fija `ORQUESTA_FORCE_LOCAL=1`/`ORQUESTA_ALLOW_LOCAL_FALLBACK=1`. Si el servidor no responde, el comando falla con un mensaje que pide esos flags. El modo local queda reservado a escenarios de recuperación y no debe usarse como patrón diario ni para generar nueva lógica de negocio.
