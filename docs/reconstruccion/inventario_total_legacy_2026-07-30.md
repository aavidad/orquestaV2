# Inventario total del legado: problema y procedimiento autorizado

Fecha: 2026-07-30.

Estado: procedimiento autorizado y trabajo pendiente. Este documento no afirma
que el inventario esté terminado, no acredita equivalencia con el producto
nuevo y no modifica el estado de ninguna capacidad.

## 1. Problema que se debe resolver

Los censos anteriores fueron útiles para localizar código Go, documentos,
ramas, incidencias y generaciones históricas. No demostraron que toda conducta
útil del sistema antiguo estuviera representada en el catálogo y reimplementada
en Orquesta V2.

Un censo de paquetes o funciones Go no cubre por sí solo:

- órdenes de línea de comandos, rutas HTTP, herramientas MCP, kit de desarrollo
  y web;
- claves de configuración, nombres alternativos, valores por defecto y secretos;
- migraciones SQL, procedimientos de recuperación y datos existentes;
- guiones del intérprete de órdenes o Python, instaladores y tareas de
  mantenimiento;
- unidades de servicio, contenedores, despliegues y operaciones de actualización;
- archivos, sockets, procesos, perfiles, sesiones y demás estado operativo;
- pruebas, casos de prueba, pruebas de humo, mutaciones y comprobaciones
  manuales;
- requisitos, decisiones, incidencias, manuales operativos y evidencia
  histórica;
- conductas implícitas en la composición o en herramientas externas.

Tampoco basta con que una ruta aparezca en una lista. Para acreditar
equivalencia hay que clasificar su semántica, enlazarla con una capacidad
canónica, expresar sus invariantes como contrato o prueba neutral, reimplementarla
sin depender del entorno de ejecución antiguo y obtener evidencia de la misma
revisión y
composición.

Los inventarios previos son, por tanto, fuentes de búsqueda y trazabilidad.
Nunca sustituyen a `product/roadmap.json`, `product/capabilities.json` ni a los
justificantes de `product/evidence/`.

## 2. Autoridad y límites

El orden de autoridad continúa siendo:

1. `AGENTS.md`;
2. `product/roadmap.json`;
3. `product/capabilities.json` y `product/evidence/`;
4. `docs/reconstruccion/ruta_total_100.md`;
5. contratos locales compatibles.

El legado es material de consulta. No puede convertirse de nuevo en autoridad
de ejecución, dependencia de pruebas o fuente de estado compartida.

Quedan prohibidos:

- importar paquetes completos desde el árbol antiguo;
- arrancar su entorno de ejecución para completar el inventario;
- escribir en `/home/alberto/Trabajo/orquesta` o en la copia de consulta;
- desactivar el `sparse-checkout` del producto nuevo;
- crear adaptadores, dobles escrituras o rutas de respaldo hacia el legado;
- declarar equivalencia por similitud de nombres, número de ficheros o ausencia
  de resultados en una búsqueda;
- convertir automáticamente un hallazgo en capacidad, vertical o decisión.

Una discrepancia con la autoridad vigente detiene únicamente el conjunto de
escritura afectado. Se resuelve primero en el catálogo.

## 3. Universos incluidos

El inventario debe recorrer, como universos separados, al menos:

### 3.1 Producto nuevo

- todos los ficheros versionados obtenidos mediante `git ls-files`;
- ficheros nuevos todavía no versionados que pertenezcan a un conjunto de
  cambios declarado;
- configuración, código generado, casos de prueba y evidencia;
- binarios, servicios y artefactos construidos por procedimientos vigentes;
- estado operativo creado por una prueba o despliegue reclamado.

Este universo sirve para comprobar qué existe realmente en V2 y qué cobertura
lo acredita.

### 3.2 Consulta legacy

- `/home/alberto/Trabajo/orquestaV2-legacy-consulta`;
- `/home/alberto/Trabajo/orquesta`, únicamente cuando una fuente no esté
  representada en la copia de consulta y siempre en solo lectura;
