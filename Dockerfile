# Multi-stage Dockerfile para Orquesta
# Perfil opcional de despliegue para el plano de control.

# --- Fase de construcción (Build) ---
FROM golang:1.25-alpine AS builder

# Instalamos dependencias básicas para Go/SQLite si hiciera falta (modernc no las requiere normalmente)
RUN apk add --no-cache git gcc musl-dev

WORKDIR /app

# Copiamos primero dependencias para cachear capas
COPY go.mod go.sum ./
RUN go mod download

# Copiamos el resto del código
COPY . .

# Construimos el binario estático
RUN CGO_ENABLED=0 go build -o orquesta-bin main.go

# --- Fase final (Run) ---
FROM alpine:latest

# Añadimos certificados para conexiones HTTPS salientes si fuera necesario (MCP/API)
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copiamos el binario desde la fase anterior
COPY --from=builder /app/orquesta-bin /app/orquesta

# Creamos directorios para persistencia de datos, logs y workspace montado
RUN mkdir -p /app/data /app/logs /app/workspace

# Variables de entorno por defecto
# El backend SQLite de Orquesta lee ORQUESTA_DB como ruta efectiva.
ENV ORQUESTA_DB=/app/data/orquesta.db
ENV ORQUESTA_WORKSPACE_ROOT=/app/workspace

# Exponemos el puerto del panel web / API
EXPOSE 8080

# Punto de entrada por defecto.
# Se deja el comando separado para poder sobreescribirlo desde compose o docker run.
ENTRYPOINT ["/app/orquesta"]
CMD ["serve", "--puerto", "8080"]
