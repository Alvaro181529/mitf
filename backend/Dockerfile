# Multi-stage Dockerfile para el Backend Go (MITFV2)

# Etapa 1: Compilación (Go 1.26+)
FROM golang:latest AS builder

WORKDIR /app

# Cachear dependencias de Go
COPY go.mod go.sum ./
RUN go mod download

# Copiar el código fuente completo
COPY . .

# Compilar binario estático optimizado
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o server .

# Etapa 2: Imagen Final Ligera
FROM alpine:latest AS runner

WORKDIR /app

# Instalar certificados, zona horaria y utilidades esenciales (cliente SSH, curl, bash, python3)
RUN apk add --no-cache ca-certificates tzdata openssh-client curl bash python3

# Copiar binario compilado desde el builder
COPY --from=builder /app/server /app/server

# Copiar configuración predeterminada si existe
COPY .env* ./

# Puerto de escucha del backend
EXPOSE 8080

# Ejecutar el servidor
CMD ["/app/server"]