- ramas, etiquetas, paquetes Git y espacios de trabajo históricos ya
  identificados;
- repositorios auxiliares relacionados que estén registrados en la matriz de
  rescate;
- documentación, incidencias, pruebas y datos de ejemplo asociados.

Cada origen debe conservar ruta o referencia Git, revisión cuando exista,
resumen criptográfico y fecha de observación. Una misma conducta repetida en
varias generaciones se deduplica por significado, pero se conservan todos sus
orígenes.

El recuento físico inicial de `/home/alberto/Trabajo/orquesta` encontró 185
referencias, 152 puntas únicas, 4.204 commits y 4.176 árboles alcanzables. Son
medidas de descubrimiento, no una declaración de universo cerrado.

También se han localizado como fuentes candidatas:

- `orquesta.bk`;
- `orquesta-autonomia-clean`;
- dos generaciones de autoprogramación;
- la generación de objetivos y espacios de trabajo;
- el rebuild y V07;
- la generación de lease y generación;
- `PlataformaMunicipal/orquestador`;
- los worktrees de Orquesta V2.

Los nombres anteriores identifican familias que deben resolverse a rutas,
repositorios, revisiones y resúmenes exactos. No se afirmará que el universo
está cerrado hasta producir, contrarrevisar y sellar un manifiesto de fuentes
que incluya todas las raíces descubiertas y explique cualquier exclusión.

### 3.3 Estado operativo autorizado

Solo se inspeccionan raíces declaradas y reclamadas:

- bases SQLite;
- almacenes de artefactos;
- configuración efectiva redactada;
- copias de seguridad;
- espacios de trabajo;
- directorios de ejecución y perfiles;
- sockets, identificadores de proceso, grupos de control y arrendamientos;
- diarios y justificantes.

No se recorre un `HOME` completo ni se leen secretos. El inventario operativo
registra referencias opacas, propietario, tamaño, resumen, estado y política de
conservación. Los datos sensibles se omiten o redactan.

## 4. Universos excluidos

Se excluyen del análisis semántico ordinario, pero su exclusión debe quedar
contada y justificada:

- `.git/**` y objetos internos de Git;
- dependencias vendorizadas y licencias de terceros, salvo parches propios,
  conexión propia u obligaciones de licencia;
- caches, temporales y resultados de compilación no reclamados;
- datos personales, credenciales y secretos;
- sistemas de producción, OPES productivo y sesiones ajenas;
- artefactos duplicados byte a byte, conservando un registro de cada origen;
- material corrupto o ilegible, que se registra como hueco y no como ausencia
  de conducta.

Excluir no significa borrar. Ningún material se elimina como parte del
inventario.

## 5. Familias que se deben inventariar

Cada elemento se asigna a una o más familias:

1. Dominio, casos de uso y reglas de ciclo de vida.
2. Puertos, adaptadores, composición y proveedores.
3. Interfaz de órdenes (CLI), HTTP, MCP, kit de desarrollo, web, aplicación web
   instalable (PWA) y clientes instalables.
4. Registro de comandos, schemas públicos, autorización e identidad.
5. Configuración, nombres alternativos, valores por defecto, generación,
   interfaz e internacionalización.
6. SQL, migraciones, recuperación, copia, restauración y consistencia.
7. Filesystem, Git, worktrees, artefactos y políticas de conservación.
8. Agentes, perfiles, cuentas, credenciales, sesiones, procesos y cuotas.
9. Planificación, capacidad, concurrencia, latidos, tiempos máximos y
   recuperación.
10. Herramientas, recursos, instrucciones, capacidades instalables,
    complementos y contexto.
11. Red, navegador, representante de red, aislamiento, máquinas virtuales y
    recinto de ejecución.
12. Efectos externos, aprobaciones, idempotencia, reversión y justificantes.
13. Despliegue, systemd, contenedores, instalación, actualización y retirada.
14. Telemetría, salud, disponibilidad, vigilancia, alertas y notificaciones.
15. Guiones del intérprete de órdenes, Python y otros lenguajes o formatos
    ejecutables.
