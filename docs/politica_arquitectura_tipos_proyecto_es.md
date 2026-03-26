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

## Modo servidor obligatorio para toda la app

La aplicacion Orquesta, incluida su CLI, la web, los scripts auxiliares y cualquier cliente futuro, debe operar contra `orquesta serve` y su API como camino normal y oficial.

Reglas vinculantes:

1. No se debe usar la base de datos local como backend operativo normal desde clientes, scripts, paneles o comandos de negocio.
2. No se debe introducir nueva logica de producto que funcione solo en modo local o que requiera abrir `orquesta.db` sin pasar por API/servicios expuestos.
3. El acceso local queda reservado exclusivamente a recuperacion explicita, diagnostico excepcional o mantenimiento tecnico temporal mientras exista una brecha real de API.
4. Cuando una capacidad exista por API, el flujo local equivalente debe dejar de usarse y debe retirarse progresivamente.
5. La meta del proyecto es `cero operacion normal sin API`.

En consecuencia, cualquier nueva funcionalidad de la app debe nacer `server-first`, y cualquier flujo heredado que siga dependiendo de BD local se considera deuda tecnica a eliminar.
