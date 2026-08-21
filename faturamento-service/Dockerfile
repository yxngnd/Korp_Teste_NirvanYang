FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /faturamento-service ./cmd/api

FROM alpine:3.19

WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY --from=builder /faturamento-service /app/faturamento-service
COPY --from=builder /app/migrations /app/migrations

EXPOSE 8082

CMD ["/app/faturamento-service"]