16. Pruebas unitarias, contractuales, negativas, de carrera, mutación y E2E.
17. Casos, datos y pruebas de humo reales, incluidas las comprobaciones de
    activación voluntaria.
18. Requisitos, decisiones, manuales, procedimientos, relevos e incidencias.
19. Evidencias, manifiestos de publicación, binarios, imágenes y configuración
    efectiva.
20. Integraciones externas, aplicaciones consumidoras, OPES y dominios
    especializados.

La extensión del fichero solo ayuda a localizarlo. No determina su familia ni
su autoridad.

## 6. Registro mínimo por elemento

El registro estructurado debe conservar:

```text
source_ref:
source_revision:
source_digest:
source_kind:
family:
summary:
capability_ids:
acceptance_contracts:
invariants:
negative_lessons:
consumers:
v2_refs:
evidence_refs:
disposition:
reason:
review_refs:
```

Los requisitos expresados en lenguaje natural deben conservar texto de origen
por resumen criptográfico y ubicación, pero no se convierten en órdenes
automáticas. Las incidencias conservan identificador, causa arquitectónica,
invariante roto, prueba de lección y evidencia de cierre.

## 7. Estados

### 7.1 Estado del elemento inventariado

```text
discovered -> classified -> mapped -> characterized -> reviewed
```

- `discovered`: se conoce el origen, pero no su significado;
- `classified`: tiene familia, autoridad y disposición;
- `mapped`: enlaza con capacidades y criterios de aceptación, o documenta un
  hueco;
- `characterized`: la conducta útil está expresada mediante contrato, caso de
  prueba o prueba neutral;
- `reviewed`: una revisión independiente confirmó clasificación y mapeo.

Estos estados describen el inventario. No acreditan producto.

### 7.2 Disposición

Cada elemento revisado recibe exactamente una disposición:

- `reimplemented_and_accredited`;
- `reimplemented_not_accredited`;
- `planned`;
- `candidate_requires_roadmap_decision`;
- `historical_evidence_only`;
- `duplicate_with_traced_origin`;
- `third_party`;
- `rejected_with_reason`;
- `unresolved_gap`.

Una disposición diferida, candidata o histórica no cuenta como equivalencia.

### 7.3 Estado de capacidad

El único vocabulario de cierre sigue siendo:

```text
declared -> implemented -> wired -> exercised -> accredited
```

Solo `accredited` cuenta como terminado.

## 8. Gates de cero omisiones

El inventario total solo podrá declararse completo cuando una comprobación
mecánica y una revisión independiente demuestren simultáneamente:

```text
fuentes_descubiertas_sin_clasificar == 0
elementos_clasificados_sin_disposicion == 0
disposiciones_sin_razon_o_revision == 0
requisitos_normativos_sin_capacidad_o_hueco == 0
capacidades_sin_criterio_ejecutable == 0
criterios_sin_prueba_y_evidencia_exigibles == 0
guiones_activos_sin_validacion == 0
migraciones_sin_upgrade_restart_recovery == 0
comandos_sin_paridad_publica_exigida == 0
textos_publicos_sin_catalogo == 0
incidencias_sin_invariante_y_prueba_de_leccion == 0
evidencias_sin_revision_y_composicion_exactas == 0
artefactos_operativos_reclamados_sin_inventario == 0
elementos_unresolved_gap == 0
```

Además:

- el número de entradas debe reconciliarse contra un censo físico reproducible;
- altas, bajas y cambios entre dos ejecuciones deben quedar explicados;
- cada exclusión debe pertenecer a una categoría cerrada;
- muestras adversariales deben demostrar que un fichero nuevo, una extensión
  desconocida, una ruta con espacios y un symlink hostil no desaparecen;
- otra revisión debe repetir el censo desde una copia limpia;
- el control global del catálogo debe seguir verde sobre la misma revisión.

