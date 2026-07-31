# 07. Colaboración, Consejo, revisión y contexto

> **Responsabilidad:** explicar cómo varios agentes colaboran
> sobre un único `Goal` sin crear autoridades paralelas, cómo se entregan
> resultados causales, cómo se acredita una revisión independiente y qué
> decisión añade el Consejo.
>
> **Alcance:** Director transferible, buzón causal entre padre e hijo,
> subagentes, autor, revisión primaria, revisión adversarial, Consejo,
> retrabajo, relevo de sesión, referencias de artefactos, límites de contexto,
> presupuesto, fallos, pruebas y diagnóstico.
>
> **No acredita:** la documentación no cambia el estado de ninguna capacidad.
> El buzón causal acotado y las revisiones y Consejo descritos como vigentes sí
> tienen evidencia en sus revisiones acreditadas. La mensajería genérica y el
> relevo de sesión de `ORC-15`, y el gestor de contexto de `ORC-19`, siguen
> declarados: aquí se diseña su continuación, pero no se presentan como
> terminados.

## Al terminar este capítulo, el lector sabrá…

- encargar trabajo a subagentes sin crear autoridades paralelas;
- distinguir entrega causal, relevo de sesión y recuperación de contexto;
- acreditar la independencia de autor y revisores;
- explicar cuándo interviene el Consejo y por qué no integra ni cierra;
- diagnosticar bloqueos de capacidad, entrega, revisión o decisión.

