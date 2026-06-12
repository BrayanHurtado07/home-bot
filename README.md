# 🏠 Roomie Discipline Bot (Go + CockroachDB + Groq)

Un backend de bot de Telegram multi-tenant potente y optimizado escrito en Go. Permite la convivencia inteligente entre roomies (gestión de alquileres, limpieza y turnos de cocina) combinándolo con disciplina individual (hábitos, metas de ejercicio, alimentación y búsqueda de empleo).

Cuenta con un procesamiento inteligente del lenguaje natural a través de **Groq API** optimizado para bajo consumo de tokens.

---

## 🛠️ Stack Tecnológico
- **Lenguaje:** Go (Golang) v1.25+
- **Base de Datos:** CockroachDB / PostgreSQL (mediante el driver `jackc/pgx/v5`)
- **APIs:** Telegram Bot API (Long Polling estructurado) & Groq Cloud API (Llama 3.1 8B)
- **Capa de Configuración:** `joho/godotenv` para entorno local

---

## 📐 Arquitectura del Proyecto (DDD & Clean Architecture)

El proyecto sigue una estructura limpia para asegurar la separación de responsabilidades y evitar cuellos de botella por concurrencia en llamadas de Telegram:

```text
home-bot/
├── cmd/
│   └── bot/
│       └── main.go         # Punto de entrada. Inicialización y graceful shutdown.
├── db/
│   └── migrations/
│       └── 0001_init.sql   # Sentencias de migración de la DB (se auto-ejecutan al iniciar).
├── internal/
│   ├── config/
│   │   └── config.go       # Cargador de variables de entorno (.env).
│   ├── domain/             # Capa del Dominio (Entidades e Interfaces de Repositorios).
│   │   ├── user.go
│   │   ├── payment.go
│   │   ├── task.go
│   │   ├── meal.go
│   │   ├── habit.go
│   │   └── ai.go
│   ├── application/        # Capa de Aplicación (Servicios y Casos de Uso del negocio).
│   │   └── services.go
│   └── infrastructure/     # Capa de Infraestructura (Adaptadores externos).
│       ├── db/
│       │   ├── postgres_repo.go # Repositorios con pgxpool y transacciones seguras.
│       │   └── migration.go     # Ejecutor automático de migraciones embebidas.
│       ├── groq/
│       │   └── groq_client.go   # Cliente HTTP para Groq y resumidor de tokens.
│       └── telegram/
│           └── telegram_bot.go  # Manejador del bot, FSM de estados e inline keyboards.
├── .env.example
├── go.mod
└── go.sum
```

---

## 🤖 Optimización de Tokens de IA (Groq)
Para reducir drásticamente el uso de tokens y optimizar costos:
1. **Lógica Híbrida:** Las operaciones rutinarias (como marcar una tarea como completada, registrar comidas o aprobar un pago) se realizan mediante comandos (`/tareas`, `/pagar`, `/gimnasio`) y **Inline Keyboards** interactivos directamente en Telegram.
2. **Context Window Reducida (Contexto Deslizante):** No enviamos el historial completo de chats en cada petición. El sistema mantiene en la tabla `ai_context` un resumen condensado continuo del historial generado por Groq (limitado a 150 palabras), actualizándolo dinámicamente con cada nueva interacción del usuario.
3. **System Prompts Compactos:** Instrucciones directas de sistema obligan a Groq a responder en un tono estricto, directo y en máximo **2 frases**.

---

## 🚀 Guía de Instalación y Ejecución

### 1. Clonar el repositorio y configurar entorno
Crea tu archivo `.env` en la raíz del proyecto basándote en la plantilla `.env.example`:

```bash
cp .env.example .env
```

Edita `.env` agregando tus credenciales:
```env
TELEGRAM_BOT_TOKEN=tu_token_de_telegram
GROQ_API_KEY=tu_api_key_de_groq
COCKROACH_DB_URL=postgresql://root@localhost:26257/defaultdb?sslmode=disable
```

