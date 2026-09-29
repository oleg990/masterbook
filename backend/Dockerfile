FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o masterbook-server ./cmd/server
RUN go build -o masterbook-migrate ./cmd/migrate

FROM alpine:3.22

WORKDIR /app

COPY --from=builder /app/masterbook-server ./masterbook-server
COPY --from=builder /app/masterbook-migrate ./masterbook-migrate
COPY --from=builder /app/migrations ./migrations

EXPOSE 8080

CMD ["./masterbook-server"]