Este capítulo usa «Consejo» para la función que el código y algunos contratos
históricos llaman `Council`. Usa «revisión» para `review`, «retrabajo» para
`rework` y «relevo» para `handoff`. La forma canónica de los términos comunes,
incluidos buzón causal, ciclo de vida, recibo y huella criptográfica, se fija en
el [glosario técnico común](01_mision_limites_y_vocabulario.md#glosario-técnico-común).
Los identificadores literales se conservan cuando forman parte de un contrato o
del código.

La autoridad vigente se consulta en la base versionada
`HEAD:product/roadmap.json`, mediante `git show HEAD:product/roadmap.json`, en
[`product/capabilities.json`](../../../product/capabilities.json) y en
[`product/evidence/`](../../../product/evidence/). Los análisis de V13, V18 y
V19 explican decisiones de diseño; no sustituyen ese estado vivo.

## Índice del capítulo

- [7.1 La colaboración no crea otro orquestador](#71-la-colaboración-no-crea-otro-orquestador)
- [7.2 Capacidades y estado honesto](#72-capacidades-y-estado-honesto)
- [7.3 El Director es una función transferible](#73-el-director-es-una-función-transferible)
- [7.4 Subagentes: paralelismo acotado, no jerarquías soberanas](#74-subagentes-paralelismo-acotado-no-jerarquías-soberanas)
- [7.5 Buzón causal acreditado](#75-buzón-causal-acreditado)
- [7.6 Qué falta para mensajería y relevo de sesión](#76-qué-falta-para-mensajería-y-relevo-de-sesión)
- [7.7 Paquetes de contexto](#77-paquetes-de-contexto-diseño-para-orc-19)
- [7.8 Autor y revisiones independientes](#78-autor-y-revisiones-independientes)
- [7.9 Retrabajo sin destruir evidencia](#79-retrabajo-sin-destruir-evidencia)
- [7.10 Qué es exactamente el Consejo](#710-qué-es-exactamente-el-consejo)
- [7.11 Sujeto y políticas del Consejo](#711-sujeto-y-políticas-del-consejo)
- [7.12 Papeletas y decisión](#712-papeletas-y-decisión)
- [7.13 De la intención a la integración](#713-de-la-intención-a-la-integración)
- [7.14 Presupuestos y capacidad](#714-presupuestos-y-capacidad)
- [7.15 Fallos y respuesta correcta](#715-fallos-y-respuesta-correcta)
- [7.16 Diagnóstico operativo](#716-diagnóstico-operativo)
- [7.17 Pruebas mínimas](#717-pruebas-mínimas)
- [7.18 Orden recomendado de implementación desde cero](#718-orden-recomendado-de-implementación-desde-cero)
- [7.19 Criterio de cierre](#719-criterio-de-cierre)

## 7.1 La colaboración no crea otro orquestador

La regla central es sencilla:

```text
un Goal
  └── un DAG de WorkItems
        ├── ejecuciones de autor
        ├── ejecuciones de revisión
        ├── ejecuciones del Consejo
        └── mensajes y artefactos causales

un escritor del ciclo de vida: internal/application
un planificador de aplicación
una fuente transaccional de estado por despliegue
```

Un agente no posee un `Goal`, una cola ni una base de datos privada. Una
ejecución tampoco es una autoridad de cierre. El agente recibe una capacidad
temporal y acotada para actuar; el motor comprueba identidad, permiso,
generación, revisión esperada, presupuesto, riesgo y evidencia antes de aceptar
una mutación.

La misma regla se aplica al Director, a un subagente, a un revisor y a cada
integrante del Consejo. Sus diferencias son de función y de permiso, no de
arquitectura.

### Qué es cada cosa

| Concepto | Qué hace | Qué no hace |
|---|---|---|
| Director | Propone planes, descomposición, asignaciones y replanteamientos bajo un arrendamiento propio | No escribe directamente el ciclo de vida ni mantiene una cola privada |
| Subagente | Ejecuta un `WorkItem` hijo con alcance, presupuesto y referencias explícitas | No hereda autoridad ilimitada del padre |
| Buzón causal | Entrega una resolución compacta de un hijo a su destinatario exacto | No es un chat general ni reasigna mensajes por conveniencia |
| Autor | Produce el cambio que será sometido a revisión | No aprueba su propio cambio |
| Revisor primario | Busca corrección, ajuste al contrato, regresiones y calidad ordinaria | No sustituye al adversarial |
| Revisor adversarial | Intenta refutar el candidato mediante fallos, negativos y amenazas | No comparte lanzamiento con autor o revisor primario |
| Consejo | Decide si un candidato ya revisado puede proseguir según una política | No revisa el código de nuevo, no integra y no cierra el `Goal` |
| Integrador autorizado | Aplica una operación explícita sobre el sujeto exacto ya validado | No acepta decisiones o aprobaciones obsoletas |

## 7.2 Capacidades y estado honesto

La colaboración no debe describirse con un porcentaje subjetivo. Debe
descomponerse por capacidades:

| Bloque | Capacidades principales | Estado vivo al redactar |
|---|---|---|
| Director transferible | `GOV-09` | acreditado |
| Buzón hijo, artefactos causales y cierre del padre | `ORC-04`, `ORC-05`, `ORC-14` | acreditado |
| Autor y dos revisiones independientes | `GOV-12`, `STG-13`, `STG-14`, `STG-16`, `EVD-06` | acreditado |
| Políticas y decisión del Consejo | `GOV-11`, `GOV-13`, `GOV-14`, `STG-08`, `EVD-07` | acreditado |
| Mensajería genérica y relevo entre sesiones | `ORC-15` | declarado |
| Gestor compacto de contexto | `ORC-19` | declarado |

Que `ORC-15` o `ORC-19` estén declarados no invalida el buzón causal ya
acreditado. Significa que su alcance es deliberadamente menor: hijo directo,
destinatario exacto y resolución contractual. No se debe bautizar esa parte
como «sistema completo de colaboración» para aparentar una cobertura que aún
no existe.

Las evidencias concretas de los bloques cerrados son
[`v13_mailbox.json`](../../../product/evidence/v13_mailbox.json),
[`v18_independent_reviews.json`](../../../product/evidence/v18_independent_reviews.json)
y [`v19_council.json`](../../../product/evidence/v19_council.json). Deben
comprobarse junto con el árbol y la revisión que acreditan; no basta con que el
archivo exista.

## 7.3 El Director es una función transferible

El Director puede ser desempeñado por Hermes, Codex, Claude, Gemini, un agente
local o un operador humano siempre que todos usen el mismo protocolo. Su
autoridad se prueba mediante:

- una referencia de actor;
- una referencia de ejecución;
- un arrendamiento de Director vigente;
- un cercado monotónico que invalida poseedores anteriores;
- el proyecto y `Goal` exactos;
- permisos y presupuesto suficientes;
- una clave de idempotencia.

El arrendamiento del Director y el arrendamiento de ejecución son contratos
distintos. El primero autoriza a proponer decisiones de dirección; el segundo
autoriza a trabajar sobre un `WorkItem`. No deben combinarse en un campo
ambiguo.

### Transferencia segura

```text
Director A obtiene arrendamiento con cercado 41
        │
        ├── propone plan P3
        │
        └── pierde o entrega el arrendamiento
                         │
Director B obtiene arrendamiento con cercado 42
        │
        ├── continúa desde hechos durables
        └── toda propuesta tardía con cercado 41 es rechazada
```

El nuevo Director reconstruye la situación desde el `Goal`, sus `WorkItems`,
recibos, artefactos, decisiones y buzón. No depende de la memoria conversacional
del Director anterior.

### Fallos que debe impedir

- dos Directores escribiendo a la vez;
- un Director antiguo que reaparece después de perder el arrendamiento;
- una propuesta de otra generación;
- una decisión sin actor o proyecto explícitos;
- una replanificación que borra pruebas, artefactos o causalidad;
- una suspensión temporal confundida con fracaso terminal.

La pérdida del Director no mata a las ejecuciones válidas. El motor conserva
sus hechos y permite que un nuevo Director prosiga.

## 7.4 Subagentes: paralelismo acotado, no jerarquías soberanas

Un subagente ejecuta un `WorkItem` hijo del DAG. La relación padre-hijo expresa
causalidad y, cuando se activa de forma explícita, una obligación de entrega.
No significa que el proceso padre gestione directamente la vida del proceso
hijo.

Cada encargo debería contener como mínimo:

```text
referencia del objetivo (`GoalRef`)
referencia del proyecto (`ProjectRef`)
generación del plan (`PlanGeneration`)
referencia de la unidad de trabajo padre (`ParentWorkItemRef`), si existe
referencia de la unidad de trabajo (`WorkItemRef`)
referencias de ejecución e intento (`ExecutionRef` y `AttemptRef`)
objetivo y criterios de aceptación
conjunto de escritura permitido
dependencias causales
permisos y herramientas
presupuesto de tiempo, coste, contexto y artefactos
referencias de entrada y sus huellas criptográficas
política de entrega
```

El proveedor de agentes puede levantar cinco, veinte o setenta ejecuciones si
la capacidad, el presupuesto y el DAG lo permiten. Esa elasticidad pertenece
al plano de ejecución descrito en el capítulo 4. El modelo de colaboración no
debe contener un conjunto fijo de nombres, terminales o procesos.

### Granularidad adecuada

Una tarea para subagente debe poder:

1. entenderse con un paquete de contexto pequeño;
2. tener un conjunto de escritura estrecho;
3. verificarse de forma independiente;
4. producir una resolución estructurada;
5. finalizar o bloquearse sin dejar ambigua la autoridad.

Si dos tareas escriben el mismo fichero o dependen del mismo cambio causal, no
son paralelas solo porque haya agentes libres. Se serializan o se redefine su
frontera.

### Mantener un agente o cerrarlo

La identidad lógica y el proceso físico no son lo mismo:

- una ejecución termina cuando entrega su resultado contractual;
- sus recibos, artefactos, trazas y decisión se conservan;
- el proceso o microVM puede retirarse de forma ordenada;
- otra ejecución puede reutilizar capacidad física limpia, pero nunca debe
  heredar una identidad, un directorio o credenciales residuales;
- un agente ocioso puede mantenerse caliente si una política de capacidad lo
  justifica, sin convertirlo en autoridad residente.

Cerrar el proceso no borra su historia. Mantenerlo abierto no prorroga por sí
solo su permiso ni su arrendamiento.

## 7.5 Buzón causal acreditado

El contrato de V13 entrega resoluciones de un hijo directo a un destinatario
exacto. Sus datos principales se pueden estudiar en
[`acceptance/fixtures/v13_mailbox.json`](../../../acceptance/fixtures/v13_mailbox.json),
[`product/evidence/v13_mailbox.json`](../../../product/evidence/v13_mailbox.json)
y en la aplicación de
[`internal/application/mailbox.go`](../../../internal/application/mailbox.go).

### Envolvente

Una envolvente causal liga, como mínimo:

```text
proyecto
Goal
generación del plan
WorkItem padre
WorkItem hijo
emisor: principal + WorkItem + ejecución
destinatario: principal + WorkItem + ejecución
resumen compacto
referencias de artefactos
huella criptográfica del contenido
instante de admisión
```

Una referencia de ejecución proporcionada libremente por el cliente no basta.
La composición debe haber enlazado previamente principal, proyecto y ejecución
con autoridad verificable.

### Estados

```text
admitted
   │ reclamar con arrendamiento y cercado
   ▼
claimed
   │ entrega durable
   ▼
delivered
   │ consumo por destinatario exacto
   ▼
consumed
   ├── acknowledged
   └── blocked

fallo definitivo del destinatario exacto ──► retired
```

Los nombres aparecen en inglés porque son valores máquina del contrato. Su
significado es:

- `admitted`: admitido y persistido;
- `claimed`: reclamado con un arrendamiento válido;
- `delivered`: puesto a disposición del destinatario;
- `consumed`: leído por ese destinatario;
- `acknowledged`: aceptado como resolución suficiente;
- `blocked`: reconocido como bloqueo causal;
- `retired`: retirado porque el destinatario exacto ya no puede resolverlo.

No se salta directamente de admisión a acuse. Admisión, entrega, consumo y
resolución tienen recibos distintos.

### Propiedades obligatorias

1. **Al menos una entrega.** La cola de salida reintenta sin perder el mensaje.
2. **Idempotencia exacta.** Repetir la misma operación devuelve el mismo hecho.
3. **Cercado monotónico.** Un reclamante obsoleto no puede confirmar entrega.
4. **Ámbito exacto.** Proyecto, `Goal`, generación, hijo, padre y ejecuciones
   deben coincidir.
5. **Destinatario exacto.** La ejecución sucesora no acusa un mensaje dirigido
   a la ejecución anterior.
6. **Sin resolución inventada.** Un fallo no genera un acuse sintético.
7. **Sin redirección silenciosa.** Un mensaje retirado no cambia de
   destinatario.
8. **Artefactos por referencia.** El buzón transporta resúmenes y referencias,
   no volcados grandes.

### Barrera de cierre del padre

La mera relación `Parent` no activa siempre una barrera. La obligación
contractual es explícita mediante `HandoffRequired=true`; este indicador es
inválido si no existe padre.

Cuando la barrera está activa:

```text
hijo termina con éxito
        │
        ▼
publica resolución causal
        │
        ▼
padre exacto la consume
        │
        ├── acknowledged ──► el padre puede superar esa dependencia
        └── blocked ───────► el padre conserva un bloqueo causal
```

Un hijo fallido u omitido ya bloquea causalmente. No se fabrica una entrega
exitosa para cerrar el hueco.

## 7.6 Qué falta para mensajería y relevo de sesión

`ORC-15` sigue declarado. El buzón de V13 no debe estirarse mediante
interpretaciones hasta convertirlo en:

- chat entre pares;
- difusión a grupos;
- bandeja genérica por usuario;
- historial de conversación;
- reasignación automática a una sesión nueva;
- protocolo completo de relevo entre Directores;
- memoria semántica;
- canal de órdenes fuera del registro de comandos.

### Relevo correcto entre sesiones

Una nueva sesión no suplanta a la anterior. Debe recibir un paquete durable y
quedar enlazada por una operación explícita:

```text
sesión A
  ├── hechos ya persistidos
  ├── artefactos con huella criptográfica
  └── solicitud de relevo
           │
           ▼
decisión autorizada de relevo
  ├── cierra o revoca autoridad de A
  ├── crea la ejecución B
  ├── liga B al mismo sujeto permitido
  └── publica un paquete compacto para B
           │
           ▼
sesión B reanuda desde estado durable
```

No se debe editar el destinatario de mensajes antiguos. Si B necesita una
resolución, se crea una nueva entrega causal que referencia la original y la
decisión de relevo.

### Contrato mínimo futuro de relevo

Antes de implementar `ORC-15` hay que fijar:

- quién puede solicitar y quién puede aprobar el relevo;
- qué arrendamientos se revocan o transfieren;
- cómo se conserva la generación;
- qué mensajes quedan consumibles por A;
- qué información recibe B;
- cómo se impiden dobles consumidores;
- qué ocurre con herramientas o efectos externos en vuelo;
- qué recibos prueban el cambio;
- cómo se reintenta sin duplicar una sesión;
- cómo se diagnostica una sesión huérfana.

## 7.7 Paquetes de contexto: diseño para `ORC-19`

Esta sección es diseño orientativo, no un esquema productivo acreditado.
`ORC-19` debe introducir un gestor de contexto detrás de un puerto, sin crear
otro estado de negocio.

Un paquete compacto podría representar:

```text
paquete de contexto (`ContextPackage`)
  sujeto (`Subject`)
    referencia de proyecto (`ProjectRef`)
    referencia de objetivo (`GoalRef`)
    generación del plan (`PlanGeneration`)
    referencia de unidad de trabajo (`WorkItemRef`)
    referencia de ejecución opcional (`ExecutionRef`)
  objetivo (`Objective`)
  restricciones (`Constraints[]`)
  decisiones (`Decisions[]`)
  preguntas abiertas (`OpenQuestions[]`)
  referencias de artefactos (`ArtifactRefs[]`)
    referencia (`Ref`)
    huella criptográfica (`Digest`)
    tipo de medio (`MediaType`)
    propósito (`Purpose`)
  referencias de evidencias (`EvidenceRefs[]`)
  referencias de mensajes causales (`CausalMessageRefs[]`)
  presupuesto (`Budget`)
    máximo de bytes (`MaxBytes`)
    máximo de unidades de texto (`MaxTokens`)
    máximo de referencias de artefacto (`MaxArtifactRefs`)
  alcance (`Scope`)
    rutas permitidas (`AllowedPaths[]`)
    herramientas permitidas (`AllowedTools[]`)
    etiquetas de seguridad (`SecurityLabels[]`)
  linaje (`Lineage`)
    referencia del paquete padre (`ParentPackageRef`)
    revisión de origen (`SourceRevision`)
  huella del contenido (`ContentDigest`)
```

Los nombres son didácticos. El contrato definitivo debe versionarse, tener
esquema estricto y pasar por la autoridad de la hoja de ruta.

### Qué entra

- objetivo y aceptación;
- restricciones que cambian una decisión;
- hechos durables relevantes;
- decisiones y su causa;
- preguntas pendientes;
- referencias de código, pruebas, informes y artefactos;
- presupuesto restante;
- identidad y ámbito necesarios;
- enlace al paquete anterior.

### Qué no entra

- secretos;
- variables de entorno completas;
- credenciales;
- repositorios o registros enteros pegados al texto;
- archivos grandes que ya tienen referencia;
- conversación ornamental;
- deducciones presentadas como hechos;
- permisos implícitos;
- texto humano como sustituto de un recibo.

### Recuperación progresiva

La recuperación empieza por mecanismos baratos y verificables:

1. referencias exactas;
2. metadatos y huellas criptográficas;
3. `rg` sobre texto;
4. búsqueda de texto completo o BM25;
5. solo después, si un banco de pruebas lo justifica, vectores, reordenación o
   representaciones semánticas.

La introducción de una base vectorial no convierte sus resultados en verdad.
Cada fragmento recuperado conserva procedencia, huella criptográfica, proyecto,
permisos y vigencia.

### Puerto y adaptadores

El núcleo podría necesitar un puerto con operaciones conceptuales como:

```text
construir contexto(sujeto, política) -> referencia de paquete y huella
resolver referencias(referencia de paquete, límites) -> recursos acotados
registrar consumo(ejecución, referencia de paquete, huella) -> recibo
```

El adaptador puede usar archivos, una base documental o índices sustituibles.
No puede:

- cerrar el `Goal`;
- cambiar la generación;
- asignar trabajo;
- inventar permisos;
- guardar secretos por comodidad;
- resolver conflictos de autoridad.

## 7.8 Autor y revisiones independientes

V18 acredita la cadena de autor, revisión primaria y revisión adversarial. El
contrato detallado se conserva en
[`analisis_y_contrato_v18_reviews_independientes.md`](../analisis_y_contrato_v18_reviews_independientes.md).

### Sujeto inmutable de revisión

Las tres ejecuciones deben hablar del mismo candidato. El sujeto incluye, entre
otros:

- `Goal` y `WorkItem`;
- ejecución e intento del autor;
- generaciones de plan, `WorkItem` y `AppSpec`;
- `SpecHash`;
- recibo y referencia externa del lanzamiento del autor;
- huella criptográfica del enlace al espacio de trabajo;
- referencia y huella criptográfica del conjunto de cambios;
- identificador del árbol;
- huella criptográfica de la diferencia (`diff`);
- huella criptográfica del conjunto de escritura;
- huella criptográfica de las pruebas exigidas;
- referencias, huellas criptográficas y política de las atestaciones de prueba.

Si cambia cualquiera de los elementos que definen el candidato, la aprobación
anterior queda obsoleta. No se «actualiza» su texto para que parezca vigente.

### Independencia física y causal

Se exigen tres lanzamientos reales:

```text
A = autor
P = revisor primario
D = revisor adversarial

ExecutionRef(A) ≠ ExecutionRef(P) ≠ ExecutionRef(D)
LaunchReceipt(A) ≠ LaunchReceipt(P) ≠ LaunchReceipt(D)
proceso externo(A) ≠ proceso externo(P) ≠ proceso externo(D)
```

Pueden usar el mismo proveedor o modelo. La independencia exigida no es una
marca comercial distinta, sino ejecuciones, recibos y procesos distintos,
todos enlazados al mismo sujeto.

### Evaluación estructurada

Una evaluación contiene:

```text
rol: primary | adversarial
veredicto: approve | changes_requested
ReviewSubjectDigest
LaunchReceiptRef
hallazgos estructurados
referencias y huellas criptográficas de evidencia
identidad, proyecto, generación e intento
clave de idempotencia
```

Los valores literales son códigos máquina. «Muy bien», `PASS`, un acuse del
agente o texto libre no equivalen a `approve`.

El estado derivado de la puerta de revisión es uno de:

```text
missing
changes_requested
approved
stale
```

Solo dos aprobaciones válidas, una primaria y otra adversarial, producen
`approved`. Dos evaluaciones del mismo rol son un conflicto, no una mayoría.

### Revisión primaria

Debe comprobar, como mínimo:

- que el cambio cumple el contrato;
- que el conjunto de escritura es el autorizado;
- que las pruebas exigidas corresponden al sujeto;
- que no hay regresiones funcionales;
- que la arquitectura conserva sus fronteras;
- que los errores y estados son coherentes;
- que la documentación relevante no falsea el alcance.

### Revisión adversarial

Debe intentar invalidar el candidato:

- entradas maliciosas o límites;
- revisión, generación o recibo obsoletos;
- repetición e idempotencia;
- fallos parciales;
- pérdida de proceso y recuperación;
- cruces de proyecto o identidad;
- fugas de secretos;
- condiciones de carrera;
- artefactos manipulados;
- pruebas verdes que no prueban la composición prometida.

No es una segunda lectura amable. Su encargo debe incluir los invariantes y
negativos relevantes.

## 7.9 Retrabajo sin destruir evidencia

Si una revisión pide cambios:

1. se conserva el candidato rechazado;
2. se conserva la evaluación y sus hallazgos;
3. el Director recibe la causa estructurada;
4. propone un nuevo `WorkItem` o intento con `ReworkOf`;
5. el nuevo cambio referencia al anterior;
6. se calcula un nuevo sujeto;
7. se repite la cadena A/P/D completa.

No se reciclan aprobaciones de un árbol anterior. Tampoco se borra el fallo para
«limpiar» el historial.

```text
candidato C1 ──changes_requested──► retrabajo C2
      │                                  │
      └── evidencia conservada           └── nuevo sujeto + nuevas revisiones
```

## 7.10 Qué es exactamente el Consejo

El Consejo es una decisión de gobierno posterior a una revisión V18 válida.
No es:

- una plantilla fija de tres agentes siempre encendidos;
- un grupo de chat;
- tres votos sobre una idea sin candidato;
- un sustituto del Director;
- una tercera revisión de código;
- un integrador;
- el escritor del ciclo de vida.

Es una ronda acotada sobre un candidato exacto ya revisado. Sus tres funciones
son:

1. **proponente:** argumenta por qué el candidato debe avanzar;
2. **crítico:** busca razones de gobierno, riesgo o suficiencia para no
   aceptarlo;
3. **árbitro:** pondera el expediente completo y emite su papeleta.

Las tres son ejecuciones acreditadas y distintas entre sí, y además distintas
de autor, revisor primario y revisor adversarial. Por tanto, una cadena completa
puede requerir seis lanzamientos reales, sin contar al Director:

```text
autor
revisor primario
revisor adversarial
proponente del Consejo
crítico del Consejo
árbitro del Consejo
```

La cantidad «tres» describe funciones dentro de una ronda del Consejo. No
limita el número total de agentes que Orquesta puede levantar.

## 7.11 Sujeto y políticas del Consejo

El sujeto del Consejo liga:

- `ProjectRef`;
- huella criptográfica del sujeto de revisión V18;
- huella criptográfica de la puerta de revisión aprobada;
- política inmutable del Consejo;
- referencias causales necesarias;
- generación e identidad de los lanzamientos;
- huella criptográfica conjunta.

El contrato de V19 se encuentra en
[`analisis_y_contrato_v19_consejo.md`](../analisis_y_contrato_v19_consejo.md).

### Políticas

| Política máquina | Significado |
|---|---|
| `auto` | abre una única ronda al obtener una puerta V18 válida |
| `required` | exige que un Director con arrendamiento y cercado vigentes abra la ronda |
| `skip_by_operator` | un humano autorizado registra una omisión explícita antes de que exista ronda |

No hay degradación silenciosa. Si `required` no puede abrirse, no se convierte
en `auto` ni se omite. Si falta capacidad, la operación espera, reintenta o se
bloquea de forma diagnosticable según política; no inventa votos.

La omisión del operador registra principal, permiso, razón, instante UTC,
`SpecHash`, idempotencia y sujeto. Produce cero lanzamientos de Consejo y no
elimina la obligación de revisión V18.

## 7.12 Papeletas y decisión

Cada ejecución entrega una contribución estructurada y una papeleta:

```text
accept
reject
abstain
security_veto
```

Son códigos máquina:

- `accept`: aceptar;
- `reject`: rechazar;
- `abstain`: abstenerse;
- `security_veto`: veto de seguridad.

Se esperan las tres papeletas incluso si aparece un veto temprano. Así se
conserva el desacuerdo y se evita que el orden de llegada altere la evidencia.

La decisión derivada es:

```text
si existe security_veto          -> blocked_security
si hay al menos dos accept       -> accepted
si hay al menos dos reject       -> rejected
en cualquier otro caso           -> no_consensus
```

`blocked_security` significa bloqueo de seguridad; `accepted`, aceptado;
`rejected`, rechazado; y `no_consensus`, sin consenso.

Una decisión rechazada, sin consenso o bloqueada por seguridad conserva todo el
trabajo y vuelve al Director como causa de replanteamiento. No borra el árbol,
no hace descarte automático y no convierte una abstención en aprobación.

## 7.13 De la intención a la integración

La secuencia completa es:

```text
1. Director propone un plan
2. motor valida y persiste el DAG
3. el planificador reclama un WorkItem
4. autor produce candidato y pruebas
5. revisión primaria evalúa el sujeto
6. revisión adversarial evalúa el mismo sujeto
7. puerta de revisión queda approved
8. política decide si abre Consejo, lo exige o permite omisión autorizada
9. tres ejecuciones del Consejo contribuyen y votan
10. decisión del Consejo queda derivada
11. IntegrateChange revalida sujeto, huellas criptográficas, permisos y decisiones
12. solo la aplicación persiste el efecto de integración
```

Ni la revisión ni el Consejo ejecutan el paso 11 por su cuenta. Una operación
explícita de integración debe revalidar que no se haya alterado el árbol, diff,
pruebas, política o sujeto.

## 7.14 Presupuestos y capacidad

Cada trabajo colaborativo necesita límites independientes:

- duración;
- coste monetario;
- consumo de modelo;
- bytes y unidades de contexto;
- cantidad y tamaño de artefactos;
- intentos y reintentos;
- número máximo de ejecuciones simultáneas;
- herramientas y efectos permitidos.

El agotamiento de presupuesto no lleva automáticamente al `Goal` a un estado
fallido.
El motor registra la causa y aplica la política: esperar aprobación, reducir
alcance, replanificar, cambiar un adaptador permitido o bloquear el `WorkItem`.

La falta temporal de una plaza de agente tampoco justifica cerrar el trabajo.
El planificador conserva el elemento listo y la política elástica decide cuándo
crear capacidad.

### Compresión sin pérdida de autoridad

Cuando un paquete supera el límite:

1. se mantienen sujeto, restricciones, decisiones y preguntas;
2. se sustituyen contenidos grandes por referencias con huella criptográfica;
3. se resumen hallazgos sin eliminar su evidencia fuente;
4. se registra la política de reducción;
5. se calcula una huella criptográfica nueva del paquete;
6. el consumidor puede solicitar recursos concretos dentro de su permiso.

Nunca se elimina identidad, generación o causalidad para ahorrar texto.

## 7.15 Fallos y respuesta correcta

| Fallo | Respuesta |
|---|---|
| agente no arranca | conservar el `WorkItem`, registrar intento y reintentar según política |
| agente queda colgado | diagnosticarlo, solicitar parada cooperativa, ejecutar una parada exacta autorizada si no responde, verificar el efecto y sellar sus artefactos |
| entrega repetida | responder idempotentemente si el contenido y ámbito son exactos |
| contenido distinto con misma clave | conflicto |
| destinatario desaparece | retirar el mensaje; no reasignarlo en silencio |
| revisor entrega solo texto | rechazar por evaluación no estructurada |
| dos roles comparten ejecución | rechazar independencia |
| el árbol cambia tras aprobar | marcar puerta obsoleta |
| falta un voto | ronda incompleta; no decidir por mayoría anticipada |
| aparece veto | esperar las tres papeletas y derivar bloqueo de seguridad |
| sesión nueva reclama identidad antigua | rechazar y exigir relevo explícito |
| referencia cruza proyecto | ocultar o rechazar sin filtrar existencia |
| paquete contiene secreto | rechazar, registrar incidencia y rotar si hubo exposición |
| índice de contexto no responde | degradar a referencias o búsqueda permitida; no cambiar el ciclo de vida |

La expiración de un arrendamiento invalida escrituras posteriores mediante la
cerca, pero no mata un proceso. La retirada física es un efecto distinto:

```text
diagnóstico con identidad de proceso y ejecución
  -> solicitud cooperativa
  -> observación durante el tiempo límite
  -> parada exacta autorizada, si sigue vivo
  -> verificación de ausencia o estado detenido
  -> sellado e inventario para revisión
```

Nunca se envía una señal a un proceso deducido solo por nombre, perfil o texto
de registro. La orden liga proveedor, referencia externa, ejecución, intento,
proyecto y recibo de lanzamiento.

## 7.16 Diagnóstico operativo

Ante un atasco de colaboración, no se empieza reiniciando procesos. Se sigue la
cadena causal:

1. identificar `ProjectRef`, `GoalRef`, generación y `WorkItemRef`;
2. comprobar si el `WorkItem` está realmente listo según el DAG;
3. comprobar arrendamiento, cercado, intento y ejecución;
4. buscar el recibo de lanzamiento;
5. revisar la envolvente de buzón y su destinatario exacto;
6. verificar si existe entrega, consumo y resolución;
7. calcular el sujeto de revisión y comparar huellas criptográficas;
8. comprobar las dos funciones de revisión y sus lanzamientos;
9. identificar política y estado del Consejo;
10. verificar las tres papeletas y la decisión derivada;
11. comprobar presupuesto, capacidad y esperas;
12. revisar la cola de salida y sus reintentos;
13. decidir si procede reintento, relevo, retrabajo o replanteamiento.

### Preguntas que detectan un falso avance

- ¿Hay un recibo o solo lo afirma el agente?
- ¿El lanzamiento fue real y distinto?
- ¿El autor y los dos revisores miran exactamente el mismo sujeto?
- ¿El mensaje llegó a la ejecución autorizada?
- ¿El Consejo recibió las tres papeletas?
- ¿La integración revalidó las huellas criptográficas?
- ¿El nuevo Director reconstruyó desde estado durable?
- ¿La prueba usa la composición productiva prometida?

## 7.17 Pruebas mínimas

### Buzón

- admisión, reclamación, entrega, consumo y ambas resoluciones;
- repetición exacta;
- misma clave con contenido distinto;
- cercado obsoleto y arrendamiento expirado;
- principal, proyecto, generación, padre, hijo o ejecución incorrectos;
- sucesor intentando acusar el mensaje del predecesor;
- retirada por fallo del destinatario;
- cierre de padre con y sin barrera;
- reinicio entre cada transición;
- referencias de artefacto manipuladas.

### Revisión

- tres lanzamientos reales y distintos;
- autor intentando revisarse;
- dos revisores con la misma ejecución o recibo;
- sujeto diferente por árbol, diff, pruebas, generación o `SpecHash`;
- dos aprobaciones válidas;
- cada función solicitando cambios;
- evaluación duplicada;
- aprobación textual no estructurada;
- puerta obsoleta después de mutar el sujeto;
- retrabajo que conserva `ReworkOf`;
- integración sin puerta o con puerta antigua.

### Consejo

- políticas `auto`, `required` y omisión autorizada;
- omisión después de abrir ronda;
- proponente, crítico o árbitro duplicado;
- lanzamiento reutilizado desde V18;
- papeletas fuera de ámbito;
- repetición exacta y conflicto por manipulación;
- todas las combinaciones de decisión;
- espera de tres papeletas pese al veto;
- pérdida y recuperación tras cada contribución;
- integración con decisión no aceptada.

### Contexto y relevo futuros

Cuando se implementen `ORC-15` y `ORC-19`, deberán añadirse:

- límite de bytes, unidades y referencias;
- secreto y contenido no permitido;
- ámbito cruzado;
- referencia ausente o huella criptográfica incorrecta;
- reducción determinista;
- caída del índice con recuperación por referencias;
- relevo repetido;
- doble consumidor;
- autoridad antigua después del relevo;
- efecto externo en vuelo;
- reanudación tras reinicio.

## 7.18 Orden recomendado de implementación desde cero

1. Implementar `Goal`, DAG, revisión esperada e idempotencia.
2. Separar arrendamiento de Director y de ejecución.
3. Implementar un planificador único y lanzamientos acreditados.
4. Añadir artefactos por referencia y cola de salida.
5. Implementar buzón hijo exacto y su barrera de cierre.
6. Añadir sujeto inmutable y dos revisiones independientes.
7. Añadir retrabajo causal.
8. Añadir políticas, lanzamientos y decisión del Consejo.
9. Acreditar la integración explícita del sujeto exacto.
10. Diseñar y acreditar mensajería genérica y relevo de sesión.
11. Diseñar y acreditar el gestor de contexto con bancos de prueba.

No conviene empezar por un chat de agentes ni por una memoria vectorial. Sin
identidad, sujeto, recibos, DAG y cercado, esas superficies solo ocultan la
falta de autoridad.

## 7.19 Criterio de cierre

La colaboración está lista cuando una caída, un desacuerdo o un cambio de
agente no obliga a confiar en conversación perdida:

- el estado puede reconstruirse;
- cada participante tiene identidad y permiso;
- cada mensaje tiene causa y destinatario;
- cada aprobación liga un sujeto inmutable;
- el desacuerdo se conserva;
- la capacidad puede crecer sin cambiar el modelo;
- un agente retirado deja evidencia suficiente;
- solo la aplicación muta el ciclo de vida;
- ninguna superficie auxiliar inventa éxito.

Para Orquesta, el corte honesto actual es doble: el buzón hijo, la revisión
independiente y el Consejo cuentan con capacidades acreditadas; la mensajería
genérica, el relevo completo y el gestor de contexto todavía requieren
implementación, composición, ejercicio y evidencia de su propia revisión.