No es válido reducir el universo hasta conseguir cero huecos.

## 9. Procedimiento por oleadas

### Oleada 0: congelar método y alcance

1. Registrar revisiones, raíces permitidas y exclusiones.
2. Fijar schema, vocabulario y algoritmo de resumen.
3. Presupuestar tiempo, disco y número de fuentes.
4. Ejecutar la consulta de lecciones legacy.
5. Probar el censo con casos adversariales.

Resultado: método reproducible, todavía sin afirmar cobertura.

### Oleada 1: censo físico

1. Enumerar el producto nuevo mediante el índice Git.
2. Enumerar cada fuente legacy sin modificarla.
3. Inventariar solo las raíces operativas reclamadas.
4. Registrar tipo, revisión, tamaño y resumen.
5. Reconciliar totales y duplicados.

Resultado: universo físico conocido, todavía sin clasificación semántica.

### Oleada 2: superficies ejecutables y datos

Clasificar código, interfaces públicas, configuración, SQL, guiones,
despliegue, servicios, artefactos operativos, pruebas y casos de prueba.
Ejecutar
validadores seguros y no mutantes cuando sea posible.

Resultado: mapa de superficies y huecos técnicos.

### Oleada 3: requisitos y conocimiento

Revisar documentación, decisiones, procedimientos, relevos e incidencias por
bloques
completos. Extraer requisitos normativos, invariantes, negativos y criterios de
aceptación. Deduplicar por significado sin perder procedencia.

Resultado: mapa de obligaciones y lecciones.

### Oleada 4: mapeo contra V2

Para cada elemento caracterizado:

1. buscar capacidad y semántica vigente;
2. decidir `reuse | replace | new`;
3. localizar implementación, conexión, ejercicio y evidencia;
4. registrar contradicciones y huecos;
5. verificar la misma revisión, binario o imagen y configuración efectiva.

Resultado: matriz de equivalencia; todavía puede contener huecos.

### Oleada 5: convertir huecos en trabajo

Los huecos se agrupan por capacidad, dimensión y dependencia causal. Cada tarea
incluye:

```text
fuentes:
capability IDs:
invariante:
autoridad que escribe:
puertos y adaptadores:
conjunto de escritura:
dependencias:
prueba contractual:
negativos, carrera y reinicio:
prueba de extremo a extremo y evidencia:
presupuesto:
```

Si una capacidad acreditada carece de la conducta prometida, se registra una
incidencia de falso verde y se rectifica su acreditación según el procedimiento
autoritativo. Si el requisito es nuevo, se decide primero en el catálogo. El
inventario no crea IDs ni modifica estados por sí solo.

### Oleada 6: reimplementación y acreditación

Orquesta dirige objetivos pequeños y causales. Cada uno implementa, conecta,
ejercita, revisa y acredita una porción concreta. No se copia el entorno de
ejecución antiguo ni
se mezclan conjuntos de escritura.

### Oleada 7: contrarrevisión y cierre

1. Repetir el censo desde una revisión limpia.
2. Ejecutar los controles de cero omisiones.
3. Contrastar una muestra independiente por familia.
4. Ejecutar los controles globales de producto.
5. Sellar registro, fuentes y evidencia por resumen.

Solo entonces podrá afirmarse que el inventario está completo. La equivalencia
del producto requerirá además que todos los elementos aplicables estén
`reimplemented_and_accredited`.

## 10. Brechas ya confirmadas

Estas brechas están demostradas por el estado vigente. Su presencia no
significa que el resto del inventario esté completo.

### 10.1 Elasticidad de agentes

`ORC-10` promete 70 agentes principales por ciclo y prohíbe un techo global
oculto. El adaptador Codex actual recibe la lista de perfiles al arrancar,
construye un adaptador de capacidad uno por perfil y no incorpora o retira
capacidad en caliente.

El límite físico efectivo queda condicionado por el número de perfiles
preconfigurados. La prueba histórica de reclamaciones concurrentes no acredita
aprovisionamiento elástico, arranque físico paralelo ni reducción dinámica.