### 2. Levantar CockroachDB (Localmente con Docker)
Si no dispones de una instancia activa de CockroachDB, puedes arrancar una de prueba en local con el siguiente comando:

```bash
docker run -d --name cockroach -p 26257:26257 -p 8080:8080 cockroachdb/cockroach:v23.1.11 start-single-node --insecure
```

### 3. Compilar y Arrancar el Bot
Las migraciones de la base de datos se ejecutan **automáticamente** en cuanto el backend arranca.

Para compilar y correr:
```bash
go run cmd/bot/main.go
```

Para compilar un binario:
```bash
go build -o bin/bot cmd/bot/main.go
./bin/bot
```

---

## 🌐 Despliegue en Producción

Para desplegar el bot en un entorno de producción (ej. un VPS Linux o plataforma en la nube), sigue estos pasos divididos por responsabilidades:

### 1. Pasos que debes realizar tú (Usuario)
1. **Configurar el Bot en Telegram (BotFather)**:
   - Chatea con `@BotFather` en Telegram y usa el comando `/newbot` para registrar tu bot de producción.
   - Guarda el token generado (`TELEGRAM_BOT_TOKEN`).
2. **Obtener API Key de Groq**:
   - Regístrate en [Groq Cloud Console](https://console.groq.com/) y genera una API key (`GROQ_API_KEY`).
3. **Aprovisionar la Base de Datos**:
   - Crea una base de datos en [CockroachDB Serverless](https://www.cockroachlabs.com/products/cockroachdb-serverless/) o PostgreSQL (como Supabase, Render o RDS).
   - Obtén la cadena de conexión segura (URI). *Nota: En producción es fundamental usar SSL (`sslmode=verify-full` o similar)*.
4. **Configurar Servidor y Variables**:
   - Asegúrate de tener Docker instalado en tu servidor/VPS.
   - Crea un archivo `.env` en el servidor con tus variables reales de producción:
     ```env
     TELEGRAM_BOT_TOKEN=tu_token_de_produccion
     GROQ_API_KEY=tu_api_key_de_groq_produccion
     COCKROACH_DB_URL=tu_conexion_db_con_ssl
     ```

### 2. Pasos de Despliegue (Utilizando Docker)
Hemos preparado la infraestructura de Docker para que el despliegue sea lo más rápido y seguro posible.

* **Construir la imagen de Docker manualmente**:
  ```bash
  docker build -t home-bot:latest .
  ```

* **Desplegar usando Docker Compose (Recomendado)**:
  Para iniciar el bot en segundo plano con políticas de reinicio automático y límites en el tamaño de logs:
  ```bash
  docker compose -f docker-compose.prod.yml up -d
  ```

* **Ver los logs en tiempo real**:
  ```bash
  docker compose -f docker-compose.prod.yml logs -f --tail=100
  ```

* **Detener el bot**:
  ```bash
  docker compose -f docker-compose.prod.yml down
  ```

---

## 📖 Comandos de Telegram Soportados
- `/start` - Inicializa el bot y te registra en la DB.
- `/creargrupo` - Inicia el flujo para crear un nuevo departamento/grupo de roomies.
- `/unirse` - Únete al departamento de tus roomies mediante el ID del grupo.
- `/tareas` - Ver tareas de limpieza pendientes. Incluye botón interactivo `[✅ Completar]`.
- `/creartarea` - Agrega una nueva tarea de limpieza grupal.
- `/pagar` - Registrar un pago de alquiler o servicio (ingresar monto y subir foto de comprobante).
- `/pagospendientes` - Ver reportes de pagos. Si eres Admin, te dará botones inline de `[✅ Aprobar]` y `[❌ Rechazar]`.
- `/cocina` - Ver los turnos de cocina (Desayuno, Almuerzo, Cena) de la semana. Permite asignarte haciendo click sobre los botones inline disponibles.
- `/habitos` - Consultar tus metas personales y progreso.
- `/crearhabito` - Registrar un nuevo hábito (gimnasio, correr, postulaciones a empleo).
- `/cancelar` - Cancela cualquier flujo activo de registro/wizard actual.
