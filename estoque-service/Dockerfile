# Etapa 1: build — usa a imagem completa do Go só para compilar
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copia primeiro só os arquivos de dependência para aproveitar cache do Docker:
# se go.mod/go.sum não mudarem, a etapa de download não é refeita a cada build.
COPY go.mod go.sum* ./
RUN go mod download

COPY . .

# CGO_ENABLED=0 gera um binário estático, necessário para rodar na imagem
# alpine final (sem libc glibc completa).
RUN CGO_ENABLED=0 GOOS=linux go build -o /estoque-service ./cmd/api

# Etapa 2: imagem final — só o binário compilado, bem menor que a imagem de build
FROM alpine:3.19

WORKDIR /app

# Certificados TLS, caso o serviço precise futuramente chamar HTTPS externo.
RUN apk add --no-cache ca-certificates

COPY --from=builder /estoque-service /app/estoque-service
COPY --from=builder /app/migrations /app/migrations

EXPOSE 8081

CMD ["/app/estoque-service"]
