FROM golang:1.24 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o migration cmd/migration/main.go

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/migration .
COPY --from=builder /app/scripts/schema.sql ./scripts/

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

CMD ["./migration", "-mode", "worker"]
