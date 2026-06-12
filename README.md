# 🏠 Roomie Discipline Bot (V2 Production Edition)

Un backend de bot de Telegram multi-tenant potente y optimizado escrito en Go. Permite la convivencia inteligente entre roomies (gestión de alquileres, limpieza y turnos de cocina) y de disciplina individual (hábitos, metas de ejercicio y recordatorios), combinado con herramientas de CRM e inventarios para emprendimientos personales y compartidos (Habitando Home y Pineapple).

Cuenta con procesamiento inteligente de lenguaje natural mediante la **API de Groq** (Llama 3.1 8B) optimizado para el ahorro y compresión continua de tokens.

---

## 🛠️ Stack Tecnológico
- **Lenguaje:** Go (Golang) v1.25+
- **Base de Datos:** PostgreSQL / Neon Serverless (driver `jackc/pgx/v5`)
- **APIs:** Telegram Bot API (Long Polling estructurado) & Groq Cloud API
- **Infraestructura:** Render.com (Web Services con soporte de salud en puerto de red)
- **Monitoreo/Migraciones:** Sistema de auto-migraciones de esquema y inicializador embebido.

---

---

## 📐 Funcionalidades de Producción (V2 Activo)

### 🏠 Convivencia & Coexistencia Grupal
- **/creargrupo y /unirse:** Registro multi-tenant. Genera un identificador único (UUID) para compartir con compañeros de cuarto.
- **/roomies:** Visualiza los miembros actuales del hogar y sus roles asignados (Admin o Roomie).
- **/tareas:** Lista las tareas de limpieza del hogar con botones inline (`✅ Completada`, `🗑️ Eliminar`).
- **/creartarea:** Creador interactivo secuencial (Descripción -> Asignado -> Fecha de Vencimiento DD/MM).
- **/pagar:** Sube comprobantes de transferencias bancarias de alquiler/servicios como imagen.
- **/pagospendientes:** Muestra las transferencias pendientes. El Administrador del hogar puede usar botones inline para `✅ Aprobar` o `❌ Rechazar` el pago.
- **/cocina:** Menú consolidado interactivo y silencioso (1 solo mensaje de chat) para visualizar el rol semanal y asignarse turnos sin generar notificaciones ruidosas.
- **/asignarcocina:** Permite al Administrador asignar de forma interactiva turnos de cocina a roomies específicos.
- **/dashboard:** Muestra un resumen ejecutivo consolidado del estado actual del hogar (porcentaje de avance de tareas de limpieza, cantidad de cobros pendientes y los encargados de la cocina de hoy).

### 🏋️‍♂️ Disciplina & Hábitos Personales
- **/habitos:** Lista tus hábitos personales, rachas consecutivas (streak) y progreso. Incluye botón interactivo `🔥 Completar Hoy` (que cambia a `✅ Completado Hoy` para evitar duplicados en la base de datos) y `🗑️ Borrar`.
- **/crearhabito:** Registra un hábito diario o para días específicos con alerta horaria local (HH:MM) configurada en tu zona horaria.

### 💼 CRM de Negocios & Catálogo
- **/negocios:** Panel de control de emprendimientos del usuario. Permite registrar múltiples tiendas (como "Habitando Home" de muebles/cojines o "Pineapple" de tecnología) y co-administrarlas.
- **Compartición de Tiendas:** Permite generar un código UUID de acceso desde el botón `👥 Compartir Acceso` del panel. Un socio o colaborador puede usar `/unirstienda [ID_TIENDA]` para co-administrar productos y pedidos.
- **Venta de Catálogo:** Registra el producto vendido del inventario restando automáticamente las unidades disponibles.
- **Pedido Personalizado:** Flujo para registrar clientes con adelanto inicial depositado, saldo pendiente de cobro automático, dirección y costo de envío.
- **Control de Estados:** Actualiza el estado del pedido a `Pendiente de Envío`, `Enviado`, `Entregado` y registra el cobro final del saldo restante.

### 💵 Cobros Interactivos `/asignarpago`
- Permite al Administrador del hogar seleccionar de forma interactiva un roomie mediante botones inline, e ingresar el monto a cobrar. Esto genera automáticamente una deuda en estado pendiente en el registro del roomie para que suba su comprobante con `/pagar`.

### 🔀 Autocompletado de Comandos
- Al escribir `/` en Telegram, se despliega automáticamente el menú nativo de comandos con sus respectivas descripciones estructuradas de forma limpia.

### 🔇 UX Limpia y Formato Corporativo
- Reducción de emojis redundantes, respuestas visuales silenciosas (edición de mensajes existentes inline en lugar de enviar nuevas burbujas de notificación) y diseño estructurado en Markdown para máxima legibilidad.