Falta acreditar:

- cálculo de demanda completa;
- creación concurrente de tantos entornos como autoricen los límites reales;
- espera por capacidad sin consumir intento;
- ampliación sin reiniciar Orquesta;
- parada y liberación individual sin afectar a la cohorte;
- conservación de cada entorno hasta revisión.

### 10.2 Cuota y disponibilidad

El producto distingue presupuestos y perfiles, pero no dispone todavía de una
fuente viva completa que permita ampliar capacidad física según cuota,
credenciales autorizadas y disponibilidad del proveedor.

Una cuota agotada, credencial no válida o recurso físico insuficiente debe
quedar como causa explícita y recuperable. No puede convertirse en fallo del
agente, gasto de intento, sustitución circular ni límite estático silencioso.

Falta inventariar y acreditar la cadena completa de:

- observación de cuota con vigencia;
- reserva y liberación;
- incorporación y retirada segura de capacidad;
- aislamiento de identidad y credencial;
- espera, demora creciente y recuperación;
- métricas de demanda, capacidad preparada, activa y pendiente.

### 10.3 Entorno físico Firecracker para agentes

Existen contratos y adaptadores parciales para el paquete de entrada,
autorización de lanzamiento, red por vsock y arrendamiento de CID. Su estado es
`planned_not_applied`.

El lanzador y la máquina invitada físicos existentes pertenecen al atestador de
pruebas. No acreditan un entorno de ejecución de agentes y no deben reutilizarse
como si fueran el mismo producto.

Falta implementar y acreditar:

- una microVM aislada por ejecución de agente;
- sistema de archivos raíz específico con el agente;
- lanzamiento, observación, latido y recuperación físicos;
- intermediario y representante de red controlados por vsock, sin red IP del
  invitado;
- credenciales efímeras y entorno mínimo;
- parada cooperativa y forzada por identidad exacta;
- congelación, inventario por resumen y conservación;
- recuperación después de caída sin doble agente;
- lotes grandes, incluido el objetivo inmediato de 70 cuando el equipo y el
  presupuesto lo permitan;
- cierre con cero procesos, sockets, grupos de control o CID activos, sin borrar
  datos pendientes de revisión.

### 10.4 Órdenes durante una ejecución

El adaptador Codex actual lanza `codex exec --ephemeral`. Entrega la orden
inicial por la entrada estándar y conserva un resultado final, pero no mantiene
un canal por el que Orquesta pueda introducir otra instrucción textual en ese
mismo proceso.

El agente sí puede llamar a las herramientas MCP autorizadas para su ejecución.
Eso permite consultar o entregar información a Orquesta, pero no convierte el
buzón durable en una conversación viva. El buzón comunica trabajos y
ejecuciones; no reanuda la sesión efímera ni introduce una corrección en el
proceso que ya está trabajando.

Falta inventariar y acreditar:

- envío de una nueva orden a un agente vivo con identidad y autorización
  exactas;
- confirmación durable de recepción, aplicación o rechazo;
- reanudación de una sesión conservando el contexto autorizado;
- redirección, pausa y cancelación sin confundir proceso, ejecución y Goal;
- recuperación después de caída sin repetir una orden ya aplicada;
- prueba de que una corrección llega al agente adecuado mientras otros agentes
  continúan trabajando;
- una alternativa explícita y segura cuando el proveedor solo admita
  ejecuciones de una única orden.

## 11. Criterio de honestidad

Este documento autoriza el procedimiento, no su resultado.

Hasta que todas las oleadas se ejecuten y los controles de cero omisiones queden
sellados, las afirmaciones correctas son:

- existen censos parciales y fuentes identificadas;
- existen capacidades acreditadas de Orquesta V2;
- existen brechas confirmadas;
- el inventario total y la equivalencia completa siguen pendientes.

No se debe afirmar «legacy cubierto», «equivalencia completa» ni «Orquesta al
100 %» basándose únicamente en este documento.
